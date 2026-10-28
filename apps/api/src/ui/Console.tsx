import { useAuthConsole } from "../hooks/useAuthConsole";
import { Sidebar } from "./Sidebar";
import { StatsGrid } from "./StatsGrid";
import { AccountPanel } from "./AccountPanel";
import { OrganizationPanel } from "./OrganizationPanel";

export function Console() {
  const state = useAuthConsole();

  return <div className="shell">
    <Sidebar user={state.user} tab={state.tab} onTabChange={state.setTab} />
    <main className="main">
      <header className="topbar">
        <div className="crumb">Workspace <span>/</span> <b>{state.tab}</b></div>
        <div className="top-actions"><span className="status-dot"/> All systems operational <button className="icon-button" aria-label="Help">?</button></div>
      </header>
      <section className="content" id="overview">
        <div className="eyebrow">WEDNESDAY, OCTOBER 28</div>
        <div className="title-row"><div><h1>{state.tab}</h1><p className="lead">Your authentication workspace at a glance.</p></div><button className="button secondary" onClick={() => void state.refresh()}>↻ &nbsp;Refresh data</button></div>
        <div className="banner"><div className="banner-icon">✦</div><div><b>Welcome to your authentication workspace</b><p>Manage your profile, sessions, and organizations from one secure place.</p></div><a href="http://localhost:4322" target="_blank" rel="noreferrer">Read the quickstart ↗</a></div>
        <StatsGrid user={state.user} organizations={state.organizations} sessions={state.sessions} />
        <div className="columns">
          <AccountPanel user={state.user} busy={state.busy} notice={state.notice} email={state.email} password={state.password} setEmail={state.setEmail} setPassword={state.setPassword} signIn={state.signIn} signOut={state.signOut} />
          <OrganizationPanel organizations={state.organizations} signedIn={Boolean(state.user)} />
        </div>
        <div className="footnote"><span>🔒 &nbsp; Sessions are stored as secure, HTTP-only cookies.</span><span>Powered by Arbeiter Auth API</span></div>
      </section>
    </main>
  </div>;
}
