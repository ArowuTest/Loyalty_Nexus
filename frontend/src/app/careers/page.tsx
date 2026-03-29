import React from "react";
import Link from "next/link";
import { Zap, MapPin, Clock } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Careers | Loyalty Nexus",
  description: "Join Africa's most rewarding AI platform. Open roles at Loyalty Nexus.",
};

const ROLES = [
  {
    title:    "Senior Full-Stack Engineer (Next.js / Go)",
    team:     "Engineering",
    type:     "Full-Time",
    location: "Lagos · Hybrid",
    desc:     "Own major product features end-to-end — from Next.js frontend to Go microservices. You'll work directly on real-time recharge detection, Pulse Point ledger, and AI generation pipelines.",
    skills:   ["Next.js", "Go", "PostgreSQL", "Redis", "WebSockets"],
  },
  {
    title:    "AI/ML Engineer",
    team:     "AI Engineering",
    type:     "Full-Time",
    location: "Lagos · Remote OK",
    desc:     "Build and optimise prompt pipelines, fine-tuning workflows, and model integrations across Groq, Google Gemini, ElevenLabs, and Pollinations. You'll directly influence what 500K+ users experience.",
    skills:   ["Python", "LLM APIs", "Groq", "Gemini", "Prompt Engineering"],
  },
  {
    title:    "Product Designer (Mobile + Web)",
    team:     "Product",
    type:     "Full-Time",
    location: "Lagos · Hybrid",
    desc:     "Design delightful, fast, and accessible experiences for Nigerian users across web and Flutter mobile. Own the design system and drive usability from research to production.",
    skills:   ["Figma", "Motion Design", "User Research", "Design Systems", "Flutter"],
  },
  {
    title:    "Growth & Partnerships Manager",
    team:     "Business Development",
    type:     "Full-Time",
    location: "Lagos",
    desc:     "Drive MTN subscriber acquisition and engagement through co-marketing campaigns, USSD integration promotions, and brand partnerships. Own our subscriber growth KPIs.",
    skills:   ["Telecom BD", "Campaign Management", "Data Analytics", "MTN Ecosystem"],
  },
  {
    title:    "Backend Engineer (Go / Python)",
    team:     "Engineering",
    type:     "Full-Time",
    location: "Lagos · Remote OK",
    desc:     "Scale our Go backend handling millions of recharge events, spin transactions, and AI generations daily. Own reliability, observability, and performance.",
    skills:   ["Go", "Python", "PostgreSQL", "Docker", "Kubernetes"],
  },
  {
    title:    "Flutter Mobile Engineer",
    team:     "Engineering",
    type:     "Full-Time",
    location: "Lagos · Hybrid",
    desc:     "Build and maintain our Flutter mobile app used by hundreds of thousands of Nigerians daily. Collaborate closely with product and design to ship features fast.",
    skills:   ["Flutter", "Dart", "Riverpod", "REST APIs", "Firebase"],
  },
];

const PERKS = [
  { emoji: "💰", title: "Competitive Naira Salary",   body: "Market-rate compensation benchmarked against top Lagos tech companies, paid in NGN." },
  { emoji: "🧠", title: "Learning Budget",            body: "₦500,000/year for courses, conferences, books, and certifications." },
  { emoji: "🏥", title: "Health Insurance",           body: "Full HMO coverage for you and your immediate family through a leading Nigerian provider." },
  { emoji: "🏖️", title: "Generous Leave",            body: "25 days annual leave + public holidays + your birthday off." },
  { emoji: "💎", title: "Pulse Points Allowance",     body: "Monthly Pulse Points allocation to use every AI tool we build — on us." },
  { emoji: "🚀", title: "Ownership & Equity",         body: "ESOP available for senior roles. We want you to share in the upside you help create." },
];

export default function CareersPage() {
  return (
    <main className="min-h-screen" style={{ background: "#0a0b0e", color: "#f0f2ff" }}>
      {/* Nav */}
      <div className="border-b border-white/[0.07]">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-yellow-400 flex items-center justify-center">
              <Zap className="w-3.5 h-3.5 text-black" />
            </div>
            <span className="font-black text-[14px]">
              <span className="text-yellow-400">Loyalty</span><span className="text-white"> Nexus</span>
            </span>
          </Link>
          <Link href="/" className="text-[13px] text-white/40 hover:text-white transition-colors">← Back to Home</Link>
        </div>
      </div>

      {/* Hero */}
      <section className="py-20 text-center max-w-2xl mx-auto px-4">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-[11px] font-semibold uppercase tracking-widest mb-5 border border-yellow-400/20 text-yellow-400 bg-yellow-400/5">
          We&apos;re Hiring
        </div>
        <h1 className="text-4xl sm:text-5xl font-black mb-5">
          Help us build <span className="text-yellow-400">Africa&apos;s AI future</span>
        </h1>
        <p className="text-[16px] text-white/50 leading-relaxed">
          We&apos;re a small, fast-moving team shipping real products used by real Nigerians.
          If you want to build things that matter — and see the impact immediately — you&apos;ll fit right in.
        </p>
      </section>

      {/* Perks */}
      <section className="py-12 border-t border-white/[0.06]">
        <div className="max-w-6xl mx-auto px-4">
          <h2 className="text-2xl font-black text-center mb-8">Why Loyalty Nexus?</h2>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            {PERKS.map(({ emoji, title, body }) => (
              <div key={title} className="p-5 rounded-xl border border-white/[0.06] bg-white/[0.02]">
                <span className="text-2xl mb-3 block">{emoji}</span>
                <p className="text-[13px] font-bold mb-1.5">{title}</p>
                <p className="text-[12px] text-white/40 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Open roles */}
      <section className="py-16 border-t border-white/[0.06]">
        <div className="max-w-6xl mx-auto px-4">
          <h2 className="text-2xl font-black mb-8">Open Roles <span className="text-yellow-400">({ROLES.length})</span></h2>
          <div className="space-y-4">
            {ROLES.map((r) => (
              <div key={r.title} className="p-6 rounded-xl border border-white/[0.06] bg-white/[0.02] hover:border-yellow-400/20 transition-colors">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="flex-1">
                    <div className="flex flex-wrap items-center gap-2 mb-2">
                      <span className="text-[11px] font-bold px-2.5 py-1 rounded-full border text-yellow-400 bg-yellow-400/10 border-yellow-400/20">
                        {r.team}
                      </span>
                      <span className="flex items-center gap-1 text-[11px] text-white/30">
                        <Clock className="w-3 h-3" /> {r.type}
                      </span>
                      <span className="flex items-center gap-1 text-[11px] text-white/30">
                        <MapPin className="w-3 h-3" /> {r.location}
                      </span>
                    </div>
                    <h3 className="text-[16px] font-bold mb-2">{r.title}</h3>
                    <p className="text-[13px] text-white/40 leading-relaxed mb-3">{r.desc}</p>
                    <div className="flex flex-wrap gap-1.5">
                      {r.skills.map(s => (
                        <span key={s} className="text-[11px] px-2 py-0.5 rounded bg-white/[0.04] border border-white/[0.06] text-white/50">{s}</span>
                      ))}
                    </div>
                  </div>
                  <a
                    href={`mailto:careers@loyaltynexus.ng?subject=Application: ${encodeURIComponent(r.title)}`}
                    className="shrink-0 px-4 py-2 rounded-lg bg-yellow-400 text-black font-black text-[12px] hover:bg-yellow-300 transition-colors"
                  >
                    Apply Now
                  </a>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-8 p-6 rounded-xl border border-white/[0.06] bg-white/[0.02] text-center">
            <p className="text-[14px] text-white/50 mb-3">Don&apos;t see a role that fits? We love exceptional people.</p>
            <a
              href="mailto:careers@loyaltynexus.ng?subject=Open Application"
              className="inline-block px-5 py-2.5 rounded-lg border border-yellow-400/20 text-yellow-400 font-bold text-[13px] hover:bg-yellow-400/10 transition-colors"
            >
              Send a speculative application →
            </a>
          </div>
        </div>
      </section>
    </main>
  );
}
