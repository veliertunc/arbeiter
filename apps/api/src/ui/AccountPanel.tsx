import type { FormEvent } from "react";
import type { User } from "../types";

export function AccountPanel({ user, busy, notice, email, password, setEmail, setPassword, signIn, signOut }: {
  user: User | null;
  busy: boolean;
  notice: string;
  email: string;
  password: string;
  setEmail: (value: string) => void;
  setPassword: (value: string) => void;
  signIn: (event: FormEvent<HTMLFormElement>) => void;
  signOut: () => void;
}) {
  return <article className="panel" id="account-panel">
    <div className="panel-heading"><div><h2>Account access</h2><p>Try the authentication flow against your API.</p></div><span className={`pill ${user ? "ok" : "neutral"}`}>{user ? "Authenticated" : "Not signed in"}</span></div>
    {user ? <div className="account"><div className="avatar large">{user.name?.[0]?.toUpperCase() ?? "U"}</div><div><b>{user.name || "Your account"}</b><p>{user.email}</p><span className="verified">● &nbsp;{user.email_verified ? "Email verified" : "Email not verified"}</span></div><button className="button secondary" onClick={() => void signOut()}>Sign out</button></div> :
      <form className="auth-form" onSubmit={signIn}>
        <label>Email address<input type="email" autoComplete="email" placeholder="you@company.com" value={email} onChange={event => setEmail(event.target.value)} required /></label>
        <label>Password<input type="password" autoComplete="current-password" placeholder="Your password" value={password} onChange={event => setPassword(event.target.value)} required /></label>
        <button className="button primary" disabled={busy}>{busy ? "Connecting…" : "Sign in to workspace →"}</button>
      </form>}
    {notice && <div className="notice">{notice}</div>}
  </article>;
}
