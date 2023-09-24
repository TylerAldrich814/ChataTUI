use anyhow::{anyhow, Result};
use crossterm::terminal::{self, disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen};
use std::future::Future;
use std::sync::Arc;
use std::io::stdout;
use crossterm::event::{DisableMouseCapture, EnableMouseCapture};
use crossterm::execute;
use futures_util::stream::{SplitSink, SplitStream};
use log::error;
use ratatui::{backend::CrosstermBackend, Terminal};
use tokio::net::TcpStream;
use tokio::sync::{broadcast, mpsc, Mutex};
use tokio_tungstenite::{MaybeTlsStream, WebSocketStream};
use tokio_tungstenite::tungstenite::Message as TungsteniteMessage;
use crate::{input_handler, KeyboardInput, UICommand, UIHandler};
use crate::api::chatroom_handler::ChatroomChannels;
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
///
///      input. If a network request is detected, it'll make the request and pass on the data if needed.
///      if no Network request was made. We'll pass the input data as-is into the UIHandler.
/// - ui_handler:  will own a ui_rx, UI Changes will only occur when ui_handler Receives a UICommand
pub async fn run_application() -> Result<()> {
	// let _raw_ = terminal::enable_raw_mode()?;
	// execute!(stdout(), EnterAlternateScreen)?;
	
	enable_raw_mode()?;
	let mut stdout = stdout();
	execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
	let backend = CrosstermBackend::new(stdout);
	let mut terminal = Terminal::new(backend)?;
	
	let mut shutdown = Arc::new(Mutex::new(false));
	
	// let chatroom_handles: Arc<Mutex<VecDeque<JoinHandle<()>>>> = Default::default();
	let chatroom_channels: Arc<Mutex<ChatroomChannels>> = Default::default();
	let mut request_handler = CTUIClient::new(chatroom_channels.clone());
	
	let (input_tx, input_rx) = mpsc::channel::<KeyboardInput>(MAX_CHANNEL);
	let (ui_tx, ui_rx) = broadcast::channel::<UICommand>(MAX_CHANNEL);
	let (shutdown_sender, shutdown_receiver) = tokio::sync::oneshot::channel::<()>();
	
	let mut request_handler = CTUIClient::new(chatroom_channels.clone());
	let mut ui_handler = UIHandler::new();
	
	let ui_rx_clone = ui_tx.subscribe();
	tokio::spawn(input_handler(input_tx, ui_rx_clone, shutdown_sender) );
	tokio::spawn(async move { request_handler.start(input_rx, ui_tx).await });
	tokio::spawn(async move { ui_handler.start(ui_rx).await });
	tokio::spawn(chatroom_manager(chatroom_channels.clone()));
	
	loop {
		tokio::select! {
			shutdown = shutdown_receiver => {
				// TODO: APP Cleanup
				// Give back control of the terminal to the user's system.
				disable_raw_mode()?;
				execute!(
					terminal.backend_mut(),
					LeaveAlternateScreen,
					DisableMouseCapture,
				);
				terminal.show_cursor()?;
				
				// TODO: Make sure you've added any necessary Clean up operations. have you sent all/any
				// TODO: Websocket Exit flags? Did you locally store the user's App state(if needed)? etc..
				tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
				return Ok(());
			}
		}
	}
}

type Chatroom_Sender = SplitStream<WebSocketStream<MaybeTlsStream<TcpStream>>>;
type Chatroom_Receiver = SplitSink<WebSocketStream<MaybeTlsStream<TcpStream>>, TungsteniteMessage>;
type Chatroom_Stream = WebSocketStream<MaybeTlsStream<TcpStream>>;
// #[derive(Debug, Default)]
// pub struct ChatroomChannels {
// 	senders: VecDeque<Arc<Mutex<broadcast::Sender<WSMessage>>>>,
// 	receivers: VecDeque<Arc<Mutex<broadcast::Receiver<WSMessage>>>>,
// 	join_handles: VecDeque<JoinHandle<()>>
// }
// impl ChatroomChannels {
// 	pub fn create_channels(
// 		&mut self,
// 	) -> (Arc<Mutex<broadcast::Sender<WSMessage>>>, Arc<Mutex<broadcast::Receiver<WSMessage>>>) {
// 		let (tx, rx) = broadcast::channel(MAX_CHANNEL);
//
// 		let tx_arc = Arc::new(Mutex::new(tx));
// 		let rx_arc = Arc::new(Mutex::new(rx));
//
//
// 		self.senders.push_back(tx_arc);
// 		self.receivers.push_back(rx_arc);
//
// 		let tx_clone = tx_arc.clone();
// 		let rx_clone = rx_arc.clone();
//
// 		return (tx_clone, rx_clone);
// 	}
//
// 	pub fn new_handle(&mut self, joinhandle: JoinHandle<()>) {
// 		self.join_handles.push_back(joinhandle);
// 	}
// }

async fn chatroom_manager(chatroom_handlers: Arc<Mutex<ChatroomChannels>>) {
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