"use client";
import React from "react";
import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { Zap, Twitter, Instagram, Facebook, Youtube, Mail, Phone } from "lucide-react";

// Map of AI tool display names to their slugs
const AI_TOOLS = [
  { label: "Ask Nexus (Free)",    slug: "ai-chat" },
  { label: "AI Photo Creator",    slug: "ai-photo" },
  { label: "Video Generator",     slug: "animate-photo" },
  { label: "Business Plan AI",    slug: "bizplan" },
  { label: "Voice to Plan",       slug: "voice-to-plan" },
  { label: "Marketing Jingle",    slug: "jingle" },
];

function useAuthStatus(): boolean | null {
  const [isAuth, setIsAuth] = React.useState<boolean | null>(null);
  React.useEffect(() => {
    setIsAuth(!!localStorage.getItem("nexus_token"));
  }, []);
  return isAuth;
}

function scrollToSection(sectionId: string, pathname: string) {
  if (pathname !== "/") {
    window.location.href = `/#${sectionId}`;
    return;
  }
  document.getElementById(sectionId)?.scrollIntoView({ behavior: "smooth", block: "start" });
}

export default function Footer() {
  const pathname = usePathname();
  const router   = useRouter();
  const isAuth   = useAuthStatus();

  /** Open the auth modal by dispatching a global event picked up by the landing page */
  function openAuthModal() {
    window.dispatchEvent(new CustomEvent("nexus:open-auth"));
  }

  /** Dashboard — open auth modal if not authed, /dashboard if authed */
  function handleDashboard(e: React.MouseEvent) {
    e.preventDefault();
    if (isAuth) router.push("/dashboard");
    else        openAuthModal();
  }

  /** AI Tool — open auth modal if not authed, or deep-link into the tool */
  function handleTool(e: React.MouseEvent, slug: string) {
    e.preventDefault();
    if (isAuth) router.push(`/studio?tool=${slug}`);
    else        openAuthModal();
  }

  return (
    <footer className="border-t border-white/[0.07]" style={{ background: "#0a0b0e" }}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-14">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-10 mb-12">

          {/* ── Brand ── */}
          <div className="col-span-2 md:col-span-2">
            <Link href="/" className="flex items-center gap-2 mb-5 group w-fit">
              <div className="w-8 h-8 rounded-lg bg-gold-500 flex items-center justify-center">
                <Zap className="w-4 h-4 text-black" />
              </div>
              <span className="font-black text-[15px]">
                <span className="text-gold">Loyalty</span>
                <span className="text-white"> Nexus</span>
              </span>
            </Link>
            <p className="text-[13px] text-white/40 leading-relaxed mb-5 max-w-xs">
              Africa&apos;s most rewarding AI platform. Recharge MTN, earn Pulse Points,
              and unlock 30+ AI tools — all in one place. Built with ❤️ in Nigeria.
            </p>

            {/* Contact */}
            <div className="space-y-2 mb-5">
              <a
                href="mailto:hello@loyaltynexus.ng"
                className="flex items-center gap-2 text-[12px] text-white/40 hover:text-gold-500 transition-colors"
              >
                <Mail className="w-3.5 h-3.5" />
                hello@loyaltynexus.ng
              </a>
              <a
                href="tel:+2348000000000"
                className="flex items-center gap-2 text-[12px] text-white/40 hover:text-gold-500 transition-colors"
              >
                <Phone className="w-3.5 h-3.5" />
                +234 800 000 0000
              </a>
            </div>

            {/* Socials */}
            <div className="flex items-center gap-2">
              {[
                { Icon: Twitter,   href: "https://twitter.com/loyaltynexusng",   label: "X / Twitter" },
                { Icon: Instagram, href: "https://instagram.com/loyaltynexusng", label: "Instagram" },
                { Icon: Facebook,  href: "https://facebook.com/loyaltynexusng",  label: "Facebook" },
                { Icon: Youtube,   href: "https://youtube.com/@loyaltynexusng",  label: "YouTube" },
              ].map(({ Icon, href, label }) => (
                <a
                  key={label}
                  href={href}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={label}
                  className="w-8 h-8 rounded-lg glass border border-white/[0.08] flex items-center justify-center text-white/40 hover:text-gold-500 hover:border-gold-500/30 transition-all"
                >
                  <Icon className="w-3.5 h-3.5" />
                </a>
              ))}
            </div>
          </div>

          {/* ── Product ── */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">Product</h4>
            <ul className="space-y-2.5">
              <li>
                <button
                  onClick={() => scrollToSection("ai-studio", pathname)}
                  className="text-[13px] text-white/40 hover:text-white transition-colors text-left"
                >
                  AI Studio
                </button>
              </li>
              <li>
                <button
                  onClick={() => scrollToSection("regional-wars", pathname)}
                  className="text-[13px] text-white/40 hover:text-white transition-colors text-left"
                >
                  Regional Wars
                </button>
              </li>
              <li>
                <button
                  onClick={() => scrollToSection("spin-win", pathname)}
                  className="text-[13px] text-white/40 hover:text-white transition-colors text-left"
                >
                  Spin &amp; Win
                </button>
              </li>
              <li>
                {/* Dashboard — auth-aware */}
                <a
                  href="#"
                  onClick={handleDashboard}
                  className="text-[13px] text-white/40 hover:text-white transition-colors"
                >
                  Dashboard
                </a>
              </li>
            </ul>
          </div>

          {/* ── AI Tools ── */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">AI Tools</h4>
            <ul className="space-y-2.5">
              {AI_TOOLS.map(({ label, slug }) => (
                <li key={slug}>
                  <a
                    href="#"
                    onClick={(e) => handleTool(e, slug)}
                    className="text-[13px] text-white/40 hover:text-white transition-colors"
                  >
                    {label}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          {/* ── Company ── */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">Company</h4>
            <ul className="space-y-2.5">
              {[
                { label: "About Us",         href: "/about" },
                { label: "Blog",             href: "/blog" },
                { label: "Careers",          href: "/careers" },
                { label: "Privacy Policy",   href: "/privacy" },
                { label: "Terms of Service", href: "/terms" },
                { label: "Contact Us",       href: "mailto:hello@loyaltynexus.ng" },
              ].map(({ label, href }) => (
                <li key={label}>
                  {href.startsWith("mailto:") ? (
                    <a href={href} className="text-[13px] text-white/40 hover:text-white transition-colors">
                      {label}
                    </a>
                  ) : (
                    <Link href={href} className="text-[13px] text-white/40 hover:text-white transition-colors">
                      {label}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* ── Bottom bar ── */}
        <div className="border-t border-white/[0.06] pt-7 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-[12px] text-white/25">
            © 2026 Loyalty Nexus · All rights reserved · Made with ❤️ in Nigeria 🇳🇬
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4">
            <Link href="/privacy" className="text-[11px] text-white/25 hover:text-white/50 transition-colors">Privacy Policy</Link>
            <span className="text-white/15 text-[11px]">·</span>
            <Link href="/terms"   className="text-[11px] text-white/25 hover:text-white/50 transition-colors">Terms of Service</Link>
            <span className="text-white/15 text-[11px]">·</span>
            <p className="text-[11px] text-white/20">Powered by Nexus AI Engine</p>
          </div>
        </div>
      </div>
    </footer>
  );
}
