# 🐀 ChataTUI


#### A Terminal TUI Chatroom Application, written in Go and Rust. Currently only works locally.

##### Features: Completed and planned
 #### Backend
- [x] User Creation
- [x] User Authentication via JWT token. Can be used to auto-login.
- [x] Chatroom Creation, both public and private.
- [x] Chatroom Authentication. User must become a Chatroom member before granted access to the Chatroom websocket
- [x] Chatroom messages load via pagination, can be fine-tuned with HTTP Get query parameters.
- [ ] Configured to work on Google Cloud
- [ ] Configured to work on LibP2P GossipSub network(Websockets obviously wouldn't work with this)

#### Frontend
- [x] Users can join local instance of ChataTUI
- [x] Create/join Chatrooms
- [x] Send and receive messages
- [ ] Full TUI Graphical interface
- [ ] Screens for Login, Signup, User Data, User settings, LiveChatroom

----
![My Skills](https://skillicons.dev/icons?i=rust,golang)
