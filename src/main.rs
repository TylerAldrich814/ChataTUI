#![allow(dead_code)]

use std::error::Error;
use std::future::Future;
use std::sync::Arc;
use std::time::Duration;
use chatatui_lib::{input_handler, UIHandler};
use crossterm::execute;
use tokio::io::{self, BufReader, AsyncBufReadExt};
use tokio::sync::{Mutex, Notify};
use url::Url;
use futures_util::stream::StreamExt;
use serde::Deserialize;
use reqwest;
use crossterm::terminal::{self, disable_raw_mode, EnterAlternateScreen};
use tokio::sync::mpsc;
use tokio::time::sleep;
use chatatui_lib::request_handlers::{CTUIClient, WSArc};

const HOMEURL: &'static str = "http://127.0.0.1:8080/";
const CHATROOMSURL: &'static str = "http://127.0.0.1:8080/chatrooms";
const WSURL: &'static str = "ws://127.0.0.1:8080/chatrooms/";

const BACKOFF_MIN: u64 = 100;//ms
const BACKOFF_MAX: u64 = 10 * 1_000;//ms

struct WSConnection {
  url: Url,
}

#[derive(Deserialize)]
struct HomeMessage {
  #[serde(rename="authed")]
  Authed: bool,
  #[serde(rename="token")]
  Token:  String,
}

async fn spawn_shutdown(shutdown_signal: &Arc<Notify>) {
  let i = tokio::spawn({
    let shutdown_signal = shutdown_signal.clone();
    async move {
      tokio::signal::ctrl_c().await.expect("Failed to listen to Ctrl-C");
      shutdown_signal.notify()
    }
  });
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>>{
  let shutdown_signal = Arc::new(Notify::new());
  
  terminal::enable_raw_mode()?;
  execute!(io::stdout(), EnterAlternateScreen)?;
  
  let mut chatroom_handles = WSArc::default();
  
  let mut request_handler = CTUIClient::new(chatroom_handles.clone());
  let mut ui_handler = UIHandler::new();
  
  let (input_tx, input_rx) = mpsc::channel(32);
  let (ui_tx, ui_rx)       = mpsc::channel(32);
  
  // The Three main components of ChataTUI. All thee spawns are connected via channels for
  // communicating App changes and updates throughout the lifespan of the application.
  let input_handle   = tokio::spawn(input_handler(input_tx));
  let ui_handle      = tokio::spawn(ui_handler.start(ui_rx));
  let request_handle = tokio::spawn(request_handler.run(input_rx, ui_tx));
  
  // Each of the 3 core Handlers for ChataTUI will have their own dedicated 'retry' backoff_timers
  // just in case if any of the 3 runtimes encounters an Error.
  let mut input_backoff_duration   = Duration::from_millis(BACKOFF_MIN);
  let mut ui_backoff_duration      = Duration::from_millis(BACKOFF_MIN);
  let mut request_backoff_duration = Duration::from_millis(BACKOFF_MIN);
  'ChataTUI:loop {
    tokio::select! {
      j = run_with_backoff(input_handle, &mut input_backoff_duration) => {
        if let Err(e) = j {
          error!("Failed to Join input handle");
        }
      }
      j = run_with_backoff(ui_handle, &mut ui_backoff_duration) => {
        if let Err(e) = j {
          error!("Failed to Join UI handle");
        }
      }
      j = run_with_backoff(request_handler, &mut request_backoff_duration) => {
        if let Err(e) = j {
          error!("Failed to Join request handle");
        }
      }
      chatroom = async {
        let mut handles = chatroom_handles.lock().await;
        if let Some(handle) = handles.pop() {
          let _ = handle.await;
        }
      } => {
        info!("Chatroom Handle Completed");
      }
    }
  }
}
async fn run_with_backoff<F: Future<Output=Result<(), dyn Error>>>(
  handler: F,
  backoff_duration: &mut Duration,
){
  if handler.await.is_err() {
    *backoff_duration = handle_backoff(*backoff_duration)
  } else {
    *backoff_duration = Duration::from_millis(BACKOFF_MIN);
  }
}

async fn handle_backoff(mut current_duration: Duration) -> Duration {
  let max_duration = Duration::from_secs(BACKOFF_MAX);
  sleep(current_duration).await;
  current_duration = current_duration.saturating_mul(2);
  
  if current_duration > max_duration {
    current_duration = max_duration;
  }
  current_duration
}

// async fn testmain() -> Result<(), Box<dyn std::error::Error>>{
//     let home_resp = reqwest::get(HOMEURL)
//         .await?
//         .json::<HomeMessage>()
//         .await?;
//
//     println!(" -> Home Message: Authed - {}", home_resp.Authed);
//     println!(" -> Home Message: Token - {}", home_resp.Token);
//
//     loop {
//         println!("\t\tOptions:");
//         println!("\t\t     - 1 : Get list of Chatroom Names");
//         println!("\t\t     - 2 : Attempt to join chatroom");
//         println!("\t\t     - q : Quit");
//
//         let mut choice = String::new();
//         std::io::stdin().read_line(&mut choice)?;
//         let choice = choice.trim();
//
//         match choice {
//             "1" => {
//                 let chatrooms_resp: ChatroomList = reqwest::get(CHATROOMSURL)
//                     .await?
//                     .json::<ChatroomList>()
//                     .await?;
//
//                 println!("\t\tRoom options:");
//                 for room in chatrooms_resp.rooms {
//                     println!("\t\t - {}", room);
//                 }
//             }
//             "2" => {
//                 println!("Please enter in the room_id to attempt to gain access");
//                 let mut room_id = String::new();
//                 std::io::stdin().read_line(&mut room_id)?;
//                 let room_id = room_id.trim();
//
//                 let ws_url = format!("{}{}/ws", WSURL, room_id);
//                 let url = Url::parse(&ws_url)?;
//                 println!("   --> RUST: ws_url {}", ws_url);
//                 if let Err(e) = connect_ws(url).await {
//                     eprintln!("\t\tFailed to connect to WebSocket: {}", e);
//                 }
//             }
//             "q" => {
//                 println!("Goodbye!");
//                 return Ok(());
//             }
//             _ => continue
//         }
//     }
// }