import type { NextConfig } from "next";

const securityHeaders = [
  // Prevent the admin panel from being embedded in iframes (clickjacking)
  { key: "X-Frame-Options", value: "DENY" },
  // Prevent MIME-type sniffing
  { key: "X-Content-Type-Options", value: "nosniff" },
  // Referrer policy — don't leak admin URLs to third parties
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  // Enforce HTTPS for 2 years (HSTS)
  { key: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains; preload" },
  // Permissions policy — disable features the admin panel doesn't need
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=(), payment=()" },
  // Content-Security-Policy — tight for admin: only allow self + known CDNs
  {
    key: "Content-Security-Policy",
    value: [
      "default-src 'self'",
      "script-src 'self' 'unsafe-inline' 'unsafe-eval'", // Next.js requires unsafe-inline for hydration
      "style-src 'self' 'unsafe-inline'",                // Tailwind inline styles
      "img-src 'self' data: blob: https:",               // allow remote images in data tables
      "font-src 'self'",
      "connect-src 'self' https:",                       // allow API calls to backend
      "frame-ancestors 'none'",                          // belt-and-suspenders with X-Frame-Options
      "base-uri 'self'",
      "form-action 'self'",
    ].join("; "),
  },
];

const nextConfig: NextConfig = {
  output: "standalone",
  async headers() {
    return [
      {
        // Apply security headers to all admin routes
        source: "/(.*)",
        headers: securityHeaders,
      },
    ];
  },
};

export default nextConfig;
