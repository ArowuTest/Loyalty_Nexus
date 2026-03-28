import React from "react";
import Link from "next/link";
import { Zap, Twitter, Instagram, Facebook, Youtube } from "lucide-react";

const PRODUCT_LINKS = [
  { label: "AI Studio",      href: "/studio" },
  { label: "Dashboard",      href: "/dashboard" },
  { label: "Spin & Win",     href: "/spin" },
  { label: "Pulse Points",   href: "/dashboard" },
  { label: "Regional Wars",  href: "/wars" },
];

const AI_TOOL_LINKS = [
  "Ask Nexus",
  "AI Photo Creator",
  "Video Generator",
  "Business Plan AI",
  "Voice to Plan",
  "Marketing Jingle",
];

const COMPANY_LINKS = [
  "About Us",
  "Blog",
  "Careers",
  "Privacy Policy",
  "Terms of Service",
  "Contact",
];

export default function Footer() {
  return (
    <footer className="border-t border-white/[0.07]" style={{ background: "#0a0b0e" }}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-14">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-10 mb-12">

          {/* Brand col */}
          <div className="col-span-2 md:col-span-1">
            <Link href="/" className="flex items-center gap-2 mb-5 group w-fit">
              <div className="w-8 h-8 rounded-lg bg-gold-500 flex items-center justify-center">
                <Zap className="w-4 h-4 text-black" />
              </div>
              <span className="font-black text-[15px]">
                <span className="text-gold">Loyalty</span>
                <span className="text-white"> Nexus</span>
              </span>
            </Link>
            <p className="text-[13px] text-white/40 leading-relaxed mb-5">
              Africa&apos;s most rewarding AI platform. Recharge MTN, earn Pulse Points,
              and unlock 30+ AI tools — all in one place.
            </p>
            <div className="flex items-center gap-2">
              {[Twitter, Instagram, Facebook, Youtube].map((Icon, i) => (
                <a
                  key={i}
                  href="#"
                  className="w-8 h-8 rounded-lg glass border border-white/[0.08] flex items-center justify-center text-white/40 hover:text-gold-500 hover:border-gold-500/30 transition-all"
                  aria-label="Social link"
                >
                  <Icon className="w-3.5 h-3.5" />
                </a>
              ))}
            </div>
          </div>

          {/* Product */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">
              Product
            </h4>
            <ul className="space-y-2.5">
              {PRODUCT_LINKS.map(({ label, href }) => (
                <li key={label}>
                  <Link
                    href={href}
                    className="text-[13px] text-white/40 hover:text-white transition-colors"
                  >
                    {label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* AI Tools */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">
              AI Tools
            </h4>
            <ul className="space-y-2.5">
              {AI_TOOL_LINKS.map(t => (
                <li key={t}>
                  <Link
                    href="/studio"
                    className="text-[13px] text-white/40 hover:text-white transition-colors"
                  >
                    {t}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Company */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">
              Company
            </h4>
            <ul className="space-y-2.5">
              {COMPANY_LINKS.map(t => (
                <li key={t}>
                  <span className="text-[13px] text-white/40 hover:text-white transition-colors cursor-pointer">
                    {t}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div className="border-t border-white/[0.06] pt-7 flex flex-col sm:flex-row items-center justify-between gap-3">
          <p className="text-[12px] text-white/25">
            © 2026 Loyalty Nexus · All rights reserved · Made with ❤️ in Nigeria 🇳🇬
          </p>
          <p className="text-[11px] text-white/20">
            Powered by Groq · Google Gemini · Pollinations · ElevenLabs · AssemblyAI
          </p>
        </div>
      </div>
    </footer>
  );
}
