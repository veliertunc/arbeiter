export function Hero() {
  return <section className="hero">
    <div className="hero-copy">
      <div className="badge"><i/> AUTH INFRASTRUCTURE, REIMAGINED</div>
      <h1>Identity that<br/>moves <em>with you.</em></h1>
      <p>Give your product a secure foundation for every sign-in, user, and team. Ship the experience you imagined.</p>
      <div className="hero-actions"><a className="button primary" href="#demo">Get started <span>→</span></a><a className="button ghost" href="http://localhost:4322">Explore the docs ↗</a></div>
      <div className="trusted"><span className="avatars"><i>A</i><i>M</i><i>J</i></span><span>Built for teams who care about identity</span></div>
    </div>
    <div className="hero-visual">
      <div className="glow"/><div className="visual-ring ring-a"/><div className="visual-ring ring-b"/>
      <div className="auth-card">
        <div className="auth-card-head"><span className="mini-mark">a</span><span className="secure">🔒 SECURE SESSION</span></div>
        <div className="person"><div className="person-avatar">A</div><div><b>Alex Morgan</b><small>alex@northstar.studio</small></div><span className="verified">✓</span></div>
        <div className="card-divider"/><div className="status-row"><span>Account status</span><b><i/> Verified</b></div><div className="status-row"><span>Active session</span><b>Just now</b></div>
        <div className="card-footer"><span>Connected to <b>Northstar Studio</b></span><span>↗</span></div>
      </div>
      <div className="floating float-one">✦ &nbsp; Email verified</div><div className="floating float-two"><i/> Session protected</div>
    </div>
    <div className="hero-bottom"><span>BUILT ON OPEN STANDARDS</span><div><b>REST API</b><b>OPEN SDKs</b><b>SECURE BY DEFAULT</b></div></div>
  </section>;
}
