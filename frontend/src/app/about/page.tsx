import React from "react";
import Link from "next/link";
import { Zap, Users, Globe, Cpu, Target, TrendingUp, Phone } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "About Us | Loyalty Nexus",
  description: "Learn about Loyalty Nexus — Africa's most rewarding AI platform, built for Nigerians.",
};

const TEAM = [
  { name: "Adewale Okonkwo",  role: "CEO & Co-Founder",          bio: "Former MTN Nigeria product lead with 12 years building telco loyalty systems across West Africa." },
  { name: "Chidinma Eze",     role: "CTO & Co-Founder",          bio: "Ex-Google engineer and AI researcher. Passionate about making frontier AI accessible and affordable for every Nigerian." },
  { name: "Tunde Fashola",    role: "Head of Partnerships",       bio: "Telecom ecosystem veteran who brokered MTN's first fintech integration deals in 2019." },
  { name: "Amaka Nwosu",      role: "Head of Product",           bio: "UX strategist with deep roots in Lagos tech community. Designed experiences used by 5M+ Nigerians." },
  { name: "Emeka Dike",       role: "Head of AI Engineering",    bio: "LLM specialist building Groq & Gemini integrations that run fast enough for real-time Nigerian workflows." },
  { name: "Fatima Bello",     role: "Head of Operations",        bio: "Operations lead ensuring seamless recharge detection and prize fulfilment across all 36 states." },
];

const VALUES = [
  { icon: "🇳🇬", title: "Nigeria-First",       body: "Every product decision is made for the Nigerian user first — the data costs, the languages, the naira, the culture." },
  { icon: "🤖", title: "AI for Everyone",     body: "Frontier AI shouldn't require a UK or US credit card. We make it accessible through something you already do — recharge your phone." },
  { icon: "🔒", title: "Trust & Transparency", body: "Your recharge data is yours. No selling, no sharing. Prize payouts are instant and fully auditable." },
  { icon: "🚀", title: "Speed Over Perfection", body: "We ship fast, iterate in public, and build the plane while flying it. Feedback from our users drives every sprint." },
  { icon: "🤝", title: "Partner Success",      body: "We succeed when MTN succeeds. Our ARPU and churn metrics are open-book with our MNO partners." },
];

export default function AboutPage() {
  return (
    <main className="min-h-screen" style={{ background: "#0a0b0e", color: "#f0f2ff" }}>
      {/* Nav back */}
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
      <section className="py-20 text-center max-w-3xl mx-auto px-4">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-[11px] font-semibold uppercase tracking-widest mb-6 border border-yellow-400/20 text-yellow-400 bg-yellow-400/5">
          Our Story
        </div>
        <h1 className="text-4xl sm:text-5xl font-black mb-6">
          Turning everyday recharges into<br />
          <span className="text-yellow-400">extraordinary opportunities</span>
        </h1>
        <p className="text-[16px] text-white/50 leading-relaxed">
          Loyalty Nexus was born from a simple frustration: Nigerians spend billions of naira on airtime and data every month,
          yet receive almost nothing in return. We set out to change that by combining loyalty rewards, AI superpowers,
          and real cash prizes — all unlocked by the recharge you were going to make anyway.
        </p>
      </section>

      {/* Stats */}
      <section className="py-12 border-y border-white/[0.06]">
        <div className="max-w-5xl mx-auto px-4 grid grid-cols-2 md:grid-cols-4 gap-8 text-center">
          {[
            { v: "500K+",  l: "Active Users" },
            { v: "₦18M+",  l: "Prizes Distributed" },
            { v: "30+",    l: "AI Tools" },
            { v: "1.2M+",  l: "AI Generations" },
          ].map(({ v, l }) => (
            <div key={l}>
              <p className="text-3xl font-black text-yellow-400 mb-1">{v}</p>
              <p className="text-[13px] text-white/40">{l}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Mission */}
      <section className="py-16 max-w-6xl mx-auto px-4 grid md:grid-cols-2 gap-10 items-center">
        <div>
          <h2 className="text-3xl font-black mb-4">Our Mission</h2>
          <p className="text-[15px] text-white/50 leading-relaxed mb-4">
            To democratise access to artificial intelligence across Africa by building the
            continent&apos;s most rewarding loyalty platform — one that makes every naira you spend
            work harder for you.
          </p>
          <p className="text-[15px] text-white/50 leading-relaxed">
            We partner directly with MTN Nigeria to detect recharges in real time, convert them to
            Pulse Points, and unlock access to frontier AI tools that previously required expensive
            foreign subscriptions. No more barriers. No more excuses.
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4">
          {[
            { Icon: Users,     label: "B2B2C Model",       desc: "We work through MTN to reach every subscriber" },
            { Icon: Globe,     label: "Pan-African Vision", desc: "Nigeria first, then Ghana, Kenya, and beyond" },
            { Icon: Cpu,       label: "AI-Powered Core",   desc: "Groq, Gemini, ElevenLabs in every tool" },
            { Icon: TrendingUp, label: "Real ROI",         desc: "Measurable ARPU lift and churn reduction for MNOs" },
          ].map(({ Icon, label, desc }) => (
            <div key={label} className="p-4 rounded-xl border border-white/[0.06] bg-white/[0.02]">
              <Icon className="w-5 h-5 text-yellow-400 mb-2" />
              <p className="text-[13px] font-semibold mb-1">{label}</p>
              <p className="text-[11px] text-white/40">{desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Values */}
      <section className="py-16 border-t border-white/[0.06]">
        <div className="max-w-6xl mx-auto px-4">
          <h2 className="text-3xl font-black text-center mb-10">Our Values</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {VALUES.map(({ icon, title, body }) => (
              <div key={title} className="p-5 rounded-xl border border-white/[0.06] bg-white/[0.02]">
                <span className="text-2xl mb-3 block">{icon}</span>
                <p className="text-[14px] font-bold mb-2">{title}</p>
                <p className="text-[13px] text-white/40 leading-relaxed">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Team */}
      <section className="py-16 border-t border-white/[0.06]">
        <div className="max-w-6xl mx-auto px-4">
          <h2 className="text-3xl font-black text-center mb-10">The Team</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {TEAM.map(({ name, role, bio }) => (
              <div key={name} className="p-5 rounded-xl border border-white/[0.06] bg-white/[0.02]">
                <div className="w-10 h-10 rounded-full bg-yellow-400/10 border border-yellow-400/20 flex items-center justify-center text-lg font-black text-yellow-400 mb-3">
                  {name[0]}
                </div>
                <p className="text-[14px] font-bold">{name}</p>
                <p className="text-[12px] text-yellow-400/80 mb-2">{role}</p>
                <p className="text-[12px] text-white/40 leading-relaxed">{bio}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-16 text-center border-t border-white/[0.06]">
        <h2 className="text-2xl font-black mb-4">Want to join our journey?</h2>
        <p className="text-[14px] text-white/40 mb-6">We&apos;re always looking for exceptional people who love building for Africa.</p>
        <div className="flex justify-center gap-3">
          <Link href="/careers" className="px-5 py-2.5 rounded-lg bg-yellow-400 text-black font-black text-[13px] hover:bg-yellow-300 transition-colors">View Open Roles</Link>
          <a href="mailto:hello@loyaltynexus.ng" className="px-5 py-2.5 rounded-lg border border-white/[0.1] text-white/60 font-semibold text-[13px] hover:border-white/20 hover:text-white transition-colors flex items-center gap-2"><Phone className="w-3.5 h-3.5" />Contact Us</a>
        </div>
      </section>
    </main>
  );
}
