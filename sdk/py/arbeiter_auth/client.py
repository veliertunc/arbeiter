"""Small standard-library client for the Arbeiter authentication API."""

from __future__ import annotations

import http.cookiejar
import json
from urllib.error import HTTPError, URLError
from urllib.request import HTTPCookieProcessor, Request, build_opener


class AuthError(Exception):
    def __init__(self, status: int, message: str, payload=None):
        super().__init__(message or f"Authentication API returned HTTP {status}")
        self.status = status
        self.payload = payload


class AuthClient:
    """Cookie-session API client. Reuse one instance to retain the session cookie."""

    def __init__(self, base_url: str, *, opener=None, timeout: float = 15.0):
        if not base_url:
            raise ValueError("base_url is required")
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.opener = opener or build_opener(HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def _request(self, method: str, path: str, body=None):
        data = None if body is None else json.dumps(body).encode("utf-8")
        headers = {"Accept": "application/json"}
        if data is not None:
            headers["Content-Type"] = "application/json"
        request = Request(self.base_url + path, data=data, headers=headers, method=method)
        try:
            with self.opener.open(request, timeout=self.timeout) as response:
                if response.status == 204:
                    return None
                raw = response.read()
                return json.loads(raw) if raw else None
        except HTTPError as error:
            raw = error.read()
            try:
                payload = json.loads(raw) if raw else None
            except json.JSONDecodeError:
                payload = {"message": raw.decode("utf-8", errors="replace")}
            raise AuthError(error.code, (payload or {}).get("message", ""), payload) from error
        except URLError as error:
            raise ConnectionError(f"authentication API request failed: {error.reason}") from error

    def sign_up(self, email: str, password: str, name: str | None = None):
        body = {"email": email, "password": password}
        if name is not None:
            body["name"] = name
        return self._request("POST", "/v1/sign-up", body)

    def sign_in(self, email: str, password: str): return self._request("POST", "/v1/sign-in", {"email": email, "password": password})
    def sign_out(self): return self._request("POST", "/v1/sign-out", {})
    def me(self): return self._request("GET", "/v1/me")
    def update_me(self, name: str): return self._request("PATCH", "/v1/me", {"name": name})
    def delete_account(self, password: str): return self._request("DELETE", "/v1/me", {"password": password})
    def list_sessions(self): return self._request("GET", "/v1/sessions")
    def revoke_session(self, session_id: str): return self._request("DELETE", f"/v1/sessions/{_segment(session_id)}")
    def request_email_verification(self): return self._request("POST", "/v1/email/verification", {})
    def verify_email(self, token: str): return self._request("POST", "/v1/email/verification/consume", {"token": token})
    def request_password_reset(self, email: str): return self._request("POST", "/v1/password/reset", {"email": email})
    def reset_password(self, token: str, password: str): return self._request("POST", "/v1/password/reset/consume", {"token": token, "password": password})
    def list_organizations(self): return self._request("GET", "/v1/organizations")
    def create_organization(self, name: str, slug: str): return self._request("POST", "/v1/organizations", {"name": name, "slug": slug})
    def delete_organization(self, organization_id: str): return self._request("DELETE", f"/v1/organizations/{_segment(organization_id)}")
    def list_members(self, organization_id: str): return self._request("GET", f"/v1/organizations/{_segment(organization_id)}/members")
    def upsert_member(self, organization_id: str, user_id: str, role: str): return self._request("PUT", f"/v1/organizations/{_segment(organization_id)}/members/{_segment(user_id)}", {"role": role})
    def delete_member(self, organization_id: str, user_id: str): return self._request("DELETE", f"/v1/organizations/{_segment(organization_id)}/members/{_segment(user_id)}")


def _segment(value: str) -> str:
    from urllib.parse import quote

    return quote(value, safe="")
