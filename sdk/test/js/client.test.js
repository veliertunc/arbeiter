import test from "node:test";
import assert from "node:assert/strict";
import { AuthClient, AuthError } from "../../js/index.js";

test("sign-in stores the session cookie for Node requests", async () => {
  const calls = [];
  const client = new AuthClient({ baseURL: "https://auth.example.test", fetchImpl: async (url, options) => {
    calls.push({ url, options });
    return calls.length === 1
      ? new Response('{"id":"u1"}', { status: 200, headers: { "content-type": "application/json", "set-cookie": "session=opaque; Path=/; HttpOnly; SameSite=Lax" } })
      : new Response('{"id":"u1","email":"a@example.test"}', { status: 200, headers: { "content-type": "application/json" } });
  } });
  await client.signIn("a@example.test", "long-password");
  const user = await client.me();
  assert.equal(user.email, "a@example.test");
  assert.equal(calls[0].url, "https://auth.example.test/v1/sign-in");
  assert.equal(calls[1].options.headers.Cookie, "session=opaque");
  assert.equal(calls[1].options.credentials, "include");
});

test("API errors retain status and payload", async () => {
  const client = new AuthClient({ baseURL: "https://auth.example.test", fetchImpl: async () => new Response('{"message":"authentication required"}', { status: 401 }) });
  await assert.rejects(client.me(), (error) => error instanceof AuthError && error.status === 401 && error.message === "authentication required");
});
