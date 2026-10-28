export type AuthResult = unknown;

export class AuthError extends Error {
  status: number;
  payload?: AuthResult;
  constructor(status: number, message?: string, payload?: AuthResult);
}

export interface AuthClientOptions {
  baseURL: string;
  fetchImpl?: typeof fetch;
  headers?: Record<string, string>;
}

/** Cookie-session client for the Arbeiter Auth API. Reuse one instance per browser session. */
export class AuthClient {
  constructor(options: AuthClientOptions);
  request(method: string, path: string, body?: unknown): Promise<AuthResult | undefined>;
  signUp(email: string, password: string, name?: string): Promise<AuthResult | undefined>;
  signIn(email: string, password: string): Promise<AuthResult | undefined>;
  signOut(): Promise<AuthResult | undefined>;
  me(): Promise<AuthResult | undefined>;
  updateMe(name: string): Promise<AuthResult | undefined>;
  deleteAccount(password: string): Promise<AuthResult | undefined>;
  listSessions(): Promise<AuthResult | undefined>;
  revokeSession(id: string): Promise<AuthResult | undefined>;
  requestEmailVerification(): Promise<AuthResult | undefined>;
  verifyEmail(token: string): Promise<AuthResult | undefined>;
  requestPasswordReset(email: string): Promise<AuthResult | undefined>;
  resetPassword(token: string, password: string): Promise<AuthResult | undefined>;
  listOrganizations(): Promise<AuthResult | undefined>;
  createOrganization(name: string, slug: string): Promise<AuthResult | undefined>;
  deleteOrganization(id: string): Promise<AuthResult | undefined>;
  listMembers(orgID: string): Promise<AuthResult | undefined>;
  upsertMember(orgID: string, userID: string, role: string): Promise<AuthResult | undefined>;
  deleteMember(orgID: string, userID: string): Promise<AuthResult | undefined>;
}
