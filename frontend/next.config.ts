import type { NextConfig } from "next";

const BACKEND_URL = process.env.NEXT_PUBLIC_API_URL
  ? process.env.NEXT_PUBLIC_API_URL.replace("/api/v1", "")
  : "https://loyalty-nexus-api.onrender.com";

const nextConfig: NextConfig = {
  output: "standalone",
  experimental: {
    turbo: {},
  },
  // Proxy /s/{id} → backend so generated sites are served on the frontend domain
  async rewrites() {
    return [
      {
        source: "/s/:id",
        destination: `${BACKEND_URL}/s/:id`,
      },
    ];
  },
  async headers() {
    return [
      {
        // Don't apply DENY to the proxied /s/* pages — they need to be embeddable
        source: "/s/(.*)",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
        ],
      },
      {
        source: "/((?!s/).*)",
        headers: [
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
        ],
      },
    ];
  },
};

export default nextConfig;
