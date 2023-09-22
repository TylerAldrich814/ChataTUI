package db

import (
	"time"
	// "chatatui_backend/token"
  // "github.com/ugorji/go/codec"
)

// Message: Meta data for each message stored on the DB.
//          Total size of this datastrucure ~ 64Bytes + length(content)

type Status int
const (
  Online = iota
  Background
  Offline
  Delete
)

type Message struct {
  ID         UUID      `codec:"id"`
  TimeStamp  time.Time `codec:"time_stamp"`
  UserID     UUID      `codec:"user_id"`
  Content    string    `codec:"content"`
}

type Chatroom struct {
  RoomID      UUID      `codec:"room_id"`
  RoomName    RoomName  `codec:"room_name"`
  OwnerID     UUID      `codec:"owner_id"`
  Public      bool      `codec:"public"`
}
