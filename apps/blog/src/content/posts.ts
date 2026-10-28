export type Article = {
  category: string;
  date: string;
  title: string;
  summary: string;
  slug: string;
  intro: string;
  sections: Array<{ heading: string; paragraphs: string[] }>;
};

export const posts: Article[] = [
  {
    category: "ENGINEERING",
    date: "Oct 21, 2026",
    title: "A better way to think about session security",
    summary: "Why opaque server-side sessions remain a strong default for products that need control, revocation, and clear boundaries.",
    slug: "session-security",
    intro: "A good session gives the user a smooth return and the product a reliable way to end access. Those goals should be designed together.",
    sections: [
      { heading: "Start with the boundary", paragraphs: ["Trust is built in the details. Authentication systems should keep the user experience simple while protecting credentials, revoking sessions, and requiring clear intent for sensitive actions.", "Keep identity operations behind one well-defined API. Give the client a small, understandable surface and let the server own sensitive decisions."] },
      { heading: "Make the secure choice the easy choice", paragraphs: ["Good defaults give teams a solid base: secure cookies, short-lived action tokens, thoughtful rate limits, and audit events that are useful without collecting secrets."] },
    ],
  },
  {
    category: "PRODUCT",
    date: "Oct 08, 2026",
    title: "Identity should feel invisible",
    summary: "Authentication is infrastructure. The best experience gives people confidence without getting in their way.",
    slug: "invisible-identity",
    intro: "Authentication is the first promise your product makes: your account is yours, and we will protect it.",
    sections: [
      { heading: "Put people at ease", paragraphs: ["Clear status, predictable recovery, and thoughtful security prompts turn a background system into a product people can trust.", "Consistency matters across every surface. Keep language calm and help users understand what happened and what to do next."] },
      { heading: "Keep the product moving", paragraphs: ["Good identity flows protect the account while letting people return to the work they came to do."] },
    ],
  },
  {
    category: "ENGINEERING",
    date: "Sep 24, 2026",
    title: "Multi-tenant authorization starts with ownership",
    summary: "A practical model for organizations, membership roles, and the rules that protect account boundaries.",
    slug: "organization-authorization",
    intro: "A tenant boundary is only as strong as the rules attached to it. Model ownership and membership explicitly.",
    sections: [
      { heading: "Start with ownership", paragraphs: ["Every organization should have a clear owner and a membership record for each user. Roles should explain what a user can do, not be trusted merely because a client supplied them.", "Check membership on every operation that reads or changes tenant data. A visible organization ID is context, never proof of authorization."] },
      { heading: "Make changes auditable", paragraphs: ["Record important ownership and role changes so teams can understand how access changed over time."] },
    ],
  },
];
