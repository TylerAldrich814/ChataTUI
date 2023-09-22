mod input;
mod errors;

use itertools::Itertools;
use log::error;
use ratatui::{layout::Constraint::*, prelude::*, widgets::*};
use tokio::sync::mpsc;
use tokio::sync::mpsc::Receiver;

pub use input::input_handler;
pub use input::KeyboardInput;
use crate::tui::errors::TUIError;

#[derive(Clone, Debug)]
pub enum UICommand {

}

pub struct UIHandler {

}

impl UIHandler {
	pub fn new() -> UIHandler {
		UIHandler{
		
		}
	}
	
	pub async fn start(&mut self, ui_rx: Receiver<UICommand>){
		
	}
}

pub async fn draw_tui(mut rx: mpsc::UnboundedReceiver<String>) {
	let backend = CrosstermBackend::new(std::io::stdout());
	let mut terminal = Terminal::new(backend).unwrap();
	let mut input = String::new();
	
	loop {
		if let Err(e) = terminal.draw(|f| {
			let chunks = Layout::default()
				.direction(Direction::Vertical)
				.margin(2)
				.constraints([Constraint::Percentage(90), Constraint::Percentage(10)].as_ref())
				.split(f.size());
			
			let chat_block = Block::default().title("Chatroom").borders(Borders::ALL);
			f.render_widget(chat_block, chunks[0]);
			
			let input_paragraph = Paragraph::new(input.as_str())
				.block(
					Block::default()
						.title("Input")
						.borders(Borders::ALL)
				);
			f.render_widget(input_paragraph, chunks[1])
			
		}){
			error!("Error occurred while drawing UI: {}", e);
		}
		if let Some(new_input) = rx.recv().await {
			input.push_str(&new_input)
		}
	}
}