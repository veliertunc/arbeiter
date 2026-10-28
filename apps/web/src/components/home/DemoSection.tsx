import AuthPanel from "../../app/ui/AuthPanel";

export function DemoSection() {
  return <section className="demo" id="demo">
    <div><span>TRY IT YOURSELF</span><h2>Your identity workspace,<br/>ready when you are.</h2><p>Connect to your Arbeiter Auth API and sign in to see your account.</p></div>
    <AuthPanel />
  </section>;
}
