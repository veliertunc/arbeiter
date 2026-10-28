import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { trpc } from "../trpc";

type User = { id?: string; email?: string; name?: string; email_verified?: boolean };
type Org = { id?: string; name?: string; slug?: string };

export function Console() {
  const [user, setUser] = useState<User | null>(null);
  const [organizations, setOrganizations] = useState<Org[]>([]);
  const [sessions, setSessions] = useState<unknown[]>([]);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [notice, setNotice] = useState("");
  const [tab, setTab] = useState("Overview");
  const [busy, setBusy] = useState(false);

  async function refresh() {
    try {
      const data = await trpc.auth.me.query() as User;
      setUser(data);
      const [orgs, activeSessions] = await Promise.all([trpc.organizations.list.query(), trpc.sessions.query()]);
      setOrganizations(Array.isArray(orgs) ? orgs as unknown as Org[] : []);
      setSessions(Array.isArray(activeSessions) ? activeSessions as unknown[] : []);
    } catch { setUser(null); }
  }
  useEffect(() => { void refresh(); }, []);

  async function signIn(event: FormEvent) {
    event.preventDefault(); setBusy(true); setNotice("");
    try { await trpc.auth.signIn.mutate({ email, password }); await refresh(); setNotice("Signed in. Your session is ready."); }
    catch (error) { setNotice(error instanceof Error ? error.message : "Could not sign in."); }
    finally { setBusy(false); }
  }
  async function signOut() { await trpc.auth.signOut.mutate(); setUser(null); setNotice("Session ended."); }

  return <div className="shell">
    <aside className="sidebar"><a className="brand" href="#"><span className="brand-mark">a</span> arbeiter<span className="brand-small">AUTH</span></a><div className="project"><span className="project-icon">A</span><span><b>Acme Studio</b><small>Development</small></span><span className="chevron">⌄</span></div><div className="side-label">WORKSPACE</div>
      {[["Overview","▦"],["Profile","◉"],["Organizations","⌂"],["Sessions","◷"],["API guide","⌘"]].map(([label, icon]) => <button key={label} className={`nav-item ${tab === label ? "active" : ""}`} onClick={() => { if (label === "API guide") { window.open("http://localhost:4322", "_blank", "noopener,noreferrer"); return; } setTab(label); document.getElementById(label.toLowerCase())?.scrollIntoView({ behavior: "smooth" }); }}><span>{icon}</span>{label}</button>)}
      <div className="sidebar-bottom"><div className="help-card"><span className="spark">✳</span><b>Ship auth faster</b><p>Explore the API reference and SDK guides.</p><a href="http://localhost:4322" target="_blank">Open documentation ↗</a></div><div className="profile"><div className="avatar">{user?.name?.[0]?.toUpperCase() ?? "D"}</div><span><b>{user?.name ?? "Developer"}</b><small>{user?.email ?? "Local workspace"}</small></span><button aria-label="Profile options">•••</button></div></div>
    </aside>
    <main className="main"><header className="topbar"><div className="crumb">Workspace <span>/</span> <b>{tab}</b></div><div className="top-actions"><span className="status-dot"/> All systems operational <button className="icon-button">?</button></div></header>
      <section className="content" id="overview"><div className="eyebrow">WEDNESDAY, OCTOBER 28</div><div className="title-row"><div><h1>{tab}</h1><p className="lead">Your authentication workspace at a glance.</p></div><button className="button secondary" onClick={() => void refresh()}>↻ &nbsp;Refresh data</button></div>
        <div className="banner"><div className="banner-icon">✦</div><div><b>Welcome to your authentication workspace</b><p>Manage users, sessions, and organizations from one secure place.</p></div><a href="http://localhost:4322" target="_blank">Read the quickstart ↗</a></div>
        <div className="stats"><article className="stat-card" id="profile"><div className="stat-head"><span>USERS</span><span className="stat-icon purple">◉</span></div><strong>{user ? "1" : "—"}</strong><small>{user ? "Current signed-in user" : "Sign in to load user data"}</small><div className="sparkline purple-line">▁▂▂▃▃▄▅▆▆▇</div></article><article className="stat-card" id="organizations"><div className="stat-head"><span>ORGANIZATIONS</span><span className="stat-icon blue">⌂</span></div><strong>{user ? organizations.length : "—"}</strong><small>Across your workspace</small><div className="sparkline blue-line">▁▁▂▃▂▄▅▅▆▇</div></article><article className="stat-card" id="sessions"><div className="stat-head"><span>ACTIVE SESSIONS</span><span className="stat-icon green">◷</span></div><strong>{user ? sessions.length : "—"}</strong><small>Secure cookie sessions</small><div className="sparkline green-line">▁▂▃▂▄▃▅▆▆▇</div></article></div>
        <div className="columns"><article className="panel"><div className="panel-heading"><div><h2>Account access</h2><p>Try the authentication flow against your API.</p></div><span className={`pill ${user ? "ok" : "neutral"}`}>{user ? "Authenticated" : "Not signed in"}</span></div>{user ? <div className="account"><div className="avatar large">{user.name?.[0]?.toUpperCase() ?? "U"}</div><div><b>{user.name || "Your account"}</b><p>{user.email}</p><span className="verified">● &nbsp;{user.email_verified ? "Email verified" : "Email not verified"}</span></div><button className="button secondary" onClick={() => void signOut()}>Sign out</button></div> : <form className="auth-form" onSubmit={signIn}><label>Email address<input type="email" autoComplete="email" placeholder="you@company.com" value={email} onChange={e => setEmail(e.target.value)} required /></label><label>Password<input type="password" autoComplete="current-password" placeholder="Your password" value={password} onChange={e => setPassword(e.target.value)} required /></label><button className="button primary" disabled={busy}>{busy ? "Connecting…" : "Sign in to workspace →"}</button></form>}{notice && <div className="notice">{notice}</div>}</article>
          <article className="panel"><div className="panel-heading"><div><h2>Organizations</h2><p>Your connected teams and workspaces.</p></div><a className="text-link" href="http://localhost:4322" target="_blank" rel="noreferrer">API guide ↗</a></div>{organizations.length ? <div className="org-list">{organizations.map((org,i) => <div className="org-row" key={org.id ?? i}><span className="org-logo">{org.name?.[0] ?? "O"}</span><span><b>{org.name}</b><small>/{org.slug}</small></span><span className="role">Owner</span></div>)}</div> : <div className="empty"><span>⌂</span><b>{user ? "No organizations yet" : "Sign in to view organizations"}</b><p>Your organizations will appear here.</p></div>}</article></div>
        <div className="footnote"><span>🔒 &nbsp; Sessions are stored as secure, HTTP-only cookies.</span><span>Powered by Arbeiter Auth API</span></div>
      </section>
    </main>
  </div>;
}
