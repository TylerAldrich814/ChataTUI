package db

import (
	// "log"
	// "fmt"

  "chatatui_backend/token"
	"github.com/google/uuid"
)


const (
  DefaultPageSize = 128
)

type MemberType int
const (
  Owner MemberType = iota
  Moderator
  Member
  Blocked
)

type (
  UUID       = uuid.UUID
  RoomName   = string
  UserName   = string
  MessageID  = UUID
)

type User struct {
  UserID         UUID                    `codec:"user_id"`
  Username       UserName                `codec:"user_name"`
  // AccessToken    token.Token             `codec:"access_token"`
  HashedPassword []byte                  `codec:"hashed_password"`
  // Chatrooms      map[RoomName]MemberType `codec:"chatrooms"`
}

// JoinedChatroom :: Data structure for tracking a User's Joined Chatrooms.
//   When a user Joins a chatroom, either Public or Private. That user will
//   receive, upon approval, a JWT token of authenticity. This token will be k
//   used for Authenticating their Chatroom Access.
type JoinedChatroom struct {
  Chatroom   string      `codec:"chatroom"`
  RoomToken  token.Token `codec:"room_token"`
  MemberType MemberType  `code:"member_type"`
}

// func(user *User)JoinedChatroom(
//   roomName RoomName,
//   // jwtToken JwtToken,
// ) error {
//   user.Chatrooms[roomName] = Member
//   return nil
// }
//
// func(user *User)CreatedChatroom(roomName RoomName) {
//   user.Chatrooms[roomName] = Moderator
// }
//
// type CtuiUsers struct {
//   Users map[UUID]User `codec:"users"`
// }
//
// func(u *CtuiUsers)OnSignup(
//   username UserName,
//   HashedPassword []byte,
// )( uuid.UUID, error ){
//   // Check to see if 'username' is taken.
//   for _, user := range u.Users {
//     if user.Username == username {
//       log.Printf(
//         " --> Error: OnSignup - Username \"%s\" already exists\n",
//         username,
//       )
//       return uuid.Nil, fmt.Errorf(
//         "Error: \"OnSignup\": The User \"%s\" already exist",
//         username,
//       )
//     }
//   }
//   uid := uuid.New()
//   u.Users[uid] = User{
//       UserID: uid,
//       Username: username,
//       IsOnline: true,
//       Chatrooms: make(map[RoomName]MemberType),
//   }
//   return uid, nil
// }
//
// // TODO: Add the Logic for Validating User Credentials.
// func(u *CtuiUsers)userLoggedIn(uid UUID) error {
//   queryUser, exists := u.Users[uid]
//   if !exists {
//     log.Println(" --> Error: userLoggedIn - User doesn't exist")
//     return fmt.Errorf("Error: \"userLoggedIn\": The User Doesn't exist")
//   }
//
//   queryUser.IsOnline = true
//   return nil
// }
// func(u *CtuiUsers)UserLoggedOut(uid UUID) error {
//   queryUser, exists := u.Users[uid]
//   if !exists {
//     log.Println(" --> Error: userLoggedIn - User doesn't exist")
//     return fmt.Errorf("Error: \"userLoggedIn\": The User Doesn't exist")
//   }
//
//   queryUser.IsOnline = false
//   return nil
// }
