package db

import (
	"bytes"
	"chatatui_backend/token"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/ugorji/go/codec"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

var JSONHandle codec.JsonHandle

// BBoltDB -> Implements 'ChatatuiDatabase'. A local database for Development. All Database functions should
//    be able to be recreated with any other database in the future.
type BBoltDB struct {
  db *bbolt.DB
}

func NewDatabase(path string)( *BBoltDB, error ){
  log.Printf(" -> NewDatabase: Before bbolt.Open(%s, 0600, nil)", path)
  db, err := bbolt.Open(path, 0600, nil)
  if err != nil {
    log.Printf(" -> NewDatabase: bbol.Open(%s, 0600, nil) FAILURE: %s", path, err.Error())
    return nil, err
  }
  log.Printf(" -> NewDatabase: After successful bbolt.Open(%s, 0600, nil)", path)

  return &BBoltDB{
    db,
  }, nil
}

func(db *BBoltDB)GetChatroom(name string)( *Chatroom, error ){
  var chatroom Chatroom
  err := db.db.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket([]byte(CHATROOMS))
    if b == nil {
      log.Printf(" -> GetChatroom: Bucket \"%s\" not found", CHATROOMS)
      return BucketNotFoundError{CHATROOMS}
    }
    data := b.Get([]byte(name))
    if data == nil {
      return GetDataError{name, CHATROOMS}
    }

    dec := codec.NewDecoderBytes(data, &JSONHandle)
    if dec == nil {
      return DecoderError{"Failed to create Decoder"}
    }
    return dec.Decode(&chatroom)
  })
  if err != nil {
    return nil, err
  }
  return &chatroom, err
}

// SaveChatroom : Requires a Chatroom Object. Gets/creates the CHATROOMS bucket
//    Tests to see if the chatroom name is taken yet. Then Creates an entry for the
//    new chatroom under /Chatrooms/{room_name}.
//    Used for both Creating and Updating a Chatroom.
//    an Error will be returned iff update is true, and chatroom doesn't exist.
func(db *BBoltDB)SaveChatroom(
  chatroom *Chatroom,
  update   bool,
) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    bucket, err := tx.CreateBucketIfNotExists([]byte(CHATROOMS))
    if err != nil {
      log.Printf(" -> Error: SaveChatroom - Failed to get %s Bucket: %s", CHATROOMS, err)
      return BucketNotFoundError{CHATROOMS}
    }


    if !update {
      exist, err := db.DoesChatroomExist(chatroom.RoomName)
      if err != nil {
        fmt.Printf(" -> SaveChatroom: Error checking if chatroom exists.")
        return err
      }
      if exist {
        fmt.Printf(" -> SaveChatroom: Chatroom name already taken")
        return fmt.Errorf("Chatroom name already taken")
      }

      if err := db.SaveChatroomMember(
        chatroom.RoomName,
        chatroom.OwnerID,
        Owner,
      ); err != nil {
        log.Printf(" -> SaveChatroom: Failed to store Chatroom Owner in /ChatroomMembers")
        return fmt.Errorf("Sever Error: Couldn't Store Chatroom Owner")
      }
    } else {
      status, err := db.GetChatroomMemberStatus(
        chatroom.RoomName,
        chatroom.OwnerID,
      )
      if err != nil {
        log.Printf(" -> SaveChatroom: Failed to retreive UserStatus for OwneID in Chatroom")
        return fmt.Errorf("OwnerID not found in Chatroom Members.")
      }
      if *status != Owner || *status != Moderator {
        log.Printf(" -> SaveChatroom: Invalid Credentials for updating Chatroom")
        return fmt.Errorf("Invalid Credentials for updating Chatroom")
      }
    }

    var data []byte
    enc := codec.NewEncoderBytes(&data, &JSONHandle)
    if err := enc.Encode(chatroom); err != nil {
      log.Printf(" -> SaveChatroom: Failed to encode Chatroom")
      return EncoderError{err.Error()}
    }
    if err := bucket.Put([]byte(chatroom.RoomName), data); err != nil {
      return PutDataError{chatroom.RoomName, CHATROOMS, err.Error()}
    }
    return nil
  })
}

// DeactivateChatroom: We Deactivate a Chatroom by first testing if the requestee
//    is the Owner of said chatroom. If so, we simply copy over the Chatroom from
//    Bucket /Chatrooms -> /InactiveChatrooms. Which isn't accessed from outside
//    the server.
func(db *BBoltDB)DeactivateChatroom(roomName string, userID UUID) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    activeBucket := tx.Bucket([]byte(CHATROOMS))
    if activeBucket == nil {
      log.Printf(" -> Error: SaveChatroom - Failed to get %s Bucket", CHATROOMS)
      return BucketNotFoundError{CHATROOMS}
    }
    inactiveBucket, err := tx.CreateBucketIfNotExists([]byte(INACTIVECHATROOMS))
    if err != nil {
      log.Printf(" -> Error: SaveChatroom - Failed to get %s Bucket: %s", INACTIVECHATROOMS, err)
      return BucketNotFoundError{CHATROOMS}
    }

    cm := activeBucket.Get([]byte(roomName))
    if cm == nil {
      return fmt.Errorf("Chatroom Doesn't Exist")
    }

    status, err := db.GetChatroomMemberStatus(roomName, userID)
    if err != nil {
      log.Printf(" -> SaveChatroom: Failed to retreive UserStatus for OwneID in Chatroom")
      return fmt.Errorf("OwnerID not found in Chatroom Members.")
    }
    if *status != Owner{
      log.Printf(" -> SaveChatroom: Invalid Credentials for updating Chatroom")
      return fmt.Errorf("Invalid Credentials for updating Chatroom")
    }

    if err := inactiveBucket.Put([]byte(roomName), cm); err != nil {
      fmt.Printf(" -> DeactivateChatroom: Failed to move Chatroom to /InactiveChatrooms")
      return PutDataError{roomName, INACTIVECHATROOMS, err.Error()}
    }

    if err := activeBucket.Delete([]byte(roomName)); err != nil {
      fmt.Printf(" -> DeactivateChatroom: Failed to Remove Chatroom from /Chatrooms")
      return DeleteDataError{roomName, CHATROOMS, err.Error()}
    }

    return nil
  })
}

// Get Chatroom: For joining Private chatrooms. Secret cannot be nil. If
// /Chatrooms/{chatroom}'s secret is NOT nil, an Error will be returned.
func(db *BBoltDB)JoinChatroom(
  chatroom string,
  username string,
  invitation []byte,
) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    cr, err := db.GetChatroom(chatroom)
    if err != nil {
      log.Printf(" -> Error: JoinChatroom: Chatroom doesn't exist")
      return err
    }

    invitations := tx.Bucket([]byte(INVITATIONS))
    if invitations == nil {
      log.Printf(" -> Error: JoinChatroom - Failed to get %s Bucket: %s", INVITATIONS, err)
      return BucketNotFoundError{INVITATIONS}
    }
    user, err := db.GetUserbyUsername(username)
    if err != nil {
      return err
    }

    if !cr.Public {
      inviteKey := inviteKey(&cr.RoomID, &user.UserID)
      roomInvitation := invitations.Get([]byte(inviteKey))
      if roomInvitation == nil {
        log.Printf(" -> JoinChatroom: Room Invitation doesn't exist")
        return GetDataError{inviteKey, INVITATIONS}
      }
      if err := CompareSecret(invitation, roomInvitation); err != nil {
        log.Printf(" -> JoinChatroom: Invitation was incorrect.")
        return FailedSecurityCheckError{"Invitation", err.Error()}
      }

      // Invitation not needed anymore. Remove it.
      if err := db.RemoveInvitation(cr.RoomID, user.UserID); err != nil {
        return err
      }
    }

    // Add User as a Memeber in Chatroom
    return db.SaveChatroomMember(chatroom, user.UserID, Member)
  })
}

func(db *BBoltDB)DoesChatroomExist(
  chatroom string,
)( bool,error ){
  exists := false
  err := db.db.View(func(tx *bbolt.Tx) error {
    active := tx.Bucket([]byte(CHATROOMS))
    if active == nil {
      log.Printf(" -> DoesChatroomExist: Failed to retreive Bucket \"%s\"", CHATROOMS)
      return BucketNotFoundError{CHATROOMS}
    }
    deactive := tx.Bucket([]byte(DEACTIVATEDUSERS))
    if deactive == nil {
      log.Printf(" -> DoesChatroomExist: Failed to retreive Bucket \"%s\"", CHATROOMS)
      return BucketNotFoundError{CHATROOMS}
    }
    if cm := active.Get([]byte(chatroom)); cm != nil {
      exists = true
    }
    if cm := deactive.Get([]byte(chatroom)); cm != nil {
      exists = true
    }
    return nil
  })
  return exists, err
}

func(db *BBoltDB)GetChatroomMembers(chatroomName string)( map[UUID]MemberType, error) {
  members := make(map[UUID]MemberType)

  err := db.db.View(func(tx *bbolt.Tx) error {
    bucket := tx.Bucket([]byte(CHATROOMMEMBERS))
    if bucket == nil {
      return BucketNotFoundError{CHATROOMMEMBERS}
    }

    prefix := []byte(chatroomName + "-")
    c := bucket.Cursor()

    for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
      // Here, we remove prefix for 'k', thus leaving us with the 'UserID'
      keyStr := string(k[len(prefix):])
      userID, err := uuid.Parse(keyStr)
      if err != nil {
        log.Printf(" -> GetChatroomMembers: Failed to parse UserID")
        return err
      }
      memberType := MemberType(v[0])
      members[userID] = memberType
    }
    return nil
  })
  if err != nil {
    log.Printf(" -> GetChatroomMembers: Failed to retreive all the Members of \"%s\"", chatroomName)
    return nil, err
  }

  return members, nil
}

// StoreInvitation: Stores a Chatroom User invitation. The actualy invitation should be created outside this func. Stores the invite in /Invitations/{roomID-userID}
func(db *BBoltDB)StoreInvitation(roomID UUID, userID UUID, invitation []byte) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    bucket, err := tx.CreateBucketIfNotExists([]byte(INVITATIONS))
    if err != nil {
      log.Printf(" -> StoreInvitation: Failed to create/get /%s Bucket", INVITATIONS)
      return BucketNotFoundError{INVITATIONS}
    }
    inviteKey := inviteKey(&roomID, &userID)

    if err := bucket.Put([]byte(inviteKey), invitation); err != nil {
      log.Printf(" -> StoreInvitation: Failed to store Chatroom Invitation.")
      return PutDataError{INVITATIONS, inviteKey, err.Error()}
    }

    return nil
  })
}

func(db *BBoltDB)RemoveInvitation(roomID UUID, userID UUID) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    bucket := tx.Bucket([]byte(INVITATIONS))
    if bucket == nil {
      return BucketNotFoundError{INVITATIONS}
    }

    inviteKey := inviteKey(&roomID, &userID)

    if err := bucket.Delete([]byte(inviteKey)); err != nil {
      log.Printf(" -> RemoveInvitation: Failed to remove /%s/%s", INVITATIONS, inviteKey)
      return DeleteDataError{INVITATIONS, inviteKey, err.Error()}
    }
    return nil
  })
}

func(db *BBoltDB)HandleRawMessage(raw []byte) error{
  return db.db.Update(func(tx *bbolt.Tx) error {
    bucket, err := tx.CreateBucketIfNotExists([]byte(MESSAGES))
    if err != nil {
      return BucketNotFoundError{MESSAGES}
    }
    extraction := struct{
      Chatroom string  `codec:"chatroom"`
      Message  Message `codec:"message"`
    }{}
    dec := codec.NewDecoderBytes(raw, &JSONHandle)
    if err := dec.Decode(&extraction); err != nil {
      return DecoderError{err.Error()}
    }

    messageKey := extraction.Chatroom + "-" + extraction.Message.TimeStamp.Format(DATEFMT)

    var data[]byte
    enc := codec.NewEncoderBytes(&data, &JSONHandle)
    if err := enc.Encode(extraction.Message); err != nil {
      return EncoderError{err.Error()}
    }
    if err := bucket.Put([]byte(messageKey), data); err != nil {
      return PutDataError{messageKey, MESSAGES, err.Error()}
    }

    return nil
  })
}

// SaveMessage :: Takes and stores a New Message object under /Messages/{chatroom-timestamp}. Might change this later.
func(db *BBoltDB)SaveMessage(chatroom string, message *Message) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    b, err := tx.CreateBucketIfNotExists([]byte(MESSAGES))
    if err != nil {
      log.Printf(" -> Error: SaveMessage - Failed to get %s Bucket: %s", CHATROOMS, err)
      return BucketNotFoundError{MESSAGES}
    }
    exist, err := db.DoesChatroomExist(chatroom)
    if err != nil {
      log.Printf(" -> SaveMessage: Error while checking if chatroom exists.")
      return err
    }
    if !exist {
      log.Printf(" -> SaveMessage: Error - Chatroom Does't exist.")
      return fmt.Errorf("Error: Received Message for a chatroom that doesn't exist")
    }

    messageKey := chatroom + "-" + message.TimeStamp.Format(DATEFMT)

    var data []byte
    enc := codec.NewEncoderBytes(&data, &JSONHandle)
    if err := enc.Encode(message); err != nil {
      return EncoderError{err.Error()}
    }
    if err := b.Put([]byte(messageKey), data); err != nil {
      return PutDataError{messageKey, MESSAGES, err.Error()}
    }
    return nil
  })
}

// Pagination: Based on time. At the moment, this only paginates where
//  a page of 1 == 1 Day. Will need to find a more refined approach for paginating messages
func(db *BBoltDB)Paginate(
  chatroomName string,
  page, limit int,
)( []byte, error ){
  if limit <= 0 || page <= 0 {
    return nil, fmt.Errorf("Invalid page or limit value")
  }
  var rawMessages [][]byte
  err := db.db.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket([]byte(MESSAGES))
    if b == nil {
      log.Printf("Bucket %s not found", MESSAGES)
      return BucketNotFoundError{MESSAGES}
    }

    c := b.Cursor()

    startKey := chatroomName + "-" + computeTimestampForPage(page, limit)
    k, v := c.Seek([]byte(startKey))

    limiter := func(i int) bool {
      return i < limit &&
        k != nil &&
        bytes.HasPrefix(k,[]byte(chatroomName+"-"));
    }

    for i := 0; limiter(i); i++ {
      // var message Message
      // dec := codec.NewDecoderBytes(v, &JSONHandle)
      // if err := dec.Decode(&message); err != nil {
      //   return DecoderError{err.Error()}
      // }
      rawMessages = append(rawMessages, v)
      k, v = c.Next()
    }
    return nil
  })
  if err != nil {
    return nil, err
  }
  // We'll seperate our Messages with byte("[") & byte("]") for client-side serialzation
  // I'm not too sure if I want to Ship out messages this way, or by Encoding and Decoding, from
  // the Database => HTTP.Repsonse body. I'll need to run some benchmarks to see if this is more
  // Viable.. Maybe I'll abstract the Byte manipulation away, that way I can support more formats
  // other than just JSON.
  var combined = []byte("[")
  for i, msg := range rawMessages {
    combined = append(combined, msg...)
    if i != len(rawMessages)-1 {
      combined = append(combined, ',')
    }
  }
  combined = append(combined, ']')

  return combined, nil
}
func computeTimestampForPage(page, limit int) string {
  date := time.Now().AddDate(0, 0, -((page - 1) * limit))
  return date.Format(DATEFMT)
}

func(db *BBoltDB)UpdateChatroomUserStatus(chatroom, username string, status Status) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    userBucket := tx.Bucket([]byte(USERNAMES))
    if userBucket == nil {
      return BucketNotFoundError{USERNAMES}
    }
    data := userBucket.Get([]byte(username))
    if data == nil {
      return GetDataError{username, USERNAMES}
    }
    var user User
    dec := codec.NewDecoderBytes(data, &JSONHandle)
    if err := dec.Decode(&user); err != nil {
      return DecoderError{err.Error()}
    }

    memtype, err := db.GetChatroomMemberStatus(chatroom, user.UserID)
    if err != nil || memtype == nil || *memtype == Blocked{
      return FailedSecurityCheckError{
        "MemberType: nil or blocked",
        err.Error(),
      }
    }

    liveBucket, err := tx.CreateBucketIfNotExists([]byte(LIVEMEMBER))
    if err != nil {
      return BucketNotFoundError{LIVEMEMBER}
    }

    data = liveBucket.Get([]byte(chatroom))

    users := make(map[UserName]string)

    dec = codec.NewDecoderBytes(data, &JSONHandle)
    if err := dec.Decode(&users); err != nil {
      return DecoderError{err.Error()}
    }

    switch status {
    case Online:
      users[username] = "Online"
    case Background:
      users[username] = "Background"
    case Offline:
      users[username] = "Offline"
    case Delete:
      // Should only be called from Owner/Moderator/member(when deleting self)
      delete(users, username)
    default:
      return fmt.Errorf("Unknonw Status Request")
    }

    var bytes []byte
    enc := codec.NewEncoderBytes(&bytes, &JSONHandle)
    if err := enc.Encode(users); err != nil {
      return EncoderError{err.Error()}
    }

    if err := liveBucket.Put([]byte(chatroom),bytes); err != nil {
      return PutDataError{LIVEMEMBER, chatroom, err.Error()}
    }

    return nil
  })
}

func(db *BBoltDB)GetChatroomUserStatus(chatroom string)( map[UserName]string, error ) {
  var stats = make(map[UserName]string)
  err := db.db.View(func(tx *bbolt.Tx) error {
    bucket := tx.Bucket([]byte(LIVEMEMBER))
    if bucket == nil {
      return BucketNotFoundError{LIVEMEMBER}
    }

    data := bucket.Get([]byte(chatroom))
    if data == nil {
      return GetDataError{chatroom, LIVEMEMBER}
    }

    dec := codec.NewDecoderBytes(data, &JSONHandle)
    if err := dec.Decode(&stats); err != nil {
      return DecoderError{err.Error()}
    }

    return nil
  })
  if err != nil {
    return nil, err
  }

  return stats, nil
}

// SaveChatroomMember :: For Saving/updating a Chatroom's Member's List.
func( db *BBoltDB )SaveChatroomMember(
  chatroomName string,
  userID UUID,
  memberType MemberType,
) error{
  return db.db.Update(func(tx *bbolt.Tx) error {
    bucket, err := tx.CreateBucketIfNotExists([]byte(CHATROOMMEMBERS))
    if err != nil {
      log.Printf(" -> Error: SaveChatroomMember - Failed to get %s Bucket: %s", CHATROOMMEMBERS, err)
      return BucketNotFoundError{CHATROOMMEMBERS}
    }
    key := chatroomName + "-" + userID.String()

    if err := bucket.Put([]byte(key), []byte{byte(memberType)}); err != nil {
      log.Printf(" -> SaveChatroomMember: Failed to update/save Chatroom Member")
      return PutDataError{key, CHATROOMMEMBERS, err.Error()}
    }
    return nil
  })
}

func(db *BBoltDB)GetChatroomMemberStatus(
  chatroomName string,
  userID UUID,
)( *MemberType, error ){
  var userStatus *MemberType = nil
  err := db.db.View(func(tx *bbolt.Tx) error {
    bucket := tx.Bucket([]byte(CHATROOMMEMBERS))
    if bucket == nil {
      return BucketNotFoundError{CHATROOMMEMBERS}
    }

    key := chatroomName + "-" + userID.String()
    data := bucket.Get([]byte(key))
    if data == nil {
      return GetDataError{key, CHATROOMMEMBERS}
    }

    stat := MemberType(data[0])

    userStatus = &stat
    return nil
  })

  return userStatus, err
}

// SaveUser :: Takes in a User Object. Will create 5 seperate bucket entries.
//      /Users           : For storing the User data sturct   [   userID : User            ]
//      /Usernames       : For indexing /Users via Username   [ username : userID          ]
//      /UsersOnline     : For storing user's online state    [ username : bool            ]
//      /UserTokens      : For storing a user's AccessTokens  [   userID : Token           ]
//      /JoinedChatrooms : For storing users joined chatrooms [ username : JoinedChatrooms ]
func(db *BBoltDB)SaveUser(user User, token *token.Token) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    uid := []byte(user.UserID.String())
    username := []byte(user.Username)

    // /Users -> Holds Entire User Object [ userID : User ]
    bucket, err := tx.CreateBucketIfNotExists([]byte(USERS))
    if err != nil {
      log.Printf(" -> Error: SaveUser - Failed to get %s Bucket: %s", USERS, err)
      return BucketNotFoundError{USERS}
    }

    var out []byte
    enc := codec.NewEncoderBytes(&out, &JSONHandle)
    if err := enc.Encode(user); err != nil {
      log.Printf(" -> Error: SaveUser - Failed to Encode new User: %s", err)
      return EncoderError{err.Error()}
    }

    if err = bucket.Put(uid, out); err != nil {
      log.Printf(" -> Error: Failed to Create new user in Database")
      return PutDataError{user.UserID.String(), USERS, err.Error()}
    }

    // /Usernames -> Holds Entire Users for indexing Users
    //               via Username [ username : userID ]
    bucket, err = tx.CreateBucketIfNotExists([]byte(USERNAMES))
    if err != nil {
      log.Printf(" -> Error: Failed to Create %s Bucket: %s", USERNAMES, err)
      return BucketNotFoundError{USERNAMES}
    }

    if err = bucket.Put(username, uid); err != nil {
      log.Printf(" -> Error: The Username \"%s\" is aldready taken: %s", username, err)
      return PutDataError{user.Username, USERNAMES, err.Error()}
    }

    // /UsersOnline -> Used for storing a user's online status [ username : bool ]
    if err := db.SaveUsersOnlineStatus(user.Username, true); err != nil {
      return err
    }

    if token != nil {
      if err := db.SaveUserToken(user.UserID, token); err != nil {
        return err
      }
    }

    if err := bucket.Put(username, BoolToBytes(true)); err != nil {
      return PutDataError{user.Username, USERNAMES, err.Error()}
    }
    return nil
  })
}

func(db *BBoltDB)GetUserByID(id UUID)( *User,error ){
  var user *User = nil
  err := db.db.View(func(tx *bbolt.Tx) error {
    bucket := tx.Bucket([]byte(USERS))
    if bucket == nil {
      log.Printf(" -> GetUserByID - Bucket \"%s\" not found.", USERS)
      return BucketNotFoundError{USERS}
    }
    data := bucket.Get(id[:])
    if data == nil {
      log.Printf(" -> GetUserById - User ID \"%s\" not found.", id)
      return GetDataError{id.String(), USERS}
    }
    dec := codec.NewDecoderBytes(data, &JSONHandle)
    return dec.Decode(&user)
  })

  return user, err
}

func(db *BBoltDB)ActivateUser(userID UUID) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    deactivatedBucket := tx.Bucket([]byte(DEACTIVATEDUSERS))
    if deactivatedBucket == nil {
      return fmt.Errorf(" -> ActivateUser: Failed to get \"%s\" Bucket", DEACTIVATEDUSERS)
    }
    data := deactivatedBucket.Get([]byte(userID.String()))
    if data == nil {
      return GetDataError{userID.String(), DEACTIVATEDUSERS}
    }

    var user User
    dec := codec.NewDecoderBytes(data, &JSONHandle)
    if dec == nil {
      return fmt.Errorf(" -> ActivateUser: Failed to create codec Decoder.")
    }
    if err := dec.Decode(&user); err != nil {
      log.Printf(" -> ActivateUser: Failed to Decode User")
      return err
    }

    usersBucket := tx.Bucket([]byte(USERS))
    if usersBucket == nil {
      return fmt.Errorf(" -> ActivateUser: Failed to get \"%s\" Bucket", USERS)
    }

    usernameBucket := tx.Bucket([]byte(USERNAMES))
    if usernameBucket == nil {
      return fmt.Errorf(" ->  ActivateUser: Failed to get \"%s\" Bucket", USERNAMES)
    }

    if err := usersBucket.Put(userID[:], data); err != nil {
      log.Printf(" -> ActivateUser: Failed to Move User to /%s Bucket.", USERS)
      return err
    }
    if err := usernameBucket.Put([]byte(user.Username), userID[:]); err != nil {
      log.Printf(" -> ActivateUser: Failed to add User to /%s Bucket.", USERNAMES)
      return err
    }

    if err := deactivatedBucket.Delete(userID[:]); err != nil {
      log.Printf(" -> ActivateUser: Failed to Delete User to /%s Bucket.", DEACTIVATEDUSERS)
      return err
    }

    return nil
  })
}

// DeactivateUser: Takes a User from /User/{user_id} and moves it to the /DeletedUser's Bucket
func(db *BBoltDB)DeactivateUser(userID UUID) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    // Get User from /Users
    usersBucket := tx.Bucket([]byte(USERS))
    if usersBucket == nil {
      return fmt.Errorf(" -> DeactivateUser: Failed to get \"%s\" Bucket", USERS)
    }
    data := usersBucket.Get([]byte(userID.String()))
    if data == nil {
      return GetDataError{userID.String(), USERS}
    }

    var user User
    dec := codec.NewDecoderBytes(data, &JSONHandle)
    if dec == nil {
      return fmt.Errorf(" -> DeactivateUser: Failed to create codec Decoder.")
    }
    if err := dec.Decode(&user); err != nil {
      log.Printf(" -> DeactivateUser: Failed to Decode User")
      return DecoderError{err.Error()}
    }

    deactivatedBucket, err := tx.CreateBucketIfNotExists([]byte(DEACTIVATEDUSERS ))
    if err != nil {
      log.Printf(" -> DeactivateUser: Failed to create or get \"%s\" Bucket.", DEACTIVATEDUSERS)
      return BucketNotFoundError{DEACTIVATEDUSERS}
    }
    if err := deactivatedBucket.Put(userID[:], data); err != nil {
      log.Printf(" -> DeactivateUser: Failed to Move User to /%s Bucket.", DEACTIVATEDUSERS)
      return PutDataError{userID.String(), DEACTIVATEDUSERS, err.Error()}
    }

    usernameBucket := tx.Bucket([]byte(USERNAMES))
    if usernameBucket == nil {
      return BucketNotFoundError{USERNAMES}
    }

    // Removes User from /Usernames bucket
    if err := usernameBucket.Delete([]byte(user.Username)); err != nil {
      log.Printf(" -> DeactivateUser: Failed to remove \"%s\" from /%s Bucket", user.Username, USERNAMES)
      return DeleteDataError{user.Username, USERNAMES, err.Error()}
    }
    // Once we know everything has passed, we now attempt to delete the User from /Users.
    // Yea, I know that BBolt will rollback any changes if this anonymous function returns an error. But safer than sorry.
    if err := usersBucket.Delete([]byte(userID.String())); err != nil {
      log.Printf(" -> DeactivateUser: Failed to decode User \"%s\" from /%s bucket", userID.String(), USERS)
      return DeleteDataError{userID.String(), USERS, err.Error()}
    }
    return nil
  })
}

func(db *BBoltDB)GetUserbyUsername(username string)( *User, error ){
  var user *User = nil

  err := db.db.View(func(tx *bbolt.Tx) error {
    bucket := tx.Bucket([]byte(USERNAMES))
    if bucket == nil {
      log.Printf(" -> GetUserbyUsername - Bucket \"%s\" not found.", USERNAMES)
      return BucketNotFoundError{USERNAMES}
    }
    uid := bucket.Get([]byte(username))
    if uid == nil {
      log.Printf(" -> GetUserbyUsername - User ID \"%s\" not found.", username)
      return GetDataError{username, USERNAMES}
    }

    bucket = tx.Bucket([]byte(USERS))
    if bucket == nil {
      log.Printf(" -> GetUserbyUsername - Bucket \"%s\" not found.", USERS)
      return BucketNotFoundError{USERS}
    }
    data := bucket.Get(uid)
    if data == nil {
      log.Printf(" -> GetUserbyUsername - User not found.")
      return GetDataError{username, USERS}
    }

    dec := codec.NewDecoderBytes(data, &JSONHandle)
    return dec.Decode(&user)
  })

  return user, err
}

func(db *BBoltDB)SaveUsersOnlineStatus(username string, isOnline bool) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    bucket, err := tx.CreateBucketIfNotExists([]byte(USERSONLINE))
    if err != nil {
      log.Printf(" -> SaveUsersOnlineStatus: Failed to Create or get \"%s\" Bucket", USERSONLINE)
      return BucketNotFoundError{USERSONLINE}
    }
    if err := bucket.Put([]byte(username), BoolToBytes(isOnline)); err != nil {
      var s string
      if isOnline{ s = "Online" } else { s = "Offline" }
      log.Printf(
        " -> SaveUsersOnlineStatus: Failed to update/create \"%s\"'s online status to \"%s\": %s",
        username, s, err,
      )
      return PutDataError{s, USERSONLINE, err.Error()}
    }
    return nil
  })
}

// SaveUserToken :: Used for both creating and updating a UserToken within the
//     /UserTokens Bucket.
func(db *BBoltDB)SaveUserToken(userID UUID, token *token.Token) error {
  return db.db.Update(func(tx *bbolt.Tx) error {
    b, err := tx.CreateBucketIfNotExists([]byte(USERTOKENS))
    if err != nil {
      return BucketNotFoundError{USERTOKENS}
    }

    var data []byte
    enc := codec.NewEncoderBytes(&data, &JSONHandle)
    if err := enc.Encode(token); err != nil {
      return EncoderError{err.Error()}
    }

    if err := b.Put([]byte(userID.String()), data); err != nil {
      return PutDataError{userID.String(), USERTOKENS, err.Error()}
    }
    return nil
  })
}

func(db *BBoltDB)GetUserToken(userID UUID)( *token.Token, error ){
  var token token.Token
  err := db.db.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket([]byte(USERTOKENS))
    if b == nil {
      log.Printf(" -> GetUserToken: Bucket \"%s\" not found", USERTOKENS)
      return BucketNotFoundError{USERTOKENS}
    }
    data := b.Get([]byte(userID.String()))
    if data == nil {
      log.Printf(" -> GetUserToken: User Token for \"%s\" doesn't exist", userID.String())
      return GetDataError{userID.String(), USERTOKENS}
    }

    dec := codec.NewDecoderBytes(data, &JSONHandle)
    if dec == nil {
      return DecoderError{"Decoder is nil"}
    }
    return dec.Decode(&token)
  })
  if err != nil {
    return nil, err
  }
  return &token, nil
}

func(db *BBoltDB)Close() {
  db.db.Close()
}

// ----------------------------- DB Helper Funcs -----------------------------
func BoolToBytes(b bool) []byte {
  if b { return []byte{1} }
  return []byte{0}
}
func BytesToBool(b []byte) bool {
  return b[0] == 1
}

func inviteKey(roomID *UUID, userID *UUID) string {
  return roomID.String() + "-" + userID.String()
}

func SaltSecret(secret *string)( []byte,error ){
  sec, err := bcrypt.GenerateFromPassword(
    []byte(*secret),
    bcrypt.DefaultCost,
  )
  if err != nil {
    return nil, fmt.Errorf("Failed to Salt Secret")
  }

  return sec, nil
}

func CompareSecret(got []byte, want []byte) error {
  if err := bcrypt.CompareHashAndPassword(
    got, want,
  ); err != nil {
    return fmt.Errorf("passed secret is not correct.")
  }
  return nil
}
