import type { Organization } from "../types";

export function OrganizationPanel({ organizations, signedIn }: { organizations: Organization[]; signedIn: boolean }) {
  return <article className="panel" id="organization-panel">
    <div className="panel-heading"><div><h2>Organizations</h2><p>Your connected teams and workspaces.</p></div><a className="text-link" href="http://localhost:4322" target="_blank" rel="noreferrer">API guide ↗</a></div>
    {organizations.length ? <div className="org-list">{organizations.map((org, index) => <div className="org-row" key={org.id ?? index}><span className="org-logo">{org.name?.[0] ?? "O"}</span><span><b>{org.name}</b><small>/{org.slug}</small></span><span className="role">{org.role ?? "Member"}</span></div>)}</div> :
      <div className="empty"><span>⌂</span><b>{signedIn ? "No organizations yet" : "Sign in to view organizations"}</b><p>Your organizations will appear here.</p></div>}
  </article>;
}
