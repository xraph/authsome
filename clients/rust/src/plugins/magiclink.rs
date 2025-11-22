// Auto-generated magiclink plugin

use reqwest::Method;
use serde::{Deserialize, Serialize};

use crate::client::AuthsomeClient;
use crate::error::Result;
use crate::plugin::ClientPlugin;
use crate::types::*;

pub struct MagiclinkPlugin {{
    client: Option<AuthsomeClient>,
}

impl MagiclinkPlugin {{
    pub fn new() -> Self {
        Self { client: None }
    }

    #[derive(Debug, Serialize)]
    pub struct SendRequest {
        #[serde(rename = "email")]
        pub email: String,
    }

    pub async fn send(
        &self,
        _request: SendRequest,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

    pub async fn verify(
        &self,
    ) -> Result<()> {
        // TODO: Implement plugin method
        unimplemented!("Plugin methods need client access")
    }

}

impl ClientPlugin for MagiclinkPlugin {{
    fn id(&self) -> &str {
        "magiclink"
    }

    fn init(&mut self, client: AuthsomeClient) {
        self.client = Some(client);
    }
}
