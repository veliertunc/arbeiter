import type { ConsoleTab } from "../types";
import type { User } from "../types";

const navigation: Array<{ label: ConsoleTab | "API guide"; icon: string; target?: string }> = [
  { label: "Overview", icon: "▦", target: "overview" },
  { label: "Profile", icon: "◉", target: "profile" },
  { label: "Organizations", icon: "⌂", target: "organizations" },
  { label: "Sessions", icon: "◷", target: "sessions" },
  { label: "API guide", icon: "⌘" },
];

export function Sidebar({ user, tab, onTabChange }: {
  user: User | null;
  tab: ConsoleTab;
  onTabChange: (tab: ConsoleTab) => void;
}) {
  function navigate(item: typeof navigation[number]) {
    if (item.label === "API guide") {
      window.open("http://localhost:4322", "_blank", "noopener,noreferrer");
      return;
    }
    onTabChange(item.label);
    if (item.target) document.getElementById(item.target)?.scrollIntoView({ behavior: "smooth" });
  }

  return <aside className="sidebar">
    <a className="brand" href="#"><span className="brand-mark">a</span> arbeiter<span className="brand-small">AUTH</span></a>
    <div className="project"><span className="project-icon">A</span><span><b>Acme Studio</b><small>Development</small></span><span className="chevron">⌄</span></div>
    <div className="side-label">WORKSPACE</div>
    {navigation.map(item => <button key={item.label} className={`nav-item ${tab === item.label ? "active" : ""}`} onClick={() => navigate(item)}>
      <span>{item.icon}</span>{item.label}
    </button>)}
    <div className="sidebar-bottom">
      <div className="help-card"><span className="spark">✳</span><b>Ship auth faster</b><p>Explore the API reference and SDK guides.</p><a href="http://localhost:4322" target="_blank" rel="noreferrer">Open documentation ↗</a></div>
      <div className="profile"><div className="avatar">{user?.name?.[0]?.toUpperCase() ?? "D"}</div><span><b>{user?.name ?? "Developer"}</b><small>{user?.email ?? "Local workspace"}</small></span><button aria-label="Profile options">•••</button></div>
    </div>
  </aside>;
}
