use reqwest::blocking::Client as HttpClient;
use reqwest::Method;
use serde_json::{json, Value};
use std::time::Duration;

#[derive(Debug)]
pub struct AuthError {
    pub status: u16,
    pub message: String,
    pub body: String,
}

impl std::fmt::Display for AuthError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} (HTTP {})", self.message, self.status)
    }
}

impl std::error::Error for AuthError {}

pub struct AuthClient {
    base_url: String,
    http: HttpClient,
}

impl AuthClient {
    pub fn new(base_url: impl Into<String>) -> Result<Self, reqwest::Error> {
        Self::with_timeout(base_url, Duration::from_secs(15))
    }

    pub fn with_timeout(base_url: impl Into<String>, timeout: Duration) -> Result<Self, reqwest::Error> {
        let http = HttpClient::builder().cookie_store(true).timeout(timeout).build()?;
        Ok(Self { base_url: base_url.into().trim_end_matches('/').to_owned(), http })
    }

    fn request(&self, method: Method, path: &str, body: Option<Value>) -> Result<Option<Value>, AuthError> {
        let mut request = self.http.request(method, format!("{}{}", self.base_url, path));
        if let Some(body) = body { request = request.json(&body); }
        let response = request.send().map_err(|e| AuthError { status: 0, message: e.to_string(), body: String::new() })?;
        let status = response.status().as_u16();
        let text = response.text().unwrap_or_default();
        if status == 204 { return Ok(None); }
        let value: Value = if text.is_empty() { Value::Null } else { serde_json::from_str(&text).unwrap_or_else(|_| json!({"message": text})) };
        if !(200..300).contains(&status) {
            return Err(AuthError { status, message: value.get("message").and_then(Value::as_str).unwrap_or("request failed").to_owned(), body: text });
        }
        Ok(Some(value))
    }

    pub fn sign_up(&self, email: &str, password: &str, name: Option<&str>) -> Result<Option<Value>, AuthError> {
        let mut body = json!({"email": email, "password": password});
        if let Some(name) = name { body["name"] = json!(name); }
        self.request(Method::POST, "/v1/sign-up", Some(body))
    }
    pub fn sign_in(&self, email: &str, password: &str) -> Result<Option<Value>, AuthError> { self.request(Method::POST, "/v1/sign-in", Some(json!({"email":email,"password":password}))) }
    pub fn sign_out(&self) -> Result<Option<Value>, AuthError> { self.request(Method::POST, "/v1/sign-out", Some(json!({}))) }
    pub fn me(&self) -> Result<Option<Value>, AuthError> { self.request(Method::GET, "/v1/me", None) }
    pub fn update_me(&self, name: &str) -> Result<Option<Value>, AuthError> { self.request(Method::PATCH, "/v1/me", Some(json!({"name":name}))) }
    pub fn delete_account(&self, password: &str) -> Result<Option<Value>, AuthError> { self.request(Method::DELETE, "/v1/me", Some(json!({"password":password}))) }
    pub fn list_sessions(&self) -> Result<Option<Value>, AuthError> { self.request(Method::GET, "/v1/sessions", None) }
    pub fn revoke_session(&self, id: &str) -> Result<Option<Value>, AuthError> { self.request(Method::DELETE, &format!("/v1/sessions/{}", segment(id)), None) }
    pub fn request_email_verification(&self) -> Result<Option<Value>, AuthError> { self.request(Method::POST, "/v1/email/verification", Some(json!({}))) }
    pub fn verify_email(&self, token: &str) -> Result<Option<Value>, AuthError> { self.request(Method::POST, "/v1/email/verification/consume", Some(json!({"token":token}))) }
    pub fn request_password_reset(&self, email: &str) -> Result<Option<Value>, AuthError> { self.request(Method::POST, "/v1/password/reset", Some(json!({"email":email}))) }
    pub fn reset_password(&self, token: &str, password: &str) -> Result<Option<Value>, AuthError> { self.request(Method::POST, "/v1/password/reset/consume", Some(json!({"token":token,"password":password}))) }
    pub fn list_organizations(&self) -> Result<Option<Value>, AuthError> { self.request(Method::GET, "/v1/organizations", None) }
    pub fn create_organization(&self, name: &str, slug: &str) -> Result<Option<Value>, AuthError> { self.request(Method::POST, "/v1/organizations", Some(json!({"name":name,"slug":slug}))) }
    pub fn delete_organization(&self, id: &str) -> Result<Option<Value>, AuthError> { self.request(Method::DELETE, &format!("/v1/organizations/{}", segment(id)), None) }
    pub fn list_members(&self, org_id: &str) -> Result<Option<Value>, AuthError> { self.request(Method::GET, &format!("/v1/organizations/{}/members", segment(org_id)), None) }
    pub fn upsert_member(&self, org_id: &str, user_id: &str, role: &str) -> Result<Option<Value>, AuthError> { self.request(Method::PUT, &format!("/v1/organizations/{}/members/{}",segment(org_id),segment(user_id)),Some(json!({"role":role}))) }
    pub fn delete_member(&self, org_id: &str, user_id: &str) -> Result<Option<Value>, AuthError> { self.request(Method::DELETE, &format!("/v1/organizations/{}/members/{}",segment(org_id),segment(user_id)),None) }
}

fn segment(value: &str) -> String {
    let mut out = String::new();
    for byte in value.bytes() {
        if byte.is_ascii_alphanumeric() || b"-_.~".contains(&byte) { out.push(byte as char); }
        else { out.push_str(&format!("%{byte:02X}")); }
    }
    out
}
