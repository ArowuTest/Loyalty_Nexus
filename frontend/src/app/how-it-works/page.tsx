import React from "react";
import Link from "next/link";
import { Zap, ChevronRight } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "How It Works | Loyalty Nexus",
  description:
    "Learn how Loyalty Nexus works — recharge your MTN line, earn Pulse Points, spin to win cash prizes, and unlock 30+ AI tools. Four simple steps to everything.",
};

const STEPS = [
  {
    n: "01",
    icon: "📱",
    color: "#00D4FF",
    title: "Recharge MTN",
    body: "Recharge ₦1,000 or more on any MTN line. Your recharge is automatically detected by our system — no codes, no USSD, no hassle. Just recharge as you normally would.",
    stat: "₦250 = 1 Pulse Point",
    details: [
      "Works with any MTN prepaid line",
      "Minimum qualifying recharge: ₦1,000",
      "Detected automatically within seconds",
      "No app download required to start",
    ],
  },
  {
    n: "02",
    icon: "⚡",
    color: "#F5A623",
    title: "Earn Pulse Points",
    body: "Every naira you recharge earns Pulse Points. The more you recharge, the more you earn. Accumulate points to climb tiers — Bronze, Silver, Gold, and Platinum — each unlocking better rewards.",
    stat: "Points on every recharge",
    details: [
      "Bronze → Silver: 2,000 lifetime points",
      "Silver → Gold: 10,000 lifetime points",
      "Gold → Platinum: 50,000 lifetime points",
      "Higher tiers earn bonus multipliers",
    ],
  },
  {
    n: "03",
    icon: "🎰",
    color: "#10B981",
    title: "Spin & Win",
    body: "Each qualifying recharge earns you a free wheel spin. Land on instant cash, data bundles, airtime, or bonus Pulse Points. Prizes are credited to your account instantly — no waiting.",
    stat: "₦18M+ prizes distributed",
    details: [
      "One free spin per qualifying recharge",
      "Instant cash prizes paid to your wallet",
      "Data bundles credited to your MTN line",
      "Bonus spins for streak milestones",
    ],
  },
  {
    n: "04",
    icon: "🚀",
    color: "#8B5CF6",
    title: "Unlock AI Studio",
    body: "Spend your Pulse Points to access 30+ world-class AI tools — create stunning photos, generate videos, build business plans, compose music, and more. No foreign cards, no subscriptions.",
    stat: "1.2M+ generations created",
    details: [
      "30+ AI tools across 6 categories",
      "Pay per use with Pulse Points",
      "Photo generation, video, music & more",
      "Business plans, voice-to-text, and chat",
    ],
  },
];

const FAQS = [
  {
    q: "Which networks are supported?",
    a: "Currently, Loyalty Nexus supports MTN Nigeria lines only. We are working to expand to other networks in the future.",
  },
  {
    q: "How quickly are recharges detected?",
    a: "Recharges are typically detected within a few seconds of the transaction completing. In rare cases it may take up to a minute.",
  },
  {
    q: "How do I withdraw my cash prizes?",
    a: "Cash prizes are credited to your Loyalty Nexus wallet. You can withdraw to your bank account directly from the Prizes section of the app.",
  },
  {
    q: "Do Pulse Points expire?",
    a: "Pulse Points do not expire as long as your account remains active (at least one recharge every 90 days).",
  },
  {
    q: "What AI tools are available in AI Studio?",
    a: "AI Studio includes tools for image generation, AI photo editing, video creation, music composition, business plan writing, voice-to-text, and AI chat — over 30 tools in total.",
  },
  {
    q: "Is there a minimum recharge amount?",
    a: "Yes. The minimum qualifying recharge to earn Pulse Points and a free spin is ₦1,000. Smaller recharges still work on your MTN line but do not earn rewards.",
  },
];

export default function HowItWorksPage() {
  return (
    <main className="min-h-screen" style={{ background: "#0a0b0e", color: "#f0f2ff" }}>
      {/* ── Nav ── */}
      <div className="border-b border-white/[0.07]">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-yellow-400 flex items-center justify-center">
              <Zap className="w-3.5 h-3.5 text-black" />
            </div>
            <span className="font-black text-[14px]">
              <span className="text-yellow-400">Loyalty</span>
              <span className="text-white"> Nexus</span>
            </span>
          </Link>
          <Link href="/" className="text-[13px] text-white/40 hover:text-white transition-colors">
            ← Back to Home
          </Link>
        </div>
      </div>

      {/* ── Hero ── */}
      <section className="py-20 text-center max-w-3xl mx-auto px-4">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-[11px] font-semibold uppercase tracking-widest mb-6 border border-yellow-400/20 text-yellow-400 bg-yellow-400/5">
          How It Works
        </div>
        <h1 className="text-4xl sm:text-5xl font-black mb-6">
          Four steps to{" "}
          <span className="text-yellow-400">everything</span>
        </h1>
        <p className="text-[16px] text-white/50 leading-relaxed">
          From your first recharge to winning cash prizes and creating with AI — here&apos;s the complete
          journey. It all starts with something you already do every day.
        </p>
      </section>

      {/* ── Steps ── */}
      <section className="max-w-6xl mx-auto px-4 sm:px-6 pb-20">
        <div className="space-y-6">
          {STEPS.map((step, idx) => (
            <div
              key={step.n}
              className="rounded-2xl border border-white/[0.08] p-6 sm:p-8"
              style={{ background: "rgba(255,255,255,0.02)" }}
            >
              <div className="flex flex-col sm:flex-row sm:items-start gap-6">
                {/* Icon + number */}
                <div className="flex items-center gap-4 sm:flex-col sm:items-center sm:gap-2 flex-shrink-0">
                  <div
                    className="w-14 h-14 rounded-2xl flex items-center justify-center text-3xl"
                    style={{ background: `${step.color}15`, border: `1px solid ${step.color}30` }}
                  >
                    {step.icon}
                  </div>
                  <span className="text-[11px] font-black text-white/20 font-mono">{step.n}</span>
                </div>

                {/* Content */}
                <div className="flex-1">
                  <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 mb-3">
                    <h2 className="text-xl sm:text-2xl font-black text-white">{step.title}</h2>
                    <span
                      className="text-[12px] font-bold px-3 py-1 rounded-full self-start sm:self-auto"
                      style={{ color: step.color, background: `${step.color}15`, border: `1px solid ${step.color}30` }}
                    >
                      {step.stat}
                    </span>
                  </div>
                  <p className="text-[15px] text-white/50 leading-relaxed mb-4">{step.body}</p>
                  <ul className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {step.details.map((detail) => (
                      <li key={detail} className="flex items-center gap-2 text-[13px] text-white/40">
                        <ChevronRight className="w-3.5 h-3.5 flex-shrink-0" style={{ color: step.color }} />
                        {detail}
                      </li>
                    ))}
                  </ul>
                </div>
              </div>

              {/* Connector line (not on last item) */}
              {idx < STEPS.length - 1 && (
                <div className="mt-6 flex justify-center sm:justify-start sm:ml-7">
                  <div className="w-0.5 h-6 bg-white/10 rounded-full" />
                </div>
              )}
            </div>
          ))}
        </div>
      </section>

      {/* ── Tier ladder ── */}
      <section className="max-w-6xl mx-auto px-4 sm:px-6 pb-20">
        <div className="text-center mb-10">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-[11px] font-semibold uppercase tracking-widest mb-4 border border-purple-400/20 text-purple-400 bg-purple-400/5">
            Loyalty Tiers
          </div>
          <h2 className="text-3xl font-black text-white mb-3">The higher you climb, the more you earn</h2>
          <p className="text-[15px] text-white/40 max-w-xl mx-auto">
            Your lifetime Pulse Points determine your tier. Each tier unlocks better spin prizes, bonus
            multipliers, and exclusive AI Studio access.
          </p>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          {[
            { tier: "Bronze",   icon: "🥉", color: "#CD7F32", pts: "0+",      perks: ["1× spin multiplier", "Standard prizes", "Basic AI Studio"] },
            { tier: "Silver",   icon: "🥈", color: "#C0C0C0", pts: "2,000+",  perks: ["1.2× spin multiplier", "Silver prize pool", "More AI tools"] },
            { tier: "Gold",     icon: "🥇", color: "#F5A623", pts: "10,000+", perks: ["1.5× spin multiplier", "Gold prize pool", "Priority AI access"] },
            { tier: "Platinum", icon: "💎", color: "#A78BFA", pts: "50,000+", perks: ["2× spin multiplier", "Platinum prize pool", "All AI tools unlocked"] },
          ].map((t) => (
            <div
              key={t.tier}
              className="rounded-2xl border border-white/[0.08] p-5 flex flex-col"
              style={{ background: "rgba(255,255,255,0.02)" }}
            >
              <div className="text-3xl mb-2">{t.icon}</div>
              <h3 className="font-black text-white mb-0.5">{t.tier}</h3>
              <p className="text-[11px] font-bold mb-3" style={{ color: t.color }}>{t.pts} points</p>
              <ul className="space-y-1.5 mt-auto">
                {t.perks.map((perk) => (
                  <li key={perk} className="text-[12px] text-white/40 flex items-center gap-1.5">
                    <span className="w-1 h-1 rounded-full flex-shrink-0" style={{ background: t.color }} />
                    {perk}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </section>

      {/* ── FAQ ── */}
      <section className="max-w-3xl mx-auto px-4 sm:px-6 pb-20">
        <div className="text-center mb-10">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-[11px] font-semibold uppercase tracking-widest mb-4 border border-blue-400/20 text-blue-400 bg-blue-400/5">
            FAQ
          </div>
          <h2 className="text-3xl font-black text-white">Common questions</h2>
        </div>
        <div className="space-y-3">
          {FAQS.map((faq) => (
            <div
              key={faq.q}
              className="rounded-2xl border border-white/[0.08] p-5"
              style={{ background: "rgba(255,255,255,0.02)" }}
            >
              <p className="font-bold text-white mb-2">{faq.q}</p>
              <p className="text-[14px] text-white/50 leading-relaxed">{faq.a}</p>
            </div>
          ))}
        </div>
      </section>

      {/* ── CTA ── */}
      <section className="max-w-3xl mx-auto px-4 sm:px-6 pb-24 text-center">
        <div
          className="rounded-3xl border border-yellow-400/20 p-10"
          style={{ background: "rgba(245,166,35,0.04)" }}
        >
          <div className="text-4xl mb-4">⚡</div>
          <h2 className="text-3xl font-black text-white mb-3">Ready to start earning?</h2>
          <p className="text-[15px] text-white/50 mb-6">
            Create your account in seconds. No credit card, no subscription — just recharge and earn.
          </p>
          <Link
            href="/register"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-xl font-bold text-[14px] text-black transition-all hover:scale-105 active:scale-95"
            style={{ background: "#F5A623" }}
          >
            Get Started Free
            <ChevronRight className="w-4 h-4" />
          </Link>
        </div>
      </section>

      {/* ── Footer ── */}
      <footer className="border-t border-white/[0.07] py-8">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between gap-4">
          <Link href="/" className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-lg bg-yellow-400 flex items-center justify-center">
              <Zap className="w-3 h-3 text-black" />
            </div>
            <span className="font-black text-[13px]">
              <span className="text-yellow-400">Loyalty</span>
              <span className="text-white"> Nexus</span>
            </span>
          </Link>
          <div className="flex items-center gap-6 text-[13px] text-white/30">
            <Link href="/about" className="hover:text-white transition-colors">About</Link>
            <Link href="/privacy" className="hover:text-white transition-colors">Privacy</Link>
            <Link href="/terms" className="hover:text-white transition-colors">Terms</Link>
            <Link href="/blog" className="hover:text-white transition-colors">Blog</Link>
          </div>
        </div>
      </footer>
    </main>
  );
}
