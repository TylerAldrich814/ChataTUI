package router

import (
	"chatatui_backend/db"
	"chatatui_backend/token"
	"chatatui_backend/ws"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type (
  RouteURL  = string
  UserUID   = uuid.UUID
  TokenUID  = uuid.UUID
  Token = token.Token
)
const (
  PORT = ":8080"
)

type Router struct {
  database db.ChatatuiDatabase
  wsHub    *ws.Hub
}

func NewRouter(database db.ChatatuiDatabase, wsHub *ws.Hub) *Router {
  router := Router{ database, wsHub }

  return &router
}

func( router *Router )SetupRouter() *mux.Router {
  r := mux.NewRouter()

  r.HandleFunc("/", router.Home).Methods("GET")
  r.HandleFunc("/User/Signin", router.UserSignIn).Methods("POST")
  r.HandleFunc("/User/Signup", router.UserSignup).Methods("POST")

  s := r.PathPrefix("/").Subrouter()

  s.Use(router.authenticationHandler)

  // s.HandleFunc("/home", router.Home)
  s.HandleFunc("/chatrooms", router.ListPublicChatrooms).Methods("GET");
  s.HandleFunc("/chatrooms/{room_id}", router.GetChatroomMeta).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/messages", router.GetChatroomMessages).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/join", router.BecomeChatroomMember).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/ws", router.JoinChatroom)

  return r
}
// --------------------- Router Helper Funcs ----------------------
func( router *Router)GetChatroom(roomID string)( *db.Chatroom, error ){
  // room, err := router.DbChatrooms.GetChatroom(roomID)
  room, err := router.database.GetChatroom(roomID)
  if err != nil {
    return nil, InvalidChatroomError{ cm: roomID }
  }
  return room,nil
}
func( router *Router)getUser(userID string)( *db.User, error ){
  uid, err := uuid.Parse(userID)
  if err != nil {
    return nil, InvalidUserIDError{ id: userID }
  }
  return router.database.GetUserByID(uid)
}

func( router *Router )getToken(r *http.Request)( *token.Token,error ){
  authHeader := r.Header.Get("Authentication")
  if authHeader == "" {
    return nil, MissingAuthHeaderError{ }
  }

  splitTokens := strings.Split(authHeader, "Bearer ")
  if len(splitTokens) != 2 {
    return nil, MalformedTokenError{ }
  }
  token := token.Token{ Token: splitTokens[1] }
  if err := token.Validate(); err != nil {
    return nil, InvalidTokenError{ }
  }
  return  &token, nil
}

// authenticateToken : Checks the validity of the Authentication Token provided
//                     by the user. And checks if said token is expired or not.
func(router *Router)authenticateToken(token *token.Token)( string, error ){
  userID, err := token.GetTokenID()
  if err != nil {
    return "", InvalidTokenIDError{ }
  }
  uid, err := uuid.Parse(userID)
  if err != nil {
    return "", InvalidTokenError{ }
  }
  storedToken, err := router.database.GetUserToken(uid)
  if err != nil {
    return "", InvalidTokenIDError{ }
  }
  if storedToken.Token != token.Token {
    return "", TokenNotFoundError{ }
  }

  expired, _ := token.TokenIsExpired()
  if expired {
    return "", TokenIsExpiredError{ }
  }
  return userID, nil
}

// ---------------------- Router HandleFuncs ----------------------
func( router *Router )authenticationHandler(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    token, err := router.getToken(r)
    if err != nil {
      var redirect_error string
      switch err.(type) {
      case MissingAuthHeaderError:
        redirect_error = "missing_auth_header"
      case MalformedTokenError:
        redirect_error = "malformed_access_token"
      case InvalidTokenError:
        redirect_error = "invalid_access_token"
      }

      http.Redirect(
        w, r,
        fmt.Sprintf("/User/Signin?error=%s",redirect_error),
        http.StatusUnauthorized,
      )
      return
    }

    userID, err := router.authenticateToken(token)
    if err != nil {
      var redirect_error string
      switch err.(type) {
      case InvalidTokenIDError:
        redirect_error = ""
      case InvalidTokenError:
        redirect_error = ""
      case TokenNotFoundError:
        redirect_error = ""
      case TokenIsExpiredError:
        redirect_error = ""
      }

      http.Redirect(
        w, r,
        fmt.Sprintf("/User/Signin?error=%s",redirect_error),
        http.StatusFound,
      )
      return
    }

    // Store UserID in request context for ease of access later on.
    ctx := context.WithValue(r.Context(), "userID", userID)

    next.ServeHTTP(w,r.WithContext(ctx))
  })
}

// Home Route("/") :: This Route will let a User know 3 things.
//  A.) If they're authorizaed, which will tell the front end to load the Authenticated
//      home Menu(The rest of the app will Also need to pass Authentication)
//  b.) If they're Not Authorized, We will tell the Front-end to load the Sign in/up menu.
//  c.) If they're Authenticated, BUT their token has expired. Loads the Sing-in Screen.
func( router *Router )Home(
  w http.ResponseWriter,
  r *http.Request,
){
  // Let the User either Singup or Signup by selecting the correct Option.
  w.Header().Set("Content-Type", "application/json")

  token, err := router.getToken(r)
  if err != nil {
    http.Redirect(
      w, r,
      "/Users/Signup?warn=No_auth_token_detected",
      http.StatusUnauthorized,
    )
    return
  }

  _, err = router.authenticateToken(token)
  if err != nil {
    var redirect_error string
    switch err.(type) {
    case InvalidTokenIDError:
      redirect_error = ""
    case InvalidTokenError:
      redirect_error = ""
    case TokenNotFoundError:
      redirect_error = ""
    case TokenIsExpiredError:
      redirect_error = ""
    }

    http.Redirect(
      w, r,
      fmt.Sprintf("/User/Signin?error=%s",redirect_error),
      http.StatusFound,
    )
    return
  }

  w.WriteHeader(http.StatusOK)
}

// UserSignIn : Route "/Users/Signin" - Expects to receive a 'username' and 'password'
//    from the client. Hashes and verifys if the User exists in the database. If OK,
//    creates a brand new Access Token. Stores it in the Users bucket. Then sends the
//    new Access Token back to the client.
func( router *Router )UserSignIn(
  w http.ResponseWriter,
  r *http.Request,
){
  // Check to see if user submitted a Token within the Auth header
  var userData = struct{
    Username string `json:"username"`
    Password string `json:"password"`
  }{ }

  decoder := json.NewDecoder(r.Body)
  if err := decoder.Decode(&userData); err != nil {
    http.Error(w, "Invalid Request", http.StatusBadRequest)
    return
  }
  defer r.Body.Close()

  signingUser, err := router.database.GetUserbyUsername(userData.Username)
  if err != nil {
    http.Error(w, "Invalid Username", http.StatusUnauthorized)
    return
  }

  if err := bcrypt.CompareHashAndPassword(
    signingUser.HashedPassword,
    []byte(userData.Password),
  ); err != nil {
    http.Error(w, "Password is not correct", http.StatusUnauthorized)
    return
  }
  // signingUser.IsOnline = true


  // User is now Signed in on the Server. We now create a new Token, replace
  // the old tooken in our DB and send the new Token to the user to be stored
  // client-side
  userID := signingUser.UserID

  newToken, err := token.CreateToken(userID.String())
  if err != nil {
    http.Error(w, "Failed to create new Token", http.StatusInternalServerError)
    return
  }
  if err := router.database.UpdateUserToken(userID, newToken); err != nil {
    http.Error(w, "Failed to Store New AccessToken", http.StatusInternalServerError)
    return
  }

  w.Header().Set("Authorization", "Bearer "+newToken.Token)
  w.WriteHeader(http.StatusCreated)
  w.Write([]byte("User successfully signed in."))
}

func( router *Router )UserSignup(
  w http.ResponseWriter,
  r *http.Request,
){
  // w.Header().Set("Authorization", "Bearer "+tokenString)
  var userSignupData = struct{
    Username  string `json:"username"`
    Password  string `json:"password"`
  }{ }

  decoder := json.NewDecoder(r.Body)
  if err := decoder.Decode(&userSignupData); err != nil {
    http.Error(w, "Invalid Request Payload", http.StatusBadRequest)
    return
  }
  defer r.Body.Close()

  // Check if UserName is taken yet.
  // NOTE: THis would NOT be sufficient for a very large Userpool
  exists, err := router.database.DoesUsernameExist(userSignupData.Username)
  if err != nil {
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
  }
  if exists {
    http.Error(w, "Username already taken", http.StatusBadRequest)
    return
  }

  hashedPassword, err := bcrypt.GenerateFromPassword(
    []byte(userSignupData.Password),
    bcrypt.DefaultCost,
  )
  if err != nil {
    http.Error(w, "Failed while hashing the password", http.StatusInternalServerError)
    return
  }
  uid := uuid.New()

  userToken, err := token.CreateToken(uid.String())
  if err != nil {
    http.Error(w, "Failed to Create Token", http.StatusInternalServerError)
    return
  }

  user := db.User{
    UserID: uid,
    Username: userSignupData.Username,
    // AccessToken: *userToken,
    HashedPassword: hashedPassword,
    // Chatrooms: make(map[db.RoomName]db.MemberType),
  }
  err = router.database.SaveUser(user, userToken)
  if err != nil {
    http.Error(w, "Failed to store created User in Database", http.StatusInternalServerError)
    return
  }


  w.Header().Set("Authorization", "Bearer "+userToken.Token)
  w.WriteHeader(http.StatusCreated)
  w.Write([]byte("User successfully Signed up"))
}

func( router *Router )ListPublicChatrooms(
  w http.ResponseWriter,
  r *http.Request,
){

}

func( router *Router )GetChatroomMeta(
  w http.ResponseWriter,
  r *http.Request,
){

}

func( router *Router )BecomeChatroomMember(
  w http.ResponseWriter,
  r *http.Request,
){
  w.Header().Set("Content-Type", "application/json")
  vars := mux.Vars(r)
  roomID, exists := vars["room_id"]
  if !exists {
    return
  }

  // userID, ok := r.Context().Value("userID").(uuid.UUID)
  // if !ok {
  //   http.Redirect(w,r, "/?error=userID_is_missing", http.StatusUnauthorized)
  //   return
  // }

  // userIDQuery := r.URL.Query().Get("user_id")
  // userID, err := uuid.Parse(userIDQuery)
  // if err != nil {
  //   http.Error(w, "userID is invalid", http.StatusBadRequest)
  //   return
  // }

  user, err := router.database.GetUserByID(userID)
  if err != nil {
    http.Error(w, "Failed to find UserID in Database", http.StatusBadRequest)
    return
  }
  // user, exists := router.DbUsers.Users[userID]
  // if !exists {
  //   http.Error(w, "Unknown error occurred while joining chatroom", http.StatusInternalServerError)
  //   return
  // }


  room, err := router.GetChatroom(roomID)
  if err != nil {
    json.NewEncoder(w).Encode(map[string]string{
      "ERROR": fmt.Sprintf(
        "ERROR: Chatroom \"%s\" doesn't exist",
        roomID,
      ),
    })
    return
  }

  // TODO: Handle Private Chatroom Access Logic.
  //
  // isPrivate := vars["private"]
  // ____________________________________________

  token, err := room.AddMember(user.Username)
  if err != nil {
    http.Error(w, "Unknown error occurred while creating ChatroomToken", http.StatusInternalServerError)
    return
  }

  user.Chatrooms[roomID] = db.Member

  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(map[string]string{
    "CHATROOM_TOKEN": token,
  })
}

func( router *Router )JoinChatroom(
  w http.ResponseWriter,
  r *http.Request,
){
  vars := mux.Vars(r)
  roomID := vars["room_id"]

  userQuery := r.URL.Query().Get("user_id")
  userID, err := uuid.Parse(userQuery)
  if err != nil {
    http.Error(w, "Failed to parse userID", http.StatusInternalServerError)
    return
  }

  userName, exist := router.DbUsers.Users[userID]
  if !exist {
    http.Error(w, "Error occurred while trying to find Username", http.StatusInternalServerError)
    return
  }

  room, exist := router.DbChatrooms.Rooms[roomID];
  if !exist {
    http.Error(w, "Chatroom doesn't exist", http.StatusNotFound)
    return
  }

  chatroomTokenReq := r.Header.Get("X-Chatroom-Token")
  token := token.Token{ Token: chatroomTokenReq }

  if err = token.Validate(); err != nil {
    http.Error(w, "Invalid Chatroom Token", http.StatusUnauthorized)
    return
  }

  trueChatroomToken := room.Members[userName.Username]
  if chatroomTokenReq != trueChatroomToken.Token {
    http.Error(w, "Valid token was sent, But doesn't exisst in Chatroom database.", http.StatusUnauthorized)
    return
  }

  //  After Verifying that the User is a Memeber of the chatroom. We can now
  //  upgrade their connection to a Websocket connection and start
  //  sending/receiving data

  hub, ok := router.liveChatrooms.LoadOrStore(roomID, ws.NewHub())
  if !ok {
    go hub.(*ws.Hub).Run()
  }

  ws.ServeWs(hub.(*ws.Hub), w, r)
}


func( router *Router )GetChatroomMessages(
  w http.ResponseWriter,
  r *http.Request,
){
  log.Printf(" -> Chatroom Message Paginate:")

  w.Header().Set("Content-Type", "application/json")
  vars := mux.Vars(r)
  roomID := vars["room_id"]

  room, err := router.GetChatroom(roomID)
  if err != nil {
    json.NewEncoder(w).Encode(map[string]string{
      "ERROR": fmt.Sprintf(
        "ERROR: Chatroom \"%s\" doesn't exist",
        roomID,
      ),
    })
    return
  }

  // Can't remmeber what this was..
  // user, err := router.DbUsers

  // Query and Convert any received paginate parameters. If any fail, fall back
  // onto default parameters.
  pageQuery := r.URL.Query().Get("page")
  page, err := strconv.Atoi(pageQuery)
  if err != nil {
    page = room.CurrentPage
  }
  limitQuery := r.URL.Query().Get("limit")
  limit, err := strconv.Atoi(limitQuery)
  if err != nil {
    limit = db.DefaultPageSize
  }
  beforeIDQuery := r.URL.Query().Get("beforeID")
  beforeID, err := uuid.Parse(beforeIDQuery)
  if err != nil {
    beforeID = uuid.Nil
  }
  afterIdQuery := r.URL.Query().Get("afterID")
  afterID, err := uuid.Parse(afterIdQuery)
  if err != nil {
    afterID = uuid.Nil
  }

  log.Printf("       | Page:     %v", page)
  log.Printf("       | Limit:    %v", limit)
  log.Printf("       | BeforeID: %s", beforeID)
  log.Printf("       | AfterID:  %s", afterID)

  messages, err := room.Paginate(
    page,
    limit,
    beforeID,
    afterID,
  )
  if err != nil {
    log.Printf(" --> ERROR! %s", err.Error())
    json.NewEncoder(w).Encode(map[string]string{
      "ERROR": fmt.Sprintf(
        "ERROR: Failed to Paginate Chatroom Mesages: %s",
        err.Error(),
      ),
    })
    return
  }
  json.NewEncoder(w).Encode(messages)
}
