use thiserror::Error;

#[derive(Clone, Debug,Error)]
pub enum TUIError {
	#[error("Failed to send keyboard input")]
	InputSendFailure
}