export class AuthError extends Error {
  constructor(status, message, payload = undefined) {
    super(message || `Authentication API returned HTTP ${status}`);
    this.name = "AuthError";
    this.status = status;
    this.payload = payload;
  }
}

/** Cookie-based client. Browsers manage HttpOnly cookies; Node keeps the session cookie in memory. */
export class AuthClient {
  constructor({ baseURL, fetchImpl = globalThis.fetch, headers = {} }) {
    if (!baseURL) throw new TypeError("baseURL is required");
    if (typeof fetchImpl !== "function") throw new TypeError("fetch is required");
    this.baseURL = baseURL.replace(/\/+$/, "");
    this.fetch = fetchImpl;
    this.headers = { ...headers };
    this.sessionCookie = undefined;
  }

  async request(method, path, body = undefined) {
    const headers = { Accept: "application/json", ...this.headers };
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (this.sessionCookie && typeof window === "undefined") headers.Cookie = this.sessionCookie;
    const response = await this.fetch(`${this.baseURL}${path}`, {
      method,
      headers,
      credentials: "include",
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });

    if (typeof window === "undefined") {
      const setCookies = response.headers.getSetCookie?.() ?? [response.headers.get("set-cookie")].filter(Boolean);
      for (const value of setCookies) {
        const pair = value.split(";", 1)[0];
        if (pair.startsWith("session=")) this.sessionCookie = pair;
      }
    }
    if (response.status === 204) return undefined;
    const text = await response.text();
    let payload;
    try { payload = text ? JSON.parse(text) : undefined; } catch { payload = { message: text }; }
    if (!response.ok) throw new AuthError(response.status, payload?.message, payload);
    return payload;
  }

  signUp(email, password, name = undefined) { return this.request("POST", "/v1/sign-up", { email, password, ...(name === undefined ? {} : { name }) }); }
  signIn(email, password) { return this.request("POST", "/v1/sign-in", { email, password }); }
  signOut() { return this.request("POST", "/v1/sign-out", {}); }
  me() { return this.request("GET", "/v1/me"); }
  updateMe(name) { return this.request("PATCH", "/v1/me", { name }); }
  deleteAccount(password) { return this.request("DELETE", "/v1/me", { password }); }
  listSessions() { return this.request("GET", "/v1/sessions"); }
  revokeSession(id) { return this.request("DELETE", `/v1/sessions/${encodeURIComponent(id)}`); }
  requestEmailVerification() { return this.request("POST", "/v1/email/verification", {}); }
  verifyEmail(token) { return this.request("POST", "/v1/email/verification/consume", { token }); }
  requestPasswordReset(email) { return this.request("POST", "/v1/password/reset", { email }); }
  resetPassword(token, password) { return this.request("POST", "/v1/password/reset/consume", { token, password }); }
  listOrganizations() { return this.request("GET", "/v1/organizations"); }
  createOrganization(name, slug) { return this.request("POST", "/v1/organizations", { name, slug }); }
  deleteOrganization(id) { return this.request("DELETE", `/v1/organizations/${encodeURIComponent(id)}`); }
  listMembers(orgID) { return this.request("GET", `/v1/organizations/${encodeURIComponent(orgID)}/members`); }
  upsertMember(orgID, userID, role) { return this.request("PUT", `/v1/organizations/${encodeURIComponent(orgID)}/members/${encodeURIComponent(userID)}`, { role }); }
  deleteMember(orgID, userID) { return this.request("DELETE", `/v1/organizations/${encodeURIComponent(orgID)}/members/${encodeURIComponent(userID)}`); }
}
