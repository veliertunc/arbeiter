import io
import json
from email.message import Message
from pathlib import Path
import sys
import unittest
from unittest.mock import patch
from urllib.request import AbstractHTTPHandler, HTTPCookieProcessor, build_opener
from http.cookiejar import CookieJar

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / ".." / "py"))
from arbeiter_auth import AuthClient, AuthError


class MockResponse(io.BytesIO):
    def __init__(self, body, headers, url, status):
        super().__init__(body)
        self.headers = headers
        self.url = url
        self.code = status
        self.status = status
        self.msg = "OK" if status < 400 else "Unauthorized"

    def info(self): return self.headers
    def geturl(self): return self.url


class ClientTests(unittest.TestCase):
    def setUp(self):
        self.requests = []
        self.opener = build_opener(HTTPCookieProcessor(CookieJar()))

    def fake_open(self, _http_class, request, *_args, **_kwargs):
        self.requests.append(request)
        if request.full_url.endswith("/v1/sign-in"):
            status, payload = 200, {"id": "user-1"}
        elif request.full_url.endswith("/v1/me") and request.get_header("Cookie") == "session=test-session":
            status, payload = 200, {"email": "user@example.test"}
        elif request.full_url.endswith("/v1/me"):
            status, payload = 401, {"message": "authentication required"}
        else:
            status, payload = 200, {"message": "ok"}
        headers = Message()
        headers["Content-Type"] = "application/json"
        if request.full_url.endswith("/v1/sign-in"):
            headers["Set-Cookie"] = "session=test-session; Path=/; HttpOnly; SameSite=Lax"
        body = json.dumps(payload).encode()
        return MockResponse(body, headers, request.full_url, status)

    def test_session_cookie_is_reused(self):
        client = AuthClient("https://auth.example.test", opener=self.opener)
        with patch.object(AbstractHTTPHandler, "do_open", self.fake_open):
            client.sign_in("user@example.test", "password")
            self.assertEqual(client.me()["email"], "user@example.test")
        self.assertEqual(self.requests[1].get_header("Cookie"), "session=test-session")

    def test_errors_include_status_and_payload(self):
        client = AuthClient("https://auth.example.test", opener=self.opener)
        with patch.object(AbstractHTTPHandler, "do_open", self.fake_open):
            with self.assertRaises(AuthError) as raised:
                client.me()
        self.assertEqual(raised.exception.status, 401)
        self.assertEqual(raised.exception.payload["message"], "authentication required")


if __name__ == "__main__":
    unittest.main()
