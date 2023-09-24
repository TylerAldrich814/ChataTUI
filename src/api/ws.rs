// ChataTUI's Frontend Websocket handler data structures and functions

use std::borrow::Cow;
use std::sync::Arc;
use tokio::sync::{broadcast, mpsc, Mutex};
use futures_util::stream::{SplitSink, SplitStream, StreamExt};
use futures_util::sink::SinkExt;
use log::{error, warn};
use serde::{Deserialize, Serialize};
use tokio::net::TcpStream;
use tokio_tungstenite::{MaybeTlsStream, WebSocketStream};
use tokio_tungstenite::tungstenite::Message as TungsteniteMessage;
use tokio_tungstenite::tungstenite::protocol::CloseFrame;
use tokio_tungstenite::tungstenite::protocol::frame::coding::CloseCode;
use crate::request_handlers::{APIMessage, Message};


/// WSCommand: A Handler Enum for all Inbound & Outbound Websocket Messages.
///     UserMessage: For Taking user's messages and passing it through the WS
///     APIMessage:  For Any Automatic API Messages to the Backend(User status, errors, etc..)
///     Close: Websocket Frame for telling the Backend that we'll be closing the connection.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub enum WSMessage {
    UserMessage(Box<Message>),
    APIStatus(Box<APIMessage>),
    Error(Box<String>),
    Close,
}

pub(crate) async fn receive_from_ws(
    mut ws_read: SplitStream<WebSocketStream<MaybeTlsStream<TcpStream>>>,
    mut tx: Arc<Mutex<broadcast::Sender<WSMessage>>>,
){
    while let Some(Ok(ws_msg)) = ws_read.next().await {
        let tx_clone = tx.clone();
        match ws_msg {
            TungsteniteMessage::Text(msg) => {
                let wsmsg: Result<WSMessage, serde_json::Error> = serde_json::from_str(&msg);
                if let Ok(msg) = wsmsg {
                    if let Err(e) = tx_clone.lock().await.send(msg) {
                       warn!("receive_from_ws: Failed to send WSMessage from Websocket Message: {}", e);
                    }
                }else {
                    warn!(
                        "Tungstenite Message was not Recognized: {:?}",
                        wsmsg.unwrap_err(),
                    );
                }
            }
            TungsteniteMessage::Close(reason) => {
                if let Some(close_frame) = reason {
                    warn!(
                        "Websocket closed with code: {:?} and reason: {:?}",
                        close_frame.code, close_frame.reason
                    );
                }
                if let Err(e) = tx_clone.lock().await.send(WSMessage::Close) {
                   error!("Failed to send Close Command after receiving close frame: {}", e);
                }
                break;
            }
            other_msg => {
                warn!("Websocket Message kind not currently supported: {:?}", other_msg);
            }
        }
        drop(tx_clone);
    }
}

/// Handles both Websocket Messaging cases.
/// User made messaging.
/// Background API Related Messaging.
pub(crate) async fn send_to_ws(
    mut ws_write: SplitSink<WebSocketStream<MaybeTlsStream<TcpStream>>, TungsteniteMessage>,
    mut rx: Arc<Mutex<broadcast::Receiver<WSMessage>>>,
){
    while let Ok(message) = rx.clone().lock().await.recv().await {
        match message {
            WSMessage::UserMessage(msg) => {
                match serde_json::to_string(&msg) {
                    Ok(json) => {
                        let tmsg = TungsteniteMessage::Text(json);
                        if let Err(e) = ws_write.send(tmsg).await {
                            error!("Failed to send new WSMessage::UserMessage through Websocket: {}", e);
                        }
                    }
                    Err(e) => {
                        error!("Failed to Serialize WSMessage::UserMessage: {}", e);
                    }
                }
            }
            WSMessage::APIStatus(api) => {
                match serde_json::to_string(&api) {
                    Ok(status_json) => {
                        let new_msg = TungsteniteMessage::Text(status_json);
                        if let Err(e) = ws_write.send( new_msg ).await {
                            error!("Failed to send WSMessage::APIMessage through Websocket: {}", e);
                        }
                    }
                    Err(e) => {
                        error!("Failed to Serialize WSMessage::APIMessage: {}", e)
                    }
                }
            },
            WSMessage::Error(msg) => {
                error!("handle_ws_input: Websocket returned an Error: {}", msg);
                todo!()
            }
            WSMessage::Close => {
                let close_frame = CloseFrame {
                    code: CloseCode::Normal,
                    reason: Cow::Borrowed(""),
                };
                
                if let Err(e) = ws_write.send(
                    TungsteniteMessage::Close(Some(close_frame))
                ).await {
                    error!("Failed to send Close frame to Websocket: {}", e);
                }
            }
        }
    }
}