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
  HashedPassword []byte                  `codec:"hashed_password"`
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
