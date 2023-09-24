# 🐀 ChataTUI

#### Chatatui is a multi-threaded terminal-based TUI chatroom application crafted using both Go and Rust. Developed with a Hexagonal architectural approach, I'm designing Chatatui with extensibility in mind, enabling seamless integration of various backend architectures in the future. As of now, Chatatui operates exclusively in a local environment while I flesh out the very early application requirements.

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
- [x] Multi-threaded support: Input Handler, UI handler, HTTP Request handler, Chatroom Handler(s) handlers
- [ ] Full TUI Graphical interface
- [ ] Screens for Login, Signup, User Data, User settings, LiveChatroom
- [ ] Headless mode:
  - [ ] Integrate the nix crate. Make Chatatui work in the same way Tmux works.( PTY )
  - [ ] Add 'Slide-down' notifcations when new messages come in.
  - [ ] Fast keys to quickly change inbetween TUI and Headless

----
![My Skills](https://skillicons.dev/icons?i=rust,golang)
