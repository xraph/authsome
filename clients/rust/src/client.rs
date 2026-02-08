// Auto-generated AuthSome client

use reqwest::{Client as HttpClient, Method, RequestBuilder};
use serde::{de::DeserializeOwned, Serialize};
use std::collections::HashMap;
use std::sync::Arc;

use crate::error::{AuthsomeError, Result};
use crate::plugin::ClientPlugin;
use crate::types::*;

#[derive(Clone)]
pub struct AuthsomeClient {
    base_url: String,
    http_client: HttpClient,
    token: Option<String>,
    headers: HashMap<String, String>,
}

impl AuthsomeClient {
    pub fn builder() -> AuthsomeClientBuilder {
        AuthsomeClientBuilder::default()
    }

    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
            http_client: HttpClient::new(),
            token: None,
            headers: HashMap::new(),
        }
    }

    pub fn set_token(&mut self, token: String) {
        self.token = Some(token);
    }

    async fn request<T: DeserializeOwned>(
        &self,
        method: Method,
        path: &str,
        body: Option<impl Serialize>,
        auth: bool,
    ) -> Result<T> {
        let url = format!("{}{}", self.base_url, path);
        let mut req = self.http_client.request(method, &url);

        req = req.header("Content-Type", "application/json");

        for (key, value) in &self.headers {
            req = req.header(key, value);
        }

        if auth {
            if let Some(token) = &self.token {
                req = req.bearer_auth(token);
            }
        }

        if let Some(body) = body {
            req = req.json(&body);
        }

        let resp = req.send().await?;
        let status = resp.status();

        if !status.is_success() {
            let error_body: serde_json::Value = resp.json().await.unwrap_or_default();
            let message = error_body["error"].as_str()
                .or_else(|| error_body["message"].as_str())
                .unwrap_or("Request failed")
                .to_string();
            return Err(AuthsomeError::from_status(status.as_u16(), message));
        }

        Ok(resp.json().await?)
    }

    /// Request for sign_up
    #[derive(Debug, Serialize)]
    pub struct SignUpRequest {
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "password")]
        pub password: String,
        #[serde(rename = "name", skip_serializing_if = "Option::is_none")]
        pub name: Option<String>,
    }

    /// Response for sign_up
    #[derive(Debug, Deserialize)]
    pub struct SignUpResponse {
        #[serde(rename = "session")]
        pub session: Session,
        #[serde(rename = "user")]
        pub user: User,
    }

    /// Create a new user account
    pub async fn sign_up(
        &self,
        request: SignUpRequest,
    ) -> Result<SignUpResponse> {
        let path = "/api/auth/signup";
        self.request(
            Method::POST,
            &path,
            Some(request),
            false,
        ).await
    }

    /// Request for sign_in
    #[derive(Debug, Serialize)]
    pub struct SignInRequest {
        #[serde(rename = "email")]
        pub email: String,
        #[serde(rename = "password")]
        pub password: String,
    }

    /// Response for sign_in
    #[derive(Debug, Deserialize)]
    pub struct SignInResponse {
        #[serde(rename = "user")]
        pub user: User,
        #[serde(rename = "session")]
        pub session: Session,
        #[serde(rename = "requiresTwoFactor")]
        pub requires_two_factor: bool,
    }

    /// Sign in with email and password
    pub async fn sign_in(
        &self,
        request: SignInRequest,
    ) -> Result<SignInResponse> {
        let path = "/api/auth/signin";
        self.request(
            Method::POST,
            &path,
            Some(request),
            false,
        ).await
    }

    /// Response for sign_out
    #[derive(Debug, Deserialize)]
    pub struct SignOutResponse {
        #[serde(rename = "success")]
        pub success: bool,
    }

    /// Sign out and invalidate session
    pub async fn sign_out(
        &self,
    ) -> Result<SignOutResponse> {
        let path = "/api/auth/signout";
        self.request(
            Method::POST,
            &path,
            None::<()>,
            true,
        ).await
    }

    /// Response for get_session
    #[derive(Debug, Deserialize)]
    pub struct GetSessionResponse {
        #[serde(rename = "session")]
        pub session: Session,
        #[serde(rename = "user")]
        pub user: User,
    }

    /// Get current session information
    pub async fn get_session(
        &self,
    ) -> Result<GetSessionResponse> {
        let path = "/api/auth/session";
        self.request(
            Method::GET,
            &path,
            None::<()>,
            true,
        ).await
    }

    /// Request for update_user
    #[derive(Debug, Serialize)]
    pub struct UpdateUserRequest {
        #[serde(rename = "name", skip_serializing_if = "Option::is_none")]
        pub name: Option<String>,
        #[serde(rename = "email", skip_serializing_if = "Option::is_none")]
        pub email: Option<String>,
    }

    /// Response for update_user
    #[derive(Debug, Deserialize)]
    pub struct UpdateUserResponse {
        #[serde(rename = "user")]
        pub user: User,
    }

    /// Update current user profile
    pub async fn update_user(
        &self,
        request: UpdateUserRequest,
    ) -> Result<UpdateUserResponse> {
        let path = "/api/auth/user/update";
        self.request(
            Method::POST,
            &path,
            Some(request),
            true,
        ).await
    }

    /// Response for list_devices
    #[derive(Debug, Deserialize)]
    pub struct ListDevicesResponse {
        #[serde(rename = "devices")]
        pub devices: Vec<Device>,
    }

    /// List user devices
    pub async fn list_devices(
        &self,
    ) -> Result<ListDevicesResponse> {
        let path = "/api/auth/devices";
        self.request(
            Method::GET,
            &path,
            None::<()>,
            true,
        ).await
    }

    /// Request for revoke_device
    #[derive(Debug, Serialize)]
    pub struct RevokeDeviceRequest {
        #[serde(rename = "fingerprint")]
        pub fingerprint: String,
    }

    /// Response for revoke_device
    #[derive(Debug, Deserialize)]
    pub struct RevokeDeviceResponse {
        #[serde(rename = "status")]
        pub status: String,
    }

    /// Revoke a device
    pub async fn revoke_device(
        &self,
        request: RevokeDeviceRequest,
    ) -> Result<RevokeDeviceResponse> {
        let path = "/api/auth/devices/revoke";
        self.request(
            Method::POST,
            &path,
            Some(request),
            true,
        ).await
    }

    /// Request for refresh_session
    #[derive(Debug, Serialize)]
    pub struct RefreshSessionRequest {
        #[serde(rename = "refreshToken")]
        pub refresh_token: String,
    }

    /// Response for refresh_session
    #[derive(Debug, Deserialize)]
    pub struct RefreshSessionResponse {
        #[serde(rename = "refreshExpiresAt")]
        pub refresh_expires_at: String,
        #[serde(rename = "session")]
        pub session: ,
        #[serde(rename = "accessToken")]
        pub access_token: String,
        #[serde(rename = "refreshToken")]
        pub refresh_token: String,
        #[serde(rename = "expiresAt")]
        pub expires_at: String,
    }

    /// Refresh access token using refresh token
    pub async fn refresh_session(
        &self,
        request: RefreshSessionRequest,
    ) -> Result<RefreshSessionResponse> {
        let path = "/api/auth/refresh";
        self.request(
            Method::POST,
            &path,
            Some(request),
            false,
        ).await
    }

    /// Request for request_password_reset
    #[derive(Debug, Serialize)]
    pub struct RequestPasswordResetRequest {
        #[serde(rename = "email")]
        pub email: String,
    }

    /// Response for request_password_reset
    #[derive(Debug, Deserialize)]
    pub struct RequestPasswordResetResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// Request password reset link
    pub async fn request_password_reset(
        &self,
        request: RequestPasswordResetRequest,
    ) -> Result<RequestPasswordResetResponse> {
        let path = "/api/auth/password/reset/request";
        self.request(
            Method::POST,
            &path,
            Some(request),
            false,
        ).await
    }

    /// Request for reset_password
    #[derive(Debug, Serialize)]
    pub struct ResetPasswordRequest {
        #[serde(rename = "token")]
        pub token: String,
        #[serde(rename = "newPassword")]
        pub new_password: String,
    }

    /// Response for reset_password
    #[derive(Debug, Deserialize)]
    pub struct ResetPasswordResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// Reset password using token
    pub async fn reset_password(
        &self,
        request: ResetPasswordRequest,
    ) -> Result<ResetPasswordResponse> {
        let path = "/api/auth/password/reset/confirm";
        self.request(
            Method::POST,
            &path,
            Some(request),
            false,
        ).await
    }

    /// Response for validate_reset_token
    #[derive(Debug, Deserialize)]
    pub struct ValidateResetTokenResponse {
        #[serde(rename = "valid")]
        pub valid: bool,
    }

    /// Validate password reset token
    pub async fn validate_reset_token(
        &self,
    ) -> Result<ValidateResetTokenResponse> {
        let path = "/api/auth/password/reset/validate";
        self.request(
            Method::GET,
            &path,
            None::<()>,
            false,
        ).await
    }

    /// Request for change_password
    #[derive(Debug, Serialize)]
    pub struct ChangePasswordRequest {
        #[serde(rename = "oldPassword")]
        pub old_password: String,
        #[serde(rename = "newPassword")]
        pub new_password: String,
    }

    /// Response for change_password
    #[derive(Debug, Deserialize)]
    pub struct ChangePasswordResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// Change password for authenticated user
    pub async fn change_password(
        &self,
        request: ChangePasswordRequest,
    ) -> Result<ChangePasswordResponse> {
        let path = "/api/auth/password/change";
        self.request(
            Method::POST,
            &path,
            Some(request),
            true,
        ).await
    }

    /// Request for request_email_change
    #[derive(Debug, Serialize)]
    pub struct RequestEmailChangeRequest {
        #[serde(rename = "newEmail")]
        pub new_email: String,
    }

    /// Response for request_email_change
    #[derive(Debug, Deserialize)]
    pub struct RequestEmailChangeResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// Request email address change
    pub async fn request_email_change(
        &self,
        request: RequestEmailChangeRequest,
    ) -> Result<RequestEmailChangeResponse> {
        let path = "/api/auth/email/change/request";
        self.request(
            Method::POST,
            &path,
            Some(request),
            true,
        ).await
    }

    /// Request for confirm_email_change
    #[derive(Debug, Serialize)]
    pub struct ConfirmEmailChangeRequest {
        #[serde(rename = "token")]
        pub token: String,
    }

    /// Response for confirm_email_change
    #[derive(Debug, Deserialize)]
    pub struct ConfirmEmailChangeResponse {
        #[serde(rename = "message")]
        pub message: String,
    }

    /// Confirm email change using token
    pub async fn confirm_email_change(
        &self,
        request: ConfirmEmailChangeRequest,
    ) -> Result<ConfirmEmailChangeResponse> {
        let path = "/api/auth/email/change/confirm";
        self.request(
            Method::POST,
            &path,
            Some(request),
            false,
        ).await
    }

}

#[derive(Default)]
pub struct AuthsomeClientBuilder {
    base_url: Option<String>,
    http_client: Option<HttpClient>,
    token: Option<String>,
    headers: HashMap<String, String>,
}

impl AuthsomeClientBuilder {
    pub fn base_url(mut self, url: impl Into<String>) -> Self {
        self.base_url = Some(url.into());
        self
    }

    pub fn http_client(mut self, client: HttpClient) -> Self {
        self.http_client = Some(client);
        self
    }

    pub fn token(mut self, token: impl Into<String>) -> Self {
        self.token = Some(token.into());
        self
    }

    pub fn header(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {
        self.headers.insert(key.into(), value.into());
        self
    }

    pub fn build(self) -> Result<AuthsomeClient> {
        let base_url = self.base_url.ok_or_else(|| {
            AuthsomeError::Validation("base_url is required".to_string())
        })?;

        Ok(AuthsomeClient {
            base_url,
            http_client: self.http_client.unwrap_or_else(HttpClient::new),
            token: self.token,
            headers: self.headers,
        })
    }
}
