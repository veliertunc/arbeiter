import { homeFeatures } from "../../content/homeFeatures";

export function FeatureSection() {
  return <section className="features" id="platform">
    <div className="section-title"><span>THE PLATFORM</span><h2>Everything you need.<br/><em>Nothing in your way.</em></h2><p>Identity should be the part of your product you never have to worry about.</p></div>
    <div className="feature-grid">{homeFeatures.map(feature => <article key={feature.title}><span className="feature-icon">{feature.icon}</span><h3>{feature.title}</h3><p>{feature.text}</p><a href="http://localhost:4322">Learn more ↗</a></article>)}</div>
  </section>;
}
