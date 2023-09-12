#![allow(dead_code)]
use tokio::io::{self, BufReader, AsyncBufReadExt};
use tokio_tungstenite::tungstenite::Message;
use tokio_tungstenite::connect_async;
use url::Url;
use futures_util::sink::SinkExt;
use futures_util::stream::StreamExt;
use serde::Deserialize;
use reqwest;

const HOMEURL: &'static str = "http://127.0.0.1:8080/";
const CHATROOMSURL: &'static str = "http://127.0.0.1:8080/chatrooms";
const WSURL: &'static str = "ws://127.0.0.1:8080/chatrooms/";

struct WSConnection {
    url: Url,
}

#[derive(Deserialize)]
struct APIMessage {
    message: String,
}

#[derive(Deserialize)]
struct ChatroomList {
    rooms: Vec<String>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>>{
    let home_resp = reqwest::get(HOMEURL)
        .await?
        .json::<APIMessage>()
        .await?;

    println!("{}", home_resp.message);

    loop {
        println!("\t\tOptions:");
        println!("\t\t     - 1 : Get list of Chatroom Names");
        println!("\t\t     - 2 : Attempt to join chatroom");
        println!("\t\t     - q : Quit");

        let mut choice = String::new();
        std::io::stdin().read_line(&mut choice)?;
        let choice = choice.trim();

        match choice {
            "1" => {
                let chatrooms_resp: ChatroomList = reqwest::get(CHATROOMSURL)
                    .await?
                    .json::<ChatroomList>()
                    .await?;

                println!("\t\tRoom options:");
                for room in chatrooms_resp.rooms {
                    println!("\t\t - {}", room);
                }
            }
            "2" => {
                println!("Please enter in the room_id to attempt to gain access");
                let mut room_id = String::new();
                std::io::stdin().read_line(&mut room_id)?;
                let room_id = room_id.trim();

                let ws_url = format!("{}{}/ws", WSURL, room_id);
                let url = Url::parse(&ws_url)?;
                println!("   --> RUST: ws_url {}", ws_url);
                if let Err(e) = connect_ws(url).await {
                    eprintln!("\t\tFailed to connect to WebSocket: {}", e);
                }
            }
            "q" => {
                println!("Goodbye!");
                return Ok(());
            }
            _ => continue
        }
    }
}

async fn connect_ws(url: Url) -> Result<(), Box<dyn std::error::Error>> {
    let (ws_stream, _) = connect_async(url)
        .await
        .expect("Failed to connect");

    println!("WebSocket handshake has been successfully completed");

    let (mut ws_write, mut ws_read) = ws_stream.split();

    let stdin = io::stdin();
    let mut reader = BufReader::new(stdin);
    let mut input = String::new();

    // Reading loop for incoming messages
    tokio::spawn(async move {
        while let Some(message) = ws_read.next().await {
            match message {
                Ok(msg) => {
                    if msg.is_text() || msg.is_binary() {
                        println!("Received: {}", msg);
                    }
                }
                Err(e) => {
                    eprintln!("Error receiving message: {}", e);
                }
            }
        }
    });

    'WS:loop {
        input.clear();
        reader
            .read_line(&mut input)
            .await
            .expect("Failed to read from stdin.");

        match input.trim() {
            "quit()" => {
                println!("\t\tExiting Chatroom!");
                break 'WS;
            }
            _ => {
                ws_write.send(
                    Message::Text(input.trim().into())
                ).await.expect("Failed to send message to Websocket.");
            }
        }
    }
    return Ok(());
}

// // #[tokio::main]
// async fn mainx() {
//     // Connect to the server
//     // let url = Url::parse("ws://127.0.0.1:8080/chatrooms/lydiaisacutire3/ws").unwrap();
//     let url = Url::parse("ws://127.0.0.1:8080/").unwrap();
//     let (ws_stream, _) = connect_async(url)
//         .await
//         .expect("Failed to connect");
//
//     println!("WebSocket handshake has been successfully completed");
//
//     let (mut ws_write, mut ws_read) = ws_stream.split();
//
//     let stdin = io::stdin();
//     let mut reader = BufReader::new(stdin);
//     let mut input = String::new();
//
//     // Reading loop for incoming messages
//     tokio::spawn(async move {
//         while let Some(message) = ws_read.next().await {
//             match message {
//                 Ok(msg) => {
//                     if msg.is_text() || msg.is_binary() {
//                         println!("Received: {}", msg);
//                     }
//                 }
//                 Err(e) => {
//                     eprintln!("Error receiving message: {}", e);
//                 }
//             }
//         }
//     });
//
//     loop {
//         input.clear();
//         reader
//             .read_line(&mut input)
//             .await
//             .expect("Failed to read from stdin.");
//         ws_write.send(
//             Message::Text(input.trim().into())
//         ).await.expect("Failed to send message to Websocket.");
//     }
// }
