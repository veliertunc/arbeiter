"use client";
import { useAuthPanel } from "../../hooks/useAuthPanel";

export default function AuthPanel() {
  const auth = useAuthPanel();

  return <div className="demo-card">
    <div className="demo-head">
      <span className="mini-mark">a</span>
      <div><b>{auth.user ? "Your account" : "Welcome back"}</b><small>{auth.user ? "Your session is active" : "Sign in to your workspace"}</small></div>
      <span className="lock">⌑</span>
    </div>
    {auth.user ? <div className="signed-in">
      <div className="signed-avatar">{auth.user.name?.[0]?.toUpperCase() ?? "A"}</div>
      <b>{auth.user.name || auth.user.email}</b><small>{auth.user.email}</small>
      <button onClick={() => void auth.signOut()}>Sign out</button>
    </div> : <form onSubmit={auth.signIn}>
      <label>Email address<input type="email" autoComplete="email" placeholder="you@company.com" value={auth.email} onChange={event => auth.setEmail(event.target.value)} required/></label>
      <label>Password<input type="password" autoComplete="current-password" placeholder="Enter your password" value={auth.password} onChange={event => auth.setPassword(event.target.value)} required/></label>
      <button className="submit" disabled={auth.busy}>{auth.busy ? "Signing in…" : "Continue securely  →"}</button>
    </form>}
    {auth.message && <p className="form-message">{auth.message}</p>}
    <div className="demo-secure">🔒 &nbsp; Protected with an HttpOnly session cookie</div>
  </div>;
}
