# Loyalty Nexus — Landing Page

Cinematic best-in-class marketing landing page built with **React 18 + Vite + TailwindCSS v4 + Framer Motion**.

## Features
- Animated aurora canvas hero
- Live activity ticker
- AI Studio tool showcase with category filters
- Interactive Spin & Win demo wheel
- Loyalty tier progression display
- Nigerian testimonials
- Announcement banner with login CTA
- Full mobile-first (Android + iOS optimised)

## Tech Stack
- React 18 + TypeScript
- Vite 5
- TailwindCSS 4
- Framer Motion
- shadcn/ui components
- React Router v6

## Dev
```bash
npm install
npm run dev        # http://localhost:8080
npm run build      # production build → dist/
```

## Structure
```
src/
├── pages/
│   ├── Home.tsx       ← Main landing page (cinematic hero, all sections)
│   ├── Studio.tsx     ← AI Studio showcase page
│   ├── Dashboard.tsx  ← User dashboard preview
│   └── Admin.tsx      ← Admin panel preview
├── components/
│   ├── NavBar.tsx
│   ├── Footer.tsx
│   ├── AuthModal.tsx  ← OTP login/register flow
│   └── ui/            ← shadcn/ui primitives
├── data/index.ts      ← Mock data (swap for real API calls)
└── lib/index.ts       ← Constants, types, utilities
```

## Deployment
This app is a static SPA. Build with `npm run build` and serve the `dist/` folder
from any CDN or static host (Cloudflare Pages, Render static site, Vercel, etc.).
