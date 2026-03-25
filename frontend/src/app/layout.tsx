import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Loyalty Nexus — Recharge. Earn. Win.",
  description: "Nigeria's premium mobile rewards program. Recharge your airtime and win prizes with Loyalty Nexus.",
  manifest: "/manifest.json",
  themeColor: "#5f72f9",
  viewport: "width=device-width, initial-scale=1, maximum-scale=1",
  icons: {
    icon: "/icon-192.png",
    apple: "/icon-192.png",
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>{children}</body>
    </html>
  );
}
