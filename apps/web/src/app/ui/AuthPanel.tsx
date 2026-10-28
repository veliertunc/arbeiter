"use client";
import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { AuthClient, AuthError } from "@arbeiter/auth-sdk";

const auth = new AuthClient({ baseURL: process.env.NEXT_PUBLIC_AUTH_API_URL ?? "http://localhost:8080" });
type User = { email?: string; name?: string; email_verified?: boolean };
export default function AuthPanel() {
  const [user, setUser] = useState<User | null>(null);
  const [email, setEmail] = useState(""); const [password, setPassword] = useState("");
  const [message, setMessage] = useState(""); const [busy, setBusy] = useState(false);
  useEffect(() => { void auth.me().then(value => setUser(value as User)).catch(() => setUser(null)); }, []);
  async function signIn(event: FormEvent) {
    event.preventDefault(); setBusy(true); setMessage("");
    try { await auth.signIn(email, password); setUser(await auth.me() as User); setMessage("You are signed in securely."); }
    catch (error) { setMessage(error instanceof AuthError ? error.message : "Unable to reach the auth API. Check its URL and try again."); }
    finally { setBusy(false); }
  }
  async function signOut() { await auth.signOut(); setUser(null); setMessage("You have signed out."); }
  return <div className="demo-card"><div className="demo-head"><span className="mini-mark">a</span><div><b>{user ? "Your account" : "Welcome back"}</b><small>{user ? "Your session is active" : "Sign in to your workspace"}</small></div><span className="lock">⌑</span></div>{user ? <div className="signed-in"><div className="signed-avatar">{user.name?.[0]?.toUpperCase() ?? "A"}</div><b>{user.name || user.email}</b><small>{user.email}</small><button onClick={() => void signOut()}>Sign out</button></div> : <form onSubmit={signIn}><label>Email address<input type="email" autoComplete="email" placeholder="you@company.com" value={email} onChange={e=>setEmail(e.target.value)} required/></label><label>Password<input type="password" autoComplete="current-password" placeholder="Enter your password" value={password} onChange={e=>setPassword(e.target.value)} required/></label><button className="submit" disabled={busy}>{busy ? "Signing in…" : "Continue securely  →"}</button></form>}{message&&<p className="form-message">{message}</p>}<div className="demo-secure">🔒 &nbsp; Protected with an HttpOnly session cookie</div></div>;
}
