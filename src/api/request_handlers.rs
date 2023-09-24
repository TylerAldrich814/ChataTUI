#![allow(unused_imports)]
// src/api/request_handlers.rs

// Routes
// [x] /User/Signup                   | POST
// [x] /User/Signin                   | POST
// [ ] /chatrooms/                    | GET
// [x] /chatrooms/                    | POST
// [x] /chatrooms/{room_id}           | GET
// [x] /chatrooms/{room_id}           | PUT
// [x] /chatrooms/{room_id}           | DELETE
// [x] /chatrooms/{room_id}/join      | GET
// [x] /chatrooms/{room_id}/load      | GET
// [x] /chatrooms/{room_id}/ws        | WS

use anyhow::{Result, anyhow};
use chrono::{DateTime, Utc};
use crate::{KeyboardInput, UICommand};
use crate::api::errors::ChatroomError;
use crate::api::ws::{receive_from_ws, send_to_ws};
use futures_util::{sink::SinkExt, stream::StreamExt, TryFutureExt};
use log::{error, info, warn};
use reqwest;
use reqwest::{Response, StatusCode};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use super::endpoints;
use tokio::sync:: Mutex;
use tokio::sync::mpsc::Receiver;
use tokio::sync::broadcast::Sender;
use tokio::task::JoinHandle;
use tokio_tungstenite::connect_async;
use uuid::{Uuid};
use crate::api::chatroom_handler::ChatroomChannels;

#[derive(Debug, Clone, Deserialize)]
struct APIErrorResponse {
    error: String,
}

/// HomeMessage: The Expected returned JSON Data structure for when we call http:..:8080/
#[derive(Deserialize)]
struct HomeMessage {
    #[serde(rename="authed")]
    authed: bool,
    #[serde(rename="token")]
    token:  String,
}

#[derive(Serialize)]
struct UserCredentials {
    #[serde(rename="username")]
    username: String,
    #[serde(rename="password")]
    password: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Message {
    id: Uuid,
    time_stamp: DateTime<Utc>,
    user_id: Uuid,
    content: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Messages {
    messages: Vec<Message>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub enum UserStatus {
    Online,
    Offline,
    Away,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct APIMessage {
    status: UserStatus,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct Chatroom {
    room_id: Uuid,
    room_name: String,
    owned_id: Uuid,
    public: bool
}
#[derive(Debug, Clone, Serialize, Deserialize)]
struct JoinChatroomHandle {
    room_name: String,
    user_name: String,
    invitation: Option<String>,
}
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct AuthToken {
    token: String
}
impl std::fmt::Display for AuthToken {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.token)
    }
}

/// CTUIClient:: HTTP API Handler, for communicating with our Chatatui backend server.
/// - client: A reqwest client that will for the lifetime of our Application.
/// - auth_token: a JWT authorization token used for user authentication with All HTTP endpoints, expect Singup/Signin
/// - chatroom_channels: Chatroom Websocket thread connection handlers. With ease of access via Chatroom room_id.
pub struct CTUIClient {
    client: reqwest::Client,
    auth_token: Option<AuthToken>,
    chatroom_channels: Arc<Mutex<ChatroomChannels>>,
}

// -------------------------------------- ENDPOINTS ------------------------------------gg--

impl CTUIClient {
    pub fn new(chatroom_channels: Arc<Mutex<ChatroomChannels>>) -> CTUIClient {
        CTUIClient {
            client: reqwest::Client::new(),
            auth_token: None,
            chatroom_channels,
        }
    }
    
    /// CTUIClient::Run -> When we receive data from input_rx. We match over the input. Specific
    ///     inputs will be mapped to specific HTTP API requests. Data may or may not be returned
    ///     from the API call. On Return, we take the data, and send it off to our UI via 'ui_tx
    pub async fn start(&mut self, mut input_rx: Receiver<KeyboardInput>, ui_tx: Sender<UICommand>) {
        while let Some(input) = input_rx.recv().await {
        
        }
    }

    /// HTTP Endpoints POST -> /Users/Signin
    pub async fn sign_in(&mut self, username: &str, password: &str) -> Result<()> {
        let url = endpoints::post_sign_in_url();
        let parameters = [("username", username), ("password", password)];

        let response = self.client.post(&url)
            .form(&parameters)
            .send()
            .await?
            .json::<AuthToken>()
            .await?;
        self.auth_token = Some(response);

        Ok(())
    }

    /// HTTP Endpoints POST -> /Users/Signup
    pub async fn sign_up(&mut self, username: &str, password: &str) -> Result<()> {
        let url = endpoints::post_sign_up_url();
        let parameters = [("username", username), ("password", password)];

        let response = self.client.post(&url)
            .form(&parameters)
            .send()
            .await?
            .json::<AuthToken>()
            .await?;
        self.auth_token = Some(response);

        Ok(())
    }

    /// HTTP Endpoint POST -> /chatrooms
    /// Returns a StatusOK when successful. Maybe return an 'owners' token?
    pub async fn create_chatroom(&self, chatroom: &Chatroom) -> Result<()> {
        let token = match &self.auth_token {
            Some(token) => token,
            None => {
                warn!(" -> chatroom_meta: auth_token is required.");
                return Err(anyhow!(ChatroomError::Unauthorized));
            }
        };
        let url = endpoints::get_chatrooms_url();

        if let Ok(resp) = self.client.post(url)
            .bearer_auth(token)
            .json(chatroom)
            .send()
            .await
            .map_err(|_| ChatroomError::Unknown)
        {
            if resp.status() == StatusCode::OK {
                return Ok(())
            }
            return Err(anyhow!("Response returned is not OK"))
        } else {
            Err(anyhow!(ChatroomError::Unknown))
        }
    }

    /// HTTP Endpoint GET -> /chatrooms/{room_id}
    pub async fn chatroom_meta(&self, room_name: &str) -> Result<Chatroom> {
        let token = match &self.auth_token {
            Some(token) => token,
            None => {
                warn!(" -> chatroom_meta: auth_token is required.");
                return Err(anyhow!(ChatroomError::Unauthorized))
            }
        };
        let url = endpoints::get_chatroom_by_id_url(room_name);

        let response: Result<Response, ChatroomError> = self.client.get(url)
            .bearer_auth(token)
            .send()
            .await
            .map_err(|_| ChatroomError::Unknown);

        if let Ok(resp) = response {
            if resp.status() != StatusCode::OK {
                return Err(anyhow!("Status returned is not OK"));
            }
            
            return resp.json::<Chatroom>().await.map_err(|_| anyhow!(ChatroomError::Unknown));
        }
        return Err(anyhow!(response.unwrap_err()));
    }

    /// HTTP Endpoint PUT -> /chatrooms/{room_id}
    pub async fn update_chatroom(&self, chatroom: &Chatroom) -> Result<()> {
        let token = match &self.auth_token {
            Some(token) => token,
            None => {
                warn!(" -> chatroom_meta: auth_token is required.");
                return Err(anyhow!(ChatroomError::Unauthorized));
            }
        };
        let url = endpoints::get_chatroom_by_id_url(&chatroom.room_name);

        let response = self.client.put(url)
            .bearer_auth(token)
            .json(&chatroom)
            .send()
            .await
            .map_err(|e| ChatroomError::Unknown)?;
        if response.status() != StatusCode::OK {
            return Err(anyhow!("Status returned is not OK"));
        }
        return Ok(())
    }

    /// HTTP Endpoint DELETE -> /chatrooms/{room_id}
    pub async fn delete_chatroom(&self, room_name: &str) -> Result<()> {
        let token = match &self.auth_token {
            Some(token) => token,
            None => {
                warn!(" -> delete_chatroom: auth_token is required.");
                return Err(anyhow!(ChatroomError::Unauthorized));
            }
        };

        let url = endpoints::get_chatroom_by_id_url(room_name);
        let resp = self.client.delete(url)
            .bearer_auth(token)
            .send()
            .await
            .map_err(|_| ChatroomError::Unknown)?;

        if resp.status() != StatusCode::OK {
            return Err(anyhow!("Status returned is not OK"));
        }
        return Ok(())
    }

    /// HTTP Endpoint DELETE -> /chatrooms/{room_id}/join
    pub async fn join_chatroom(
        &self,
        room_name: &str,
        user_name: &str,
        invitation: Option<String>,
    ) -> Result<()> {
        let token = match &self.auth_token {
            Some(token) => token,
            None => {
                warn!(" -> join_chatroom: auth_token is required.");
                return Err(anyhow!(ChatroomError::Unauthorized));
            }
        };
        let url = endpoints::get_chatroom_join_url(room_name);
        
        let handle = JoinChatroomHandle {
                room_name: room_name.into(),
                user_name: user_name.into(),
                invitation,
        };
        
        let response = self.client.post(url)
            .bearer_auth(token)
            .json(&handle)
            .send()
            .await
            .map_err(|_| ChatroomError::Unknown)?;

        if response.status().is_success() {
            return Ok(())
        }
        // TODO: Handle this
        let error_resp: APIErrorResponse = response.json().await?;
        warn!(" -> join_chatroom: Error Occurred: {}", error_resp.error);
        return Err(anyhow!(ChatroomError::Unknown));
    }

    /// HTTP Endpoint GET -> /chatrooms/{room_id}/load
    pub async fn chatroom_onload(&self, room_name: &str) -> Result<Messages> {
        let token = match &self.auth_token {
            Some(token) => token,
            None => {
                warn!(" -> chatroom_onload: auth_token is required.");
                return Err(anyhow!(ChatroomError::Unauthorized));
            }
        };
        let url = endpoints::chatroom_on_load(room_name);

        let response = self.client.get(&url)
            .bearer_auth(token)
            .send()
            .await
            .map_err(|_| ChatroomError::Unknown)?;

        if response.status().is_success() {
            return response.json::<Messages>().await.map_err(|_| anyhow!(ChatroomError::Unknown));
        }
        // TODO: Be more concise with this error.
        return Err(anyhow!("Failed to load Chatroom Data"));
    }

    /// HTTP Endpoint WS Update -> /chatrooms/{room_id}/ws
    pub async fn chatroom_ws(
        &mut self,
        room_id: &str,
    ) -> Result<()>{
        let ws_url = endpoints::chatroom_ws_url(room_id);
        let mut locked_chatrooms = self.chatroom_channels.lock().await;
        
        let active_chatroom = locked_chatrooms.create_chatroom(room_id);
        let tx = active_chatroom.sender.clone();
        let rx = active_chatroom.receiver.clone();
        
        match connect_async(ws_url).await {
            Ok((ws_stream, _response)) => {
                info!(" -> chatroom_ws: Successfully upgraded to Websocket");
                let (write_to_ws, read_from_ws) = ws_stream.split();
                
                // Create our asynchronous Tokio Spawns, and push them to WSHandles
                let mut channels = self.chatroom_channels.lock().await;
                channels.spawn_and_store(
                    room_id,
                    tokio::spawn(receive_from_ws(read_from_ws, tx ))
                );
                channels.spawn_and_store(
                    room_id,
                    tokio::spawn(send_to_ws(write_to_ws, rx))
                );
            }
            Err(e) => {
                error!("chatroom_ws: Failed to Connect to Websocket: {}", e);
                return Err(anyhow!(e));
            }
        }
        Ok(())
    }
}

pub struct ChatroomSpawns {
    id: Uuid,
    recv_handle: JoinHandle<()>,
    send_handle: JoinHandle<()>,
}