// Auto-generated error types

use thiserror::Error;

#[derive(Debug, Error)]
pub enum AuthsomeError {
    #[error("Network error: {0}")]
    Network(String),
    
    #[error("Validation error: {0}")]
    Validation(String),
    
    #[error("Unauthorized: {0}")]
    Unauthorized(String),
    
    #[error("Forbidden: {0}")]
    Forbidden(String),
    
    #[error("Not found: {0}")]
    NotFound(String),
    
    #[error("Conflict: {0}")]
    Conflict(String),
    
    #[error("Rate limit exceeded: {0}")]
    RateLimit(String),
    
    #[error("Server error: {0}")]
    Server(String),
    
    #[error("API error (status {status}): {message}")]
    Api {
        status: u16,
        message: String,
    },
    
    #[error("Request error: {0}")]
    Request(#[from] reqwest::Error),
    
    #[error("JSON error: {0}")]
    Json(#[from] serde_json::Error),
}

impl AuthsomeError {
    pub fn from_status(status: u16, message: String) -> Self {
        match status {
            400 => Self::Validation(message),
            401 => Self::Unauthorized(message),
            403 => Self::Forbidden(message),
            404 => Self::NotFound(message),
            409 => Self::Conflict(message),
            429 => Self::RateLimit(message),
            500..=599 => Self::Server(message),
            _ => Self::Api { status, message },
        }
    }
}

pub type Result<T> = std::result::Result<T, AuthsomeError>;
