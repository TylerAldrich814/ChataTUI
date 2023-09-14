package router

import (
	"chatatui_backend/db"
	"chatatui_backend/token"
	"chatatui_backend/ws"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

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
  // Routes map[RouteURL]func(http.ResponseWriter, *http.Request)
  DbChatrooms   *db.Chatrooms
  DbUsers       *db.CtuiUsers
  // When a user is Logged in, generate a unique UID for that user, and store
  // UserUID and TokenUID in Tokens as a key/value pair when the user
  // logs off, remove the key/value pair.
  // userTokens  map[Token]UserUID
  userTokens    map[UserUID]Token
  liveChatrooms sync.Map
}

func( router *Router )StartRouter() {
  r := mux.NewRouter()

  r.HandleFunc("/", router.Home)
  r.HandleFunc("/User/Signup", router.UserSignup).Methods("POST")
  r.HandleFunc("/User/Signin", router.UserSignIn).Methods("POST")

  s := r.PathPrefix("/").Subrouter()

  s.Use(router.authenticationHandler)

  // s.HandleFunc("/", http.HandleFunc(router.Home))
  s.HandleFunc("/chatrooms", router.ListPublicChatrooms).Methods("GET");
  s.HandleFunc("/chatrooms/{room_id}", router.GetChatroomMeta).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/messages", router.GetChatroomMessages).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/join", router.BecomeChatroomMember).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/ws", router.JoinChatroom)
  // s.HandleFunc("/User/{user_id}/SignOut", router.UserSignOut)
  http.Handle("/", r)
  if err := http.ListenAndServe(PORT, nil); err != nil {
    log.Fatalf("FATAL: ListenAndServe Error: %s", err.Error())
  }
}

// --------------------- Router Helper Funcs ----------------------
func( router *Router)GetChatroom(roomID string)( *db.Chatroom, error ){
  room, err := router.DbChatrooms.GetChatroom(roomID)
  if err != nil {
    log.Printf(" --> ERROR: Failed to retrieve Chatroom \"%s\": %s \n", roomID, err.Error())
    return nil, err
  }

  return room,nil
}
func( router *Router)GetUser(userID string)( *db.User, error ){
  for uid,user := range router.DbUsers.Users {
    if uid.String() == userID {
      return &user, nil
    }
  }
  log.Printf(" -> GetUser: Failed to find userID \"%s\" \n", userID)
  return nil, fmt.Errorf("UserID \"%s\" doesn't exist", userID)
}
func( router *Router )getToken(userID string)( *token.Token,error ){
  for uid, token := range router.userTokens {
    if uid.String() == userID {
      return &token, nil
    }
  }
  log.Printf(" -> getToken: Failed to find userID \"%s\" \n", userID)
  return nil, fmt.Errorf("UserID \"%s\" doesn't exist", userID)
}

// ---------------------- Router HandleFuncs ----------------------
func( router *Router )authenticationHandler(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
      http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
      return
    }

    splitTokens := strings.Split(authHeader, "Bearer ")
    if len(splitTokens) != 2 {
      http.Error(w, "Malformed Token", http.StatusUnauthorized)
      return
    }

    token := token.Token{
      Token: splitTokens[1],
    }

    // Validate the token
    if err := token.Validate(); err != nil {
      http.Error(w, err.Error(), http.StatusUnauthorized)
      return
    }
    serve := false
    for _, userToken := range router.userTokens {
      if token == userToken {
        serve = true
        break
      }
    }

    if !serve {
      http.Error(w, "Invalid Token", http.StatusUnauthorized)
      return
    }

    next.ServeHTTP(w,r)

    // Validate the token exists.
    // if _, ok := router.userTokens[token]; !ok {
    //   http.Error(w, "Invalid Token", http.StatusUnauthorized)
    //   return
    // }
    // next.ServeHTTP(w,r)
  })
}

// Home Route("/") :: This Route will let a User know 3 things.
//  A.) If they're authorizaed, which will tell the front end to load the Authenticated
//      home Menu(The rest of the app will Also need to pass Authentication)
//  b.) If they're Not Authorized. Which will tell the Front-end to load the Sign in/up menu.
//  c.) If they're Authenticated, BUT their token has expired. Loads the Sing-in Screen.
func( router *Router )Home(
  w http.ResponseWriter,r *http.Request,
){
  // Let the User either Singup or Signup by selecting the correct Option.
  w.Header().Set("Content-Type", "application/json")
  authHeader := r.Header.Get("Authorization")
  if authHeader != "" {
    splitTokens := strings.Split(authHeader, "Bearer ")
    if len(splitTokens) == 2 {
      token := splitTokens[1]

      userToken := Token {
        Token: token,
      }

      // Validate User Token
      if err := userToken.Validate(); err != nil {
        log.Printf(" -> Error: Home: Failed to Validate Token")
        json.NewEncoder(w).Encode(map[string]bool{
          "Authed":false,
        })
        return
      }

      exists := false
      for _, token := range router.userTokens {
        if token == userToken {
          exists = true
          break
        }
      }

      if !exists {
        http.Error(w, "Token validated, but doesn't exist in Databse. User needs to sign in manually", http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]bool{
          "Authed": false,
          "Expired": true,
        })
      }

      expired, err := userToken.TokenIsExpired()
      if err != nil {
        log.Printf(" -> Error: Failed to detect if Token is expired")
        json.NewEncoder(w).Encode(map[string]bool{
          "Authed": false,
        })
        return
      }
      if expired {
        json.NewEncoder(w).Encode(map[string]bool{
          "Authed": false,
          "Expired": true,
        })
      }

      json.NewEncoder(w).Encode(map[string]bool{
        "Authed": true,
        "Expired": false,
      })
      return
    }
  }
  json.NewEncoder(w).Encode(map[string]bool{
    "Authed": false,
  })
}

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
    http.Error(w, "Invalid Request Payload", http.StatusBadRequest)
    return
  }
  defer r.Body.Close()

  var signingUser *db.User
  for _, user := range router.DbUsers.Users {
    if userData.Username == user.Username {
      signingUser = &user
    }
  }

  if err := bcrypt.CompareHashAndPassword(
    signingUser.HashedPassword,
    []byte(userData.Password),
  ); err != nil {
    http.Error(w, "Password is not correct", http.StatusUnauthorized)
    return
  }
  signingUser.IsOnline = true

  // User is now Signin in on the Server. We now create a new Token, replace
  // the old tooken in our DB and send the new Token to the user to be stored
  // client-side
  userID := signingUser.UserID

  newToken, err := token.CreateToken(userID.String())
  if err != nil {
    http.Error(w, "Failed to create new Token", http.StatusInternalServerError)
    return
  }
  router.userTokens[userID] = *newToken

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
  for _, user := range router.DbUsers.Users {
    if user.Username == userSignupData.Username {
      http.Error(w, "Username already taken", http.StatusConflict)
      return
    }
  }

  hashedPassword, err := bcrypt.GenerateFromPassword(
    []byte(userSignupData.Password),
    bcrypt.DefaultCost,
  )
  if err != nil {
    http.Error(w, "Failed while hashing the password", http.StatusInternalServerError)
    return
  }
  uid, err := router.DbUsers.OnSignup(userSignupData.Username, hashedPassword)
  if err != nil {
    http.Error(w, "Failed to Store new User", http.StatusInternalServerError)
    return
  }

  userToken, err := token.CreateToken(uid.String())
  if err != nil {
    http.Error(w, "Failed to Create Token", http.StatusInternalServerError)
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

  userIDQuery := r.URL.Query().Get("user_id")
  userID, err := uuid.Parse(userIDQuery)
  if err != nil {
    return
  }

  user, exists := router.DbUsers.Users[userID]
  if !exists {
    http.Error(w, "Unknown error occurred while joining chatroom", http.StatusInternalServerError)
    return
  }


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
    http.Error(w, "Valid token: Doesn't exisst in Chatroom database.", http.StatusUnauthorized)
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
