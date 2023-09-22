package db

import (
	"chatatui_backend/token"
)

// --> DB Keys
const (
  CHATROOMS         = "Chatrooms"
  INACTIVECHATROOMS = "InactiveChatrooms"
  CHATROOMMEMBERS   = "ChatroomMembers"
  LIVEMEMBER        = "LiveMember"
  MESSAGES          = "Messages"
  USERS             = "Users"
  DEACTIVATEDUSERS  = "DeactivatedUsers"
  USERNAMES         = "UserNames"
  USERSONLINE       = "UsersOnline"
  USERTOKENS        = "UserTokens"
  JOINEDCHATROOMS   = "JoinedChatrooms"
  INVITATIONS       = "Invitations"
  DATEFMT           = "20060102150405.999999999"
)

// ChataTUI's Database Interface for communicating Between whatever Database that implements these
//    functions and the rest of the Application.

// Todo: Go through all of the Get DB funcitons, and see if we can return []bytes instead of Decoding(and then encoding again in HTTP function)
type ChatatuiDatabase interface {

  // GetChatroom :: Takes in a chatroom name. Queries the Database, and returns a Chatroom Object if it exists.
  GetChatroom(name string)( *Chatroom, error )

  // SaveChatroom :: Used for both Creating and Updating a Chatroom db item.
  SaveChatroom(chatroom *Chatroom, update bool) error

  // DeactivateChatroom :: Deactivates Chatroom after confirming user's identity
  DeactivateChatroom(roomName string, userID UUID) error

  // JoinChatroom :: Takes optional secret(for private chatrooms). Compares it to stores secret in /Chatrooms. If passes Will store 'JoinedChatroom' object under username in /JoinedChatrooms bucket.
  JoinChatroom(chatroom string,username string,invitation []byte) error

  // UpdateChatroomUserStatus :: Change the status of a user within a particular Chatroom.
  UpdateChatroomUserStatus(chatroom, username string, status Status) error

  // SaveChatroomMember :: in a user, and token object. Creates various Bucket entries for a new User
  SaveChatroomMember(chatroomName string,userID UUID,memberType MemberType) error

  // GetChatroomMembers :: With a given Chatroom name, this will return a map of current Chatroom Members, where map[UserName]MemberStatus
  GetChatroomMembers(chatroomName string)( map[UUID]MemberType, error)

  // StoreInvitation :: Takes in a pre-compiled invitation, we pass it though 'SaltSecret', and then store it in the buck /Invitations/{roomID-userID}
  StoreInvitation(roomID UUID, userID UUID, invitation []byte) error

  // RemoveInvitation :: Removes a Invitation from /Invitations
  RemoveInvitation(roomID UUID, userID UUID) error

  // HandleRawMessage :: Takes in a Raw message. Extracts meta data and Message, stores it in /Message Bucket
  HandleRawMessage(raw []byte) error

  // SaveMessage :: Takes in a Chatroom name and a Message Object. If Chatroom exists. Stores the Message in /Messages bucket.
  SaveMessage(chatroom string, message *Message) error

  // Pagination: Based on time. At the moment, this only paginates where a page of 1 == 1 Day. Will need to find a more refined approach for paginating messages
  Paginate(chatroomName string, page, limit int)( []byte,error )

  // GetChatroomMemberStatus: First, we check to see if /ChatroomMembers/{room_id}-{user_id} exists.If so, we return the Members Status
  GetChatroomMemberStatus(chatroomName string, userID UUID)( *MemberType, error )

  // SaveUser: Used for both Creating and Updating a User in the /Users Bucket.
  SaveUser(user User, token *token.Token) error

  // GetUserByID :: Returns a User object by indexing the /Users Bucket via userID
  GetUserByID(id UUID)( *User,error )

  // ActivateUser :: Will Restore a DeactivateUser if the UserID exists within the /DeactivateUser Bucket
  ActivateUser(userID UUID) error

  // DeactivateUser: Takes a User from /User/{user_id} and moves it to the /DeletedUser's Bucket. That way a user can reactivate their account whenevere they like.
  DeactivateUser(userID UUID) error

  // GetUserbyUsername :: Returns a User object by indexing the /Users Bucket via UserName
  GetUserbyUsername(username string)( *User, error )

  // SaveUsersOnlineStatus :: Changes the Online status of a given user.
  SaveUsersOnlineStatus(username string, isOnline bool) error

  // SaveUserToken :: Used for both creating and updating a UserToken within the /UserTokens Bucket.
  SaveUserToken(userID UUID, token *token.Token) error

  // GetUserToken :: Helper fucntion for quickly fetching a User's Access Token.
  GetUserToken(userID UUID)( *token.Token, error )
}
