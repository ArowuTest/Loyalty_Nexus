import type { Metadata } from "next";
import "./globals.css";
export const metadata: Metadata = { title: "Loyalty Nexus — Admin Cockpit", description: "Operations dashboard" };
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return <html lang="en"><body>{children}</body></html>;
}