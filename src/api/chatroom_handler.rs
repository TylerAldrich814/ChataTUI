use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{broadcast, Mutex};
use tokio::task::JoinHandle;
use crate::api::ws::WSMessage;


/// ActiveChatrooms:: Handles all Chatroom Websocket Join Handles. Bringing all
///   Channel Sender/Receivers to the root of our Application Thread. Allows us
///   to handle each Chatroom individually via their room_id. That way we can
///   seamlessly integrate/drop any number of chatrooms on the go.
#[derive(Debug)]
pub struct ActiveChatroom {
	pub sender: Arc<Mutex<broadcast::Sender<WSMessage>>>,
	pub receiver: Arc<Mutex<broadcast::Receiver<WSMessage>>>,
	pub join_handles: Vec<JoinHandle<()>>,
}

#[derive(Debug,Default)]
pub struct ChatroomChannels {
	pub chatrooms: HashMap<String, ActiveChatroom>,
}

impl ChatroomChannels {
	pub fn create_chatroom(&mut self, room_id: &str) -> &ActiveChatroom {
		let (tx, rx) = broadcast::channel(64);
		
		let chatroom = ActiveChatroom {
			sender: Arc::new(Mutex::new(tx)),
			receiver: Arc::new(Mutex::new(rx)),
			join_handles: vec![],
		};
		
		self.chatrooms.insert(room_id.into(), chatroom);
		&self.chatrooms[room_id]
	}
	
	pub fn remove_chatroom(&mut self, room_id: &str) {
		self.chatrooms.remove(room_id);
	}
	
	pub fn spawn_and_store(&mut self, room_id: &str, handle: JoinHandle<()>) {
		if let Some(chatroom) = self.chatrooms.get_mut(room_id) {
			chatroom.join_handles.push(handle);
		}
	}
}