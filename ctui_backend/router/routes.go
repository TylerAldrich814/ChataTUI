package router

import (
	"chatatui_backend/db"
	"chatatui_backend/token"
	"chatatui_backend/ws"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"unicode"

	// "log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ugorji/go/codec"
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

// type Client struct {
//   hub      *Hub
//   conn     *websocket.Conn
//   messages chan[]byte // For storing the Messages in the Database
//   send     chan[]byte
// }

type Router struct {
  database      db.ChatatuiDatabase
  wsHub         *ws.Hub
  liveChatrooms sync.Map
}

func NewRouter(database db.ChatatuiDatabase, wsHub *ws.Hub) *Router {
  return &Router{ database, wsHub, sync.Map{} }
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
  s.HandleFunc("/chatrooms", router.SaveChatroom).Methods("POST")

  s.HandleFunc("/chatrooms/{room_id}", router.GetChatroomMeta).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}", router.SaveChatroom).Methods("PUT")
  s.HandleFunc("/chatrooms/{room_id}", router.DeleteChatroom).Methods("DELETE")

  s.HandleFunc("/chatrooms/{room_id}/join", router.JoinChatrooom).Methods("GET")

  s.HandleFunc("/chatrooms/{room_id}/messages", router.GetChatroomMessages).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/load", router.OnLoadChatroom).Methods("GET")
  s.HandleFunc("/chatrooms/{room_id}/ws", router.EnterChatroom).Methods("")

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

// RespondWithDataOrError :: Can be used for a OK Response, which requires
//    a json object. Or An Error response, which requires a json object. Or,
//    just simply an HTTP Error response.
func RespondWithDataOrError(
  w http.ResponseWriter,
  r *http.Request,
  data interface{},
  err error,
  status int,
){
  if data != nil {
    w.Header().Set("Content-Type", "application/json")
    var jsonData []byte
    enc := codec.NewEncoderBytes(&jsonData, &db.JSONHandle)
    if encErr := enc.Encode(data); encErr != nil {
      http.Error(
        w,
        fmt.Sprintf("Internal-Error: %s", encErr.Error()),
        http.StatusInternalServerError,
      )
      return
    }

    w.WriteHeader(status)
    w.Write(jsonData)
    return
  }
  if err != nil {
    http.Error(w, err.Error(), status)
    return
  }
  http.Error(w, "Unknown Error", status)
}

func DecodeBodyOrError(
  w http.ResponseWriter,
  r *http.Request,
  data interface{},
) error {
  if data == nil {
    log.Fatalf(" !FATAL: DecodeBodyOrError: \"data\" should never be nil! Fix this")
  }

  decoder := codec.NewDecoder(r.Body, &db.JSONHandle)
  if decoder == nil {
    http.Error(w, "Internal-Error:", http.StatusInternalServerError)
    return db.DecoderError{}
  }

  if err := decoder.Decode(&data); err != nil {
    return db.DecoderError{}
  }
  return nil
}

func writeJSONError(w http.ResponseWriter, message string, statusCode int){
  w.WriteHeader(statusCode)
  errObj := map[string]string{
    "error": message,
  }
  json.NewEncoder(w).Encode(errObj)
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

func extractUserIDfromContext(r *http.Request) uuid.UUID {
  ctxUserID := r.Context().Value("userID")
  userID, ok := ctxUserID.(string)
  if !ok {
    log.Printf("Failed to retreive UserID from Context")
    return uuid.Nil
  }
  userUID, err := uuid.Parse(userID)
  if err != nil {
    log.Printf("Invalid UserID")
    return uuid.Nil
  }

  return userUID
}

func(router *Router)validateRoomMemeber(roomName string, userID uuid.UUID) bool {
  member, err := router.database.GetChatroomMemberStatus(roomName, userID)
  if err != nil {
    fmt.Printf("Failed to Validate User's Membership status")
    return false
  }
  return *member != db.Blocked
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
  resp := struct{
    Authed bool   `codec:"authed"`
    Token  string `codec:"token"`
  }{
    Authed: false,
    Token:  "Missing",
  }

  token, err := router.getToken(r)
  if err != nil {
    RespondWithDataOrError(w, r, resp, err, http.StatusUnauthorized)
    return
  }

  _, err = router.authenticateToken(token)
  if err != nil {
    switch err.(type) {
    case InvalidTokenIDError:
      resp.Token = "ParseError"
      RespondWithDataOrError(w, r, resp, nil, http.StatusBadRequest)
      return
    case InvalidTokenError:
      resp.Token = "Invalid"
      RespondWithDataOrError(w, r, resp, nil, http.StatusUnauthorized)
      return
    case TokenNotFoundError:
      resp.Token = "Missing"
      RespondWithDataOrError(w, r, resp, nil, http.StatusUnauthorized)
      return
    case TokenIsExpiredError:
      resp.Token = "Expired"
      RespondWithDataOrError(w, r, resp, nil, http.StatusUnauthorized)
      return
    }
  }

  resp.Authed = true
  resp.Token = "Valid"

  RespondWithDataOrError(w, r, resp, nil, http.StatusOK)
}

// UserSignIn : Route "/Users/Signin" - Expects to receive a 'username' and 'password'
//    from the client. Hashes and verifys if the User exists in the database. If OK,
//    creates a brand new Access Token. Stores it in the Users bucket. Then sends the
//    new Access Token back to the client.
func( router *Router )UserSignIn(
  w http.ResponseWriter,
  r *http.Request,
){
  w.Header().Set("Content-Type", "application/json")

  var userData = struct{
    Username string `json:"username"`
    Password string `json:"password"`
  }{ }

  DecodeBodyOrError(w, r, userData)
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

  // User is now Verified via username & password
  userID := signingUser.UserID

  if err := router.database.SaveUsersOnlineStatus(signingUser.Username, true); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  newToken, err := token.CreateToken(userID.String())
  if err != nil {
    http.Error(w, "Failed to create new Token", http.StatusInternalServerError)
    return
  }
  if err := router.database.SaveUserToken(userID, newToken); err != nil {
    http.Error(w, "Failed to Store New AccessToken", http.StatusInternalServerError)
    return
  }

  var jsonBytes []byte
  enc := codec.NewEncoderBytes(&jsonBytes, &db.JSONHandle)

  if err := enc.Encode(newToken); err != nil {
    http.Error(w, "Failed to Encode Access Token", http.StatusInternalServerError)
    return
  }

  // w.Header().Set("Authorization", "Bearer "+newToken.Token)
  w.WriteHeader(http.StatusCreated)
  w.Write(jsonBytes)
}

func( router *Router )UserSignup(
  w http.ResponseWriter,
  r *http.Request,
){
  // w.Header().Set("Authorization", "Bearer "+tokenString)
  w.Header().Set("Content-Type", "application/json")
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

  hashedPassword, err := bcrypt.GenerateFromPassword(
    []byte(userSignupData.Password),
    bcrypt.DefaultCost,
  )
  if err != nil {
    http.Error(w, "Failed while hashing the password", http.StatusInternalServerError)
    return
  }
  uid := uuid.New()

  accessToken, err := token.CreateToken(uid.String())
  if err != nil {
    http.Error(w, "Failed to Create Token", http.StatusInternalServerError)
    return
  }

  user := db.User{
    UserID: uid,
    Username: userSignupData.Username,
    HashedPassword: hashedPassword,
  }
  // Save and Store the User and new Access Token.
  err = router.database.SaveUser(user, accessToken)
  if err != nil {
    http.Error(w, "Failed to store created User in Database", http.StatusInternalServerError)
    return
  }

  var jsonBytes []byte
  enc := codec.NewEncoderBytes(&jsonBytes, &db.JSONHandle)

  if err := enc.Encode(accessToken); err != nil {
    http.Error(w, "Failed to Encode Access Token", http.StatusInternalServerError)
    return
  }

  // w.Header().Set("Authorization", "Bearer "+accessToken.Token)
  w.WriteHeader(http.StatusCreated)
  w.Write(jsonBytes)
}

func( router *Router )ListPublicChatrooms(
  w http.ResponseWriter,
  r *http.Request,
){

}

func(router *Router)ValidateChatroom(chatroom *db.Chatroom) error {
  name := chatroom.RoomName

  if len(name) <= 5 || len(name) >= 50{
    return fmt.Errorf("RoomName must be between 5 and 50 characters")
  }
  for _, r := range name {
    if !unicode.IsLetter(r)  && !unicode.IsNumber(r){
      return fmt.Errorf(
        "RoomName can only consist of Letters and Numbers: \"%v\" is not aloud",
        r,
      )
    }
  }
  if _, err := router.database.GetUserByID(chatroom.OwnerID); err != nil {
    return fmt.Errorf("Invalid UserID \"%s\"", chatroom.OwnerID)
  }

  return nil
}

// SaveChatroom: Handles both Chatroom Creation and Chatroom Updates.
func( router *Router )SaveChatroom(
  w http.ResponseWriter,
  r *http.Request,
){
  var chatroom db.Chatroom
  dec := codec.NewDecoder(r.Body, &db.JSONHandle)
  defer r.Body.Close()


  if err := dec.Decode(chatroom); err != nil {
    http.Error(w, "Invalid Chatroom Data", http.StatusBadRequest)
    return
  }

  if err := router.ValidateChatroom(&chatroom); err != nil {
    http.Error(w, err.Error(), http.StatusUnauthorized)
    return
  }

  switch r.Method {
  case http.MethodPost:
    // Handle Chatroom Creation
    if err := router.database.SaveChatroom(&chatroom, false); err != nil {
      http.Error(w, "Failed to save new Chatroom", http.StatusInternalServerError)
    }
  case http.MethodPut:
    // Handle Chatroom Update
    if err := router.database.SaveChatroom(&chatroom, true); err != nil {
      http.Error(w, "Failed to update Chatroom", http.StatusInternalServerError)
    }
  }

  w.WriteHeader(http.StatusOK)
}

// GetChatroomMeta :: /chatrooms/room_id
func( router *Router )GetChatroomMeta(
  w http.ResponseWriter,
  r *http.Request,
){
  w.Header().Set("Content-Type", "application/json")

  vars := mux.Vars(r)
  roomName, exists := vars["room_name"]
  if !exists {
    http.Error(w, "Room name is missing", http.StatusBadRequest)
    return
  }

  chatroom, err := router.database.GetChatroom(roomName)
  if err != nil {
    var redirectErrorURL string
    var httpStatus int

    switch err.(type){
      case db.BucketNotFoundError:
      case db.DecoderError:
        redirectErrorURL = "/?error=internal_server_error"
        httpStatus = http.StatusInternalServerError
      case db.GetDataError:
        redirectErrorURL = "/?error=chatroom_doesnt_exist"
        httpStatus = http.StatusBadRequest
      default:
        redirectErrorURL = "/?error=unknown_error"
        httpStatus = http.StatusInternalServerError
    }

    http.Redirect(
      w,r,
      redirectErrorURL,
      httpStatus,
    )
    return
  }
  var bytes []byte
  enc := codec.NewEncoderBytes(&bytes, &db.JSONHandle)

  if err := enc.Encode(&chatroom); err != nil {
    http.Redirect(w,r, "/?error=internal_error", http.StatusInternalServerError)
    return
  }

  w.WriteHeader(http.StatusOK)
  w.Write(bytes)
}

// DeleteChatroom :: Deactivates Chatroom. Authentication is based on the Token
//     provided by the Authentication Bearer Header
func( router *Router )DeleteChatroom(
  w http.ResponseWriter,
  r *http.Request,
){
  if r.Method != http.MethodDelete {
    http.Error(w, "Invalid HTTP Request", http.StatusMethodNotAllowed)
    return
  }
  vars := mux.Vars(r)
  roomName := vars["room_name"]

  userUID := extractUserIDfromContext(r)
  if userUID == uuid.Nil {
    http.Error(w, "Unable to find UserID in Context", http.StatusUnauthorized)
    return
  }

  if err := router.database.DeactivateChatroom(roomName, userUID); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
  w.WriteHeader(http.StatusOK)
}

// JoinChatrooom :: For becoming a Member of a particular Chatroom
func( router *Router )JoinChatrooom(
  w http.ResponseWriter,
  r *http.Request,
){
  w.Header().Set("Content-Type", "application/json")
  var joinRoomHandle struct {
    RoomName   string `json:"room_name"`
    UserName   string `json:"user_name"`
    Invitation []byte `json:"invitation"`
  }

  decoder := codec.NewDecoder(r.Body, &db.JSONHandle)
  if err := decoder.Decode(&joinRoomHandle); err != nil {
    http.Error(w, "Invalid Join room Handle", http.StatusBadRequest)
    return
  }
  defer r.Body.Close()

  // Calls upon database.EnterChatroom. If Chatroom.Public is set to false. Then
  // Invitation will be required and will fail if missing/invalid.
  if err := router.database.JoinChatroom(
    joinRoomHandle.RoomName,
    joinRoomHandle.UserName,
    joinRoomHandle.Invitation,
  ); err != nil {
    switch err.(type) {
    case db.GetDataError:
      writeJSONError(w, "Malfromed Data Provider", http.StatusBadRequest)
    case db.FailedSecurityCheckError:
      writeJSONError(w, "invitation invalid", http.StatusUnauthorized)
    case db.BucketNotFoundError, db.DecoderError:
      writeJSONError(w, "internal server error", http.StatusInternalServerError)
    default:
      writeJSONError(w, "Unknown Error", http.StatusInternalServerError)
    }
    return
  }
}

func( router *Router )EnterChatroom(
  w http.ResponseWriter,
  r *http.Request,
){
  vars := mux.Vars(r)
  roomName := vars["room_name"]

  userUID := extractUserIDfromContext(r)
  if userUID == uuid.Nil {
    http.Error(w, "Unable to find UserID in Context", http.StatusUnauthorized)
    return
  }

  // room, exist := router.DbChatrooms.Rooms[roomID];
  room, err := router.database.GetChatroom(roomName)
  if err != nil {
    http.Redirect(w,r, "/chatrooms?error=invalid_chatroom", http.StatusNotFound)
    return
  }

  member, err := router.database.GetChatroomMemberStatus(roomName, userUID)
  if err != nil {
    switch err.(type){
    case db.BucketNotFoundError:
      http.Redirect(w,r, "/chatrooms?error=internal_error", http.StatusInternalServerError)
    case db.GetDataError:
      http.Redirect(w,r, "/chatrooms?error=not_a_member", http.StatusUnauthorized)
    }
    return
  }

  if *member == db.Blocked {
    http.Redirect(w,r, "/chatrooms?error=user_is_blocked", http.StatusUnauthorized)
    return
  }

  // Change User's room Status
  // Find way of detecting if a user's ws connection disconnects ?

  hub, ok := router.liveChatrooms.LoadOrStore(room.RoomID, ws.NewHub())
  // If Chatroom is not running. Start an instance in a Goroutine.
  if !ok {
    go hub.(*ws.Hub).Run()
  }

  // If Chatroom is running, Serve the Websocket instance via hub.
  ws.ServeWs(hub.(*ws.Hub), router.database, w, r)
}

func( router *Router)OnLoadChatroom(
  w http.ResponseWriter,
  r *http.Request,
) {
  w.Header().Set("Content-Type", "application/json")

  vars := mux.Vars(r)
  defer r.Body.Close()

  userUID := extractUserIDfromContext(r)
  if userUID == uuid.Nil {
    http.Error(w, "Unable to find UserID in Context", http.StatusUnauthorized)
    return
  }
  roomName := vars["room_name"]

  if !router.validateRoomMemeber(roomName, userUID) {
    http.Error(w, "Failed to validate Chatroom Membership", http.StatusUnauthorized)
    return
  }

  msgs, err := router.database.Paginate(roomName, 0, db.DefaultPageSize)
  if err != nil {
    http.Redirect(w,r, "/chatrooms?error=internal_error", http.StatusInternalServerError)
    return
  }

  w.WriteHeader(http.StatusOK)
  w.Write(msgs)
}

func( router *Router )GetChatroomMessages(
  w http.ResponseWriter,
  r *http.Request,
) {
  w.Header().Set("Content-Type", "application/json")
  vars := mux.Vars(r)
  defer r.Body.Close()

  roomName := vars["room_name"]

  userUID := extractUserIDfromContext(r)
  if userUID == uuid.Nil {
    http.Error(w, "Unable to find UserID in Context", http.StatusUnauthorized)
    return
  }

  if !router.validateRoomMemeber(roomName, userUID) {
    http.Error(w, "Failed to validate Chatroom Membership", http.StatusUnauthorized)
    return
  }

  // TYLER: You were figuring out PAGINATING your messages on EnterChatroom

  // Query and Convert any received paginate parameters. If any fail, fall back
  // onto default parameters.
  pageQuery := r.URL.Query().Get("page")
  page, err := strconv.Atoi(pageQuery)
  if err != nil {
    // page = 0
    http.Error(w, "Failed to query page parameter", http.StatusBadRequest)
    return
  }
  limitQuery := r.URL.Query().Get("limit")
  limit, err := strconv.Atoi(limitQuery)
  if err != nil {
    // limit = db.DefaultPageSize
    http.Error(w, "Failed to query limit parameter", http.StatusBadGateway)
    return
  }

  // Load and send back Raw Chatroom messages before connecting User to Chatroom WS
  msgs, err := router.database.Paginate(roomName, page, limit)
  if err != nil {
    http.Redirect(w,r, "/chatrooms?error=internal_error", http.StatusInternalServerError)
    return
  }

  w.Write(msgs)
}
