import React from "react";
import { Link } from "react-router-dom";
import { Zap, Twitter, Instagram, Facebook, Youtube, Mail, Phone } from "lucide-react";
import { ROUTES } from "@/lib";

// Helper to scroll to a section on the home page from the footer
function scrollToSection(sectionId: string) {
  if (window.location.pathname !== "/") {
    window.location.href = `/#${sectionId}`;
    return;
  }
  document.getElementById(sectionId)?.scrollIntoView({ behavior: "smooth", block: "start" });
}

export default function Footer() {
  return (
    <footer className="border-t border-white/[0.07]" style={{ background: "oklch(0.08 0.008 240)" }}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-14">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-10 mb-12">

          {/* Brand col */}
          <div className="col-span-2 md:col-span-2">
            <Link to={ROUTES.HOME} className="flex items-center gap-2 mb-5 group w-fit">
              <div className="w-8 h-8 rounded-lg bg-gold flex items-center justify-center">
                <Zap className="w-4 h-4 text-black" />
              </div>
              <span className="font-black text-[15px]">
                <span className="text-gold">Loyalty</span>
                <span className="text-foreground"> Nexus</span>
              </span>
            </Link>
            <p className="text-[13px] text-muted-foreground leading-relaxed mb-5 max-w-xs">
              Africa's most rewarding AI platform. Recharge MTN, earn Pulse Points,
              and unlock 30+ AI tools — all in one place. Built with ❤️ in Nigeria.
            </p>

            {/* Contact */}
            <div className="space-y-2 mb-5">
              <a
                href="mailto:hello@loyaltynexus.ng"
                className="flex items-center gap-2 text-[12px] text-muted-foreground hover:text-primary transition-colors"
              >
                <Mail className="w-3.5 h-3.5" />
                hello@loyaltynexus.ng
              </a>
              <a
                href="tel:+2348000000000"
                className="flex items-center gap-2 text-[12px] text-muted-foreground hover:text-primary transition-colors"
              >
                <Phone className="w-3.5 h-3.5" />
                +234 800 000 0000
              </a>
            </div>

            {/* Socials */}
            <div className="flex items-center gap-2">
              {[
                { Icon: Twitter,   href: "https://twitter.com/loyaltynexusng",  label: "X / Twitter" },
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
                  className="w-8 h-8 rounded-lg glass border border-white/[0.08] flex items-center justify-center text-muted-foreground hover:text-primary hover:border-primary/30 transition-all"
                >
                  <Icon className="w-3.5 h-3.5" />
                </a>
              ))}
            </div>
          </div>

          {/* Product */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-muted-foreground/60 mb-4">Product</h4>
            <ul className="space-y-2.5">
              {[
                { l: "AI Studio",    action: () => scrollToSection("ai-studio"),    isScroll: true },
                { l: "Regional Wars",action: () => scrollToSection("regional-wars"), isScroll: true },
                { l: "Spin & Win",   action: () => scrollToSection("spin-win"),      isScroll: true },
                { l: "Dashboard",    to: ROUTES.DASHBOARD,   isScroll: false },
                { l: "Refer & Earn", to: ROUTES.REFERRAL,    isScroll: false },
              ].map(({ l, action, to, isScroll }) => (
                <li key={l}>
                  {isScroll ? (
                    <button
                      onClick={action}
                      className="text-[13px] text-muted-foreground hover:text-foreground transition-colors text-left"
                    >
                      {l}
                    </button>
                  ) : (
                    <Link to={to!} className="text-[13px] text-muted-foreground hover:text-foreground transition-colors">
                      {l}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </div>

          {/* AI Tools */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-muted-foreground/60 mb-4">AI Tools</h4>
            <ul className="space-y-2.5">
              {[
                { l: "Ask Nexus (Free)",   cat: "chat"   },
                { l: "AI Photo Creator",   cat: "create" },
                { l: "Video Generator",    cat: "build"  },
                { l: "Business Plan AI",   cat: "build"  },
                { l: "Voice to Plan",      cat: "build"  },
                { l: "Marketing Jingle",   cat: "create" },
              ].map(({ l, cat }) => (
                <li key={l}>
                  <Link
                    to={`${ROUTES.STUDIO}?cat=${cat}`}
                    className="text-[13px] text-muted-foreground hover:text-foreground transition-colors"
                  >
                    {l}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Company */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-muted-foreground/60 mb-4">Company</h4>
            <ul className="space-y-2.5">
              {[
                { l: "About Us",         href: "/about"           },
                { l: "Blog",             href: "/blog"            },
                { l: "Careers",          href: "/careers"         },
                { l: "Privacy Policy",   href: "/privacy"         },
                { l: "Terms of Service", href: "/terms"           },
                { l: "Contact Us",       href: "mailto:hello@loyaltynexus.ng" },
              ].map(({ l, href }) => (
                <li key={l}>
                  {href.startsWith("mailto:") ? (
                    <a
                      href={href}
                      className="text-[13px] text-muted-foreground hover:text-foreground transition-colors"
                    >
                      {l}
                    </a>
                  ) : (
                    <Link
                      to={href}
                      className="text-[13px] text-muted-foreground hover:text-foreground transition-colors"
                    >
                      {l}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Bottom bar */}
        <div className="border-t border-white/[0.06] pt-7 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-[12px] text-muted-foreground/40">
            © 2026 Loyalty Nexus · All rights reserved · Made with ❤️ in Nigeria 🇳🇬
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4">
            <Link to="/privacy" className="text-[11px] text-muted-foreground/40 hover:text-muted-foreground transition-colors">Privacy Policy</Link>
            <span className="text-muted-foreground/20 text-[11px]">·</span>
            <Link to="/terms" className="text-[11px] text-muted-foreground/40 hover:text-muted-foreground transition-colors">Terms of Service</Link>
            <span className="text-muted-foreground/20 text-[11px]">·</span>
            <p className="text-[11px] text-muted-foreground/30">
              Powered by Groq · Google Gemini · ElevenLabs · AssemblyAI
            </p>
          </div>
        </div>
      </div>
    </footer>
  );
}
