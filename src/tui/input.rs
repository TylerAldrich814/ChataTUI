use crossterm::event::{self, KeyEvent, KeyCode};
use log::{error, warn};
use tokio::sync::mpsc;
use tokio::sync::mpsc::Sender;
use crate::tui::errors::TUIError;

pub enum KeyboardInput {
	Input(String),
	Error(TUIError)
}

pub async fn input_handler(mut tx: Sender<KeyboardInput>) {
	loop {
		match event::read() {
			Ok(event::Event::Key(KeyEvent { code, ..})) => {
				match code {
					KeyCode::Char(c) => {
						if let Err(e) = tx.send(KeyboardInput::Input(c.into())).await {
							error!("input_handler: Failed to Send Keyboard input: {}", e);
						}
					}
					k => {
						warn!("input_handler: The Keycode \"{:?}\" is currently not supported", k);
					}
				}
			}
			Ok(t) => {
				warn!("input_handler: This Event Type is supported yet: {:?}", e);
			}
			Err(e) => {
				error!("input_handler: Failed to match event::read() within input_handler: {}", e)
			}
		}
	}
}