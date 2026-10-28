import type { Metadata } from "next";
import "./style.css";
export const metadata: Metadata = { title: "Arbeiter — Identity infrastructure", description: "Authentication and user management built for product teams." };
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) { return <html lang="en"><body>{children}</body></html>; }
