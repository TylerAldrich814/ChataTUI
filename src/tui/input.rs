use std::sync::Arc;
use crossterm::event::{self, KeyEvent, KeyCode};
use log::{error, warn};
use tokio::sync::{mpsc, Mutex};
use tokio::sync::mpsc::Sender;
use tokio::sync::broadcast::Receiver;
use crate::tui::errors::TUIError;
use crate::UICommand;

pub enum KeyboardInput {
	Input(String),
	Error(TUIError)
}

pub async fn input_handler(
	mut tx: Sender<KeyboardInput>,
	mut ui_tx: Receiver<UICommand>,
	mut shutdown: tokio::sync::oneshot::Sender<()>
){
	tokio::spawn(async move {
			while let Ok(ui_command) = ui_tx.recv().await {
				// TODO: detect if UICommand == UICommand::Input::Shutdown, then send and await on shutdown
				// TODO: command through KeyBoardInput and then finally send shutdown.send(())
			}
	});
	
	loop {
		match event::read() {
			Ok(event::Event::Key(KeyEvent { code, ..})) => {
				match code {
					KeyCode::Char('q') => {
						println!("Shuting Down..");
						let _ = shutdown.send(());
						return;
					}
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
				warn!("input_handler: This Event Type is supported yet: {:?}", t);
			}
			Err(e) => {
				error!("input_handler: Failed to match event::read() within input_handler: {}", e)
			}
		}
	}
}