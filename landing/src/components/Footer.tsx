import React from "react";
import { Link } from "react-router-dom";
import { Zap, Twitter, Instagram, Facebook, Youtube } from "lucide-react";
import { ROUTES } from "@/lib";

export default function Footer() {
  return (
    <footer className="border-t border-white/[0.07]" style={{ background: "oklch(0.08 0.008 240)" }}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-14">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-10 mb-12">

          {/* Brand col */}
          <div className="col-span-2 md:col-span-1">
            <Link to={ROUTES.HOME} className="flex items-center gap-2 mb-5 group w-fit">
              <div className="w-8 h-8 rounded-lg bg-gold flex items-center justify-center">
                <Zap className="w-4 h-4 text-black" />
              </div>
              <span className="font-black text-[15px]">
                <span className="text-gold">Loyalty</span>
                <span className="text-foreground"> Nexus</span>
              </span>
            </Link>
            <p className="text-[13px] text-muted-foreground leading-relaxed mb-5">
              Africa's most rewarding AI platform. Recharge MTN, earn Pulse Points,
              and unlock 30+ AI tools — all in one place.
            </p>
            <div className="flex items-center gap-2">
              {[Twitter, Instagram, Facebook, Youtube].map((Icon, i) => (
                <a
                  key={i}
                  href="#"
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
                { l: "AI Studio",    to: ROUTES.STUDIO },
                { l: "Dashboard",    to: ROUTES.DASHBOARD },
                { l: "Spin & Win",   to: ROUTES.DASHBOARD },
                { l: "Pulse Points", to: ROUTES.DASHBOARD },
                { l: "Refer & Earn", to: ROUTES.DASHBOARD },
              ].map(({ l, to }) => (
                <li key={l}>
                  <Link to={to} className="text-[13px] text-muted-foreground hover:text-foreground transition-colors">
                    {l}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Tools */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-muted-foreground/60 mb-4">AI Tools</h4>
            <ul className="space-y-2.5">
              {["Ask Nexus (Free)","AI Photo Creator","Video Generator","Business Plan AI","Voice to Plan","Marketing Jingle"].map(t => (
                <li key={t}>
                  <Link to={ROUTES.STUDIO} className="text-[13px] text-muted-foreground hover:text-foreground transition-colors">{t}</Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Company */}
          <div>
            <h4 className="text-[11px] font-black uppercase tracking-[0.18em] text-muted-foreground/60 mb-4">Company</h4>
            <ul className="space-y-2.5">
              {["About Us","Blog","Careers","Privacy Policy","Terms of Service","Contact"].map(t => (
                <li key={t}>
                  <span className="text-[13px] text-muted-foreground hover:text-foreground transition-colors cursor-pointer">{t}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div className="border-t border-white/[0.06] pt-7 flex flex-col sm:flex-row items-center justify-between gap-3">
          <p className="text-[12px] text-muted-foreground/40">
            © 2026 Loyalty Nexus · All rights reserved · Made with ❤️ in Nigeria 🇳🇬
          </p>
          <p className="text-[11px] text-muted-foreground/30">
            Powered by Groq · Google Gemini · Pollinations · ElevenLabs · AssemblyAI
          </p>
        </div>
      </div>
    </footer>
  );
}
