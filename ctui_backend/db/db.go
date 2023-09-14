package db

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"chatatui_backend/token"
)

const (
  Member   = "member"
  Moderator = "moderator"
  DefaultPageSize = 128
)

type (
  UUID       = uuid.UUID
  RoomName   = string
  UserName   = string
  MessageID  = UUID
  MemberType = string
)

type User struct {
  UserID    UUID        `json:"user_id"`
  Username  UserName    `json:"user_name"`
  IsOnline  bool        `json:"is_online"`
  HashedPassword []byte
  Chatrooms map[RoomName]MemberType `json:"chatrooms"`
}

func(user *User)JoinedChatroom(
  roomName RoomName,
  // jwtToken JwtToken,
) error {
  user.Chatrooms[roomName] = Member
  return nil
}

func(user *User)CreatedChatroom(roomName RoomName) {
  user.Chatrooms[roomName] = Moderator
}

type CtuiUsers struct {
  Users map[UUID]User `json:"users"`
}

func(u *CtuiUsers)OnSignup(
  username UserName,
  HashedPassword []byte,
)( uuid.UUID, error ){
  // Check to see if 'username' is taken.
  for _, user := range u.Users {
    if user.Username == username {
      log.Printf(
        " --> Error: OnSignup - Username \"%s\" already exists\n",
        username,
      )
      return uuid.Nil, fmt.Errorf(
        "Error: \"OnSignup\": The User \"%s\" already exist",
        username,
      )
    }
  }
  uid := uuid.New()
  u.Users[uid] = User{
      UserID: uid,
      Username: username,
      IsOnline: true,
      Chatrooms: make(map[RoomName]MemberType),
  }
  return uid, nil
}

// TODO: Add the Logic for Validating User Credentials.
func(u *CtuiUsers)userLoggedIn(uid UUID) error {
  queryUser, exists := u.Users[uid]
  if !exists {
    log.Println(" --> Error: userLoggedIn - User doesn't exist")
    return fmt.Errorf("Error: \"userLoggedIn\": The User Doesn't exist")
  }

  queryUser.IsOnline = true
  return nil
}
func(u *CtuiUsers)UserLoggedOut(uid UUID) error {
  queryUser, exists := u.Users[uid]
  if !exists {
    log.Println(" --> Error: userLoggedIn - User doesn't exist")
    return fmt.Errorf("Error: \"userLoggedIn\": The User Doesn't exist")
  }

  queryUser.IsOnline = false
  return nil
}


// Message: Meta data for each message stored on the DB.
//          Total size of this datastrucure ~ 64Bytes + length(content)
type Message struct {
  ID         UUID      `json:"id"`
  TimeStamp  time.Time `json:"time_stamp"`
  UserID     UUID      `json:"user_id"`
  Content    string    `json:"content"`
}

type Chatroom struct {
  ID          UUID                     `json:"id"`
  Name        RoomName                 `json:"name"`
  Members     map[UserName]token.Token `json:"members"`
  Messages    []Message                `json:"messages"`
  CurrentPage int                      `json:"current_page"`

  // TODO:
  //    - Will need to work the logic/security for this later.
  // InviteOnly  bool              `json:"inviteOnly"`
}

func(chatroom *Chatroom)AddMember(user string)(string, error) {

  chatroomToken, err := token.CreateTokenNoExpiration(user)
  if err != nil {
    return "", err
  }

  chatroom.Members[user] = *chatroomToken

  return chatroomToken.Token,nil
}

func(chatroom *Chatroom)RemoveMember(user User){
  delete(chatroom.Members, user.Username)
}

func(chatroom *Chatroom)NewMessage(message Message){
  chatroom.Messages = append(chatroom.Messages, message)

  messages_length := len(chatroom.Messages)
  if messages_length != 0 && messages_length % DefaultPageSize == 0 {
    chatroom.CurrentPage++
  }
}

func(chatroom *Chatroom)GetCurrentPage()( []Message,error  ){
  return chatroom.Paginate(
    chatroom.CurrentPage,
    DefaultPageSize,
    uuid.Nil,
    uuid.Nil,
  )
}
func(chatroom *Chatroom)GetPreviousPage()( []Message,error  ){
  page := 0
  if chatroom.CurrentPage != 0 {
    page = chatroom.CurrentPage - 1
  }
  return chatroom.Paginate(
    page,
    DefaultPageSize,
    uuid.Nil,
    uuid.Nil,
  )
}

// Paginate: Takes in Queried parameters from our API call to
//           '/chatrooms/{room_id}/getMessages?page=1&limit=10&beforeID={message_id}'
//           This is used for optimizing Messaging loading, especially if the Message
//           cache for any particular Chatroom exceeds a reasonable size.
// $ page int :=
func(chatroom *Chatroom)Paginate(
  page      int,
  limit     int,
  beforeID  UUID,
  afterID   UUID,
)( []Message,error ){
  if limit <= 0 || page <= 0 {
    return nil, fmt.Errorf("Invalid page or limit value")
  }

  start := (page-1)*limit
  end := start+limit

  if beforeID != uuid.Nil {
    position := 0
    for i, msg := range chatroom.Messages {
      if msg.ID == beforeID {
        position = i
        break
      }
    }
    end = position
    start = end-limit
    if start < 0 {
      start = 0
    }
  } else if afterID != uuid.Nil {
    position := len(chatroom.Messages)
    for i, msg := range chatroom.Messages {
      if msg.ID == afterID {
        position = i
        break
      }
    }
    start = position + 1
    end = start + limit
  }
  if start >= len(chatroom.Messages)   || start < 0 {
    return nil, fmt.Errorf("Page out of range")
  }
  if end > len(chatroom.Messages) {
    end = len(chatroom.Messages)
  }

  return chatroom.Messages[start:end], nil
}


// Note: This is a temporary solution before I integrate a true Database.
//       These Databases will only live as long as the Server instance(unless
//       if I push the data to a stored json)
type Chatrooms struct {
  Rooms map[RoomName]Chatroom
}

func(chatrooms *Chatrooms) GetChatroom(
  name string,
)( *Chatroom, error ){
  room, exists := chatrooms.Rooms[name];
  if !exists {
    return nil, fmt.Errorf("ERROR: Chatroom \"%s\" doesn't exist", name)
  }
  return &room, nil
}

// CreateRoom := Will create a Room, only if it exists. Returns room After creation
func(chatrooms *Chatrooms) CreateRoom(
  name string,
)( *Chatroom, error ){
  if _, exists := chatrooms.Rooms[name]; !exists {
    return nil, fmt.Errorf("ERROR: Chatroom \"%s\" already exists", name)
  }
  id := uuid.New()
  newRoom := Chatroom{
      ID:       id,
      Name:     name,
      Members:  make(map[UserName]token.Token, 128),
      Messages: make([]Message, 0, 1024),
      CurrentPage: 1,
    }

  chatrooms.Rooms[name] = newRoom
  return &newRoom, nil
}
