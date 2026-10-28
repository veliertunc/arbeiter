import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

export default defineConfig({
  site: "https://docs.arbeiter.dev",
  integrations: [starlight({
    title: "Arbeiter Auth",
    description: "API and SDK documentation for Arbeiter Auth.",
    sidebar: [
      { label: "Start here", items: ["getting-started", "authentication", "sessions"] },
      { label: "API reference", items: ["api-reference", "organizations", "errors"] },
      { label: "SDKs", items: ["sdk/javascript", "sdk/python", "sdk/rust", "sdk/java", "sdk/cpp"] },
    ],
  })],
});
