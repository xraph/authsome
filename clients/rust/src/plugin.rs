// Auto-generated plugin trait

use crate::client::AuthsomeClient;
use crate::error::Result;

pub trait ClientPlugin: Send + Sync {
    /// Returns the unique plugin identifier
    fn id(&self) -> &str;
    
    /// Initialize plugin with base client
    fn init(&mut self, client: AuthsomeClient);
}
