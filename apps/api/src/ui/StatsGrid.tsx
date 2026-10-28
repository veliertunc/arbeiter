import type { Organization } from "../types";

export function StatsGrid({ user, organizations, sessions }: {
  user: unknown;
  organizations: Organization[];
  sessions: unknown[];
}) {
  return <div className="stats">
    <article className="stat-card" id="profile"><div className="stat-head"><span>USERS</span><span className="stat-icon purple">◉</span></div><strong>{user ? "1" : "—"}</strong><small>{user ? "Current signed-in user" : "Sign in to load user data"}</small><div className="sparkline purple-line">▁▂▂▃▃▄▅▆▆▇</div></article>
    <article className="stat-card" id="organizations"><div className="stat-head"><span>ORGANIZATIONS</span><span className="stat-icon blue">⌂</span></div><strong>{user ? organizations.length : "—"}</strong><small>Across your workspace</small><div className="sparkline blue-line">▁▁▂▃▂▄▅▅▆▇</div></article>
    <article className="stat-card" id="sessions"><div className="stat-head"><span>ACTIVE SESSIONS</span><span className="stat-icon green">◷</span></div><strong>{user ? sessions.length : "—"}</strong><small>Secure cookie sessions</small><div className="sparkline green-line">▁▂▃▂▄▃▅▆▆▇</div></article>
  </div>;
}
