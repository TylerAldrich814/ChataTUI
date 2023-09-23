use crossterm::execute;
use crate::tui::errors::TUIError;
use crossterm::terminal::{self, disable_raw_mode, EnterAlternateScreen};
mod errors;
mod input;
use itertools::Itertools;
use log::error;
use ratatui::{layout::Constraint::*, prelude::*, widgets::*};
use tokio::sync::mpsc;
// use tokio::sync::mpsc::Receiver;
use tokio::sync::broadcast::Receiver;
pub use input::input_handler;
pub use input::KeyboardInput;

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
	
	/// CTUI's TUI application Loop: here is where both user input and incoming data
	/// will be received, filtered/organized and finally painted to the screen(if needed)
	pub async fn start(&self, mut ui_rx: Receiver<UICommand>) -> anyhow::Result<()> {
		terminal::enable_raw_mode()?;
		execute!(std::io::stdout(), EnterAlternateScreen)?;
		
		while let Ok(ui_command) = ui_rx.recv().await {
		
		}
		Ok(())
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