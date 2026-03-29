"use client";
import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Zap, Twitter, Instagram, Facebook, Youtube, Mail, Phone } from "lucide-react";

function scrollToSection(sectionId: string, pathname: string) {
  if (pathname !== "/") {
    window.location.href = `/#${sectionId}`;
    return;
  }
  document.getElementById(sectionId)?.scrollIntoView({ behavior: "smooth", block: "start" });
}

export default function Footer() {
  const pathname = usePathname();

  return (
    <footer className="border-t border-white/[0.07]" style={{ background: "#0a0b0e" }}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-14">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-10 mb-12">

          {/* Brand col */}
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
                { Icon: Instagram, href: "https://instagram.com/loyaltynexusng",  label: "Instagram" },
                { Icon: Facebook,  href: "https://facebook.com/loyaltynexusng",   label: "Facebook" },
                { Icon: Youtube,   href: "https://youtube.com/@loyaltynexusng",   label: "YouTube" },
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

          {/* Product */}
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
                <Link href="/dashboard" className="text-[13px] text-white/40 hover:text-white transition-colors">
                  Dashboard
                </Link>
              </li>
              <li>
                <Link href="/dashboard" className="text-[13px] text-white/40 hover:text-white transition-colors">
                  Refer &amp; Earn
                </Link>
              </li>
            </ul>
          </div>

          {/* AI Tools */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">AI Tools</h4>
            <ul className="space-y-2.5">
              {[
                "Ask Nexus (Free)",
                "AI Photo Creator",
                "Video Generator",
                "Business Plan AI",
                "Voice to Plan",
                "Marketing Jingle",
              ].map(t => (
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
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-white/30 mb-4">Company</h4>
            <ul className="space-y-2.5">
              {[
                { label: "About Us",         href: "/about" },
                { label: "Blog",             href: "/blog" },
                { label: "Careers",          href: "/careers" },
                { label: "Privacy Policy",   href: "/privacy" },
                { label: "Terms of Service", href: "/terms" },
              ].map(({ label, href }) => (
                <li key={label}>
                  <Link
                    href={href}
                    className="text-[13px] text-white/40 hover:text-white transition-colors"
                  >
                    {label}
                  </Link>
                </li>
              ))}
              <li>
                <a
                  href="mailto:hello@loyaltynexus.ng"
                  className="text-[13px] text-white/40 hover:text-white transition-colors"
                >
                  Contact Us
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom bar */}
        <div className="border-t border-white/[0.06] pt-7 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-[12px] text-white/25">
            © 2026 Loyalty Nexus · All rights reserved · Made with ❤️ in Nigeria 🇳🇬
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4">
            <Link href="/privacy" className="text-[11px] text-white/25 hover:text-white/50 transition-colors">Privacy Policy</Link>
            <span className="text-white/15 text-[11px]">·</span>
            <Link href="/terms" className="text-[11px] text-white/25 hover:text-white/50 transition-colors">Terms of Service</Link>
            <span className="text-white/15 text-[11px]">·</span>
            <p className="text-[11px] text-white/20">
              Powered by Groq · Google Gemini · ElevenLabs · AssemblyAI
            </p>
          </div>
        </div>
      </div>
    </footer>
  );
}
