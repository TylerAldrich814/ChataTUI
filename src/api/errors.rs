use reqwest::StatusCode;
use thiserror::Error;
#[derive(Clone, Debug, Error)]
pub enum ChatroomError {
	#[error("Invalid chatroom data")]
	InvalidData,
	#[error("Unauthorized")]
	Unauthorized,
	#[error("Failed to save or update chatroom")]
	DatabaseError,
	#[error("Websocket Connection Failed")]
	ConnectionFailed,
	#[error("Unknown error occurred")]
	Unknown,
}
impl Into<ChatroomError> for StatusCode {
	fn into(self) -> ChatroomError {
		match self {
			StatusCode::BAD_REQUEST => ChatroomError::InvalidData,
			StatusCode::UNAUTHORIZED => ChatroomError::Unauthorized,
			StatusCode::INTERNAL_SERVER_ERROR => ChatroomError::DatabaseError,
			_ => ChatroomError::Unknown,
			
		}
	}
}
