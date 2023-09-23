use std::collections::VecDeque;
use anyhow::{anyhow, Result};
use crossterm::terminal::{self, EnterAlternateScreen};
use std::future::Future;
use std::sync::Arc;
use futures_util::stream::{SplitSink, SplitStream};
use log::error;
use tokio::net::TcpStream;
use tokio::signal::unix::{signal, SignalKind};
use tokio::sync::{broadcast, mpsc, Mutex};
use tokio::task::JoinHandle;
use tokio_tungstenite::tungstenite::WebSocket;
use tokio_tungstenite::{MaybeTlsStream, WebSocketStream};
use tokio_tungstenite::tungstenite::Message as TungsteniteMessage;
use crate::{input_handler, KeyboardInput, UICommand, UIHandler};
use crate::request_handlers::CTUIClient;

const MAX_CHANNEL: usize = 64;

/// run: Takes in a single 'trigger_shutdown' future. When Received, we will do any needed
///      App cleanup tasks. And Gracefully shutdown the app.
///
/// The main App loop is as follows:
///   There are 4 Major components: Input_Handler, UI_Handler, Request_Handler, and Chatroom_Handler
///   All Four components will communicate with each other via Channels.
///
/// - input_handler:   central authority, owning the input_tx Sender<KeyboardCommands>, a cloned
///     Sender<UICommand>, and a oneshot::Sender<()> for triggering the Shutdown routine.
/// - request_handler: Owns input_rx and ui_tx. When request_handler receives input, it'll parse the
///      input. If a network request is detected, it'll make the request and pass on the data if needed.
///      if no Network request was made. We'll pass the input data as-is into the UIHandler.
/// - ui_handler:  will own a ui_rx, UI Changes will only occur when ui_handler Receives a UICommand
pub async fn run_application() -> Result<()> {
	terminal::enable_raw_mode()?;
	
	let mut shutdown = Arc::new(Mutex::new(false));
	
	let chatroom_channels: Arc<Mutex<ChatroomChannels>> = Default::default();
	let chatroom_handles: Arc<Mutex<VecDeque<JoinHandle<()>>>> = Default::default();
	
	let (input_tx, input_rx) = mpsc::channel::<KeyboardInput>(MAX_CHANNEL);
	let (ui_tx, ui_rx) = broadcast::channel::<UICommand>(MAX_CHANNEL);
	let (shutdown_sender, shutdown_receiver) = tokio::sync::oneshot::channel::<()>();
	
	let mut request_handler = CTUIClient::new(chatroom_handles.clone());
	let mut ui_handler = UIHandler::new();
	
	let ui_rx_clone = ui_tx.subscribe();
	tokio::spawn(input_handler(input_tx, ui_rx_clone, shutdown_sender) );
	tokio::spawn(async move { request_handler.start(input_rx, ui_tx).await });
	tokio::spawn(async move { ui_handler.start(ui_rx).await });
	tokio::spawn(chatroom_manager(chatroom_handles.clone()));
	
	loop {
		tokio::select! {
			shutdown = shutdown_receiver => {
				// TODO: APP Cleanup
				return Ok(());
			}
		}
	}
}

type Chatroom_Sender = SplitStream<WebSocketStream<MaybeTlsStream<TcpStream>>>;
type Chatroom_Receiver = SplitSink<WebSocketStream<MaybeTlsStream<TcpStream>>, TungsteniteMessage>;
#[derive(Debug, Default)]
pub struct ChatroomChannels {
	senders: Vec<Chatroom_Sender>,
	receivers: Vec<Chatroom_Receiver>,
	join_handles: VecDeque<JoinHandle<()>>
}
impl ChatroomChannels {
	pub fn new_channel(
		&mut self,
		sender: Chatroom_Sender,
		receiver: Chatroom_Receiver,
	) {
		self.senders.push(sender);
		self.receivers.push(receiver);
	}
	
	pub async fn join_chatroom(&mut self) -> impl FnOnce() {
	
	}
}

async fn chatroom_manager(chatroom_handlers: Arc<Mutex<VecDeque<JoinHandle<()>>>>) {
	loop {
		let mut handlers = chatroom_handlers.lock().await;
		// let finished_indices = handlers.iter_mut()
		// 	.enumerate()
		// 	.filter_map(|(idx, (_, recv))| {
		// 	})
		// 	.collect::<Vec<_>>();
		
		// for idx in finished_indices.iter().rev() {
		// 	handlers.remove(*idx);
		// }
		
		drop(handlers);
		tokio::time::sleep(tokio::time::Duration::from_secs(10)).await;
	}
}