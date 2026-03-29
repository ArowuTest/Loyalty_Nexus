import React from "react";
import Link from "next/link";
import { Zap } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Careers | Loyalty Nexus",
  description: "Join Africa's most rewarding AI platform. Open roles at Loyalty Nexus.",
};

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
          Careers
        </div>
        <h1 className="text-4xl sm:text-5xl font-black mb-5">
          Help us build <span className="text-yellow-400">Africa&apos;s AI future</span>
        </h1>
        <p className="text-[16px] text-white/50 leading-relaxed">
          We&apos;re a small, fast-moving team building real products for real Nigerians.
          We&apos;d love to work with exceptional people who share our mission.
        </p>
      </section>

      {/* No open positions */}
      <section className="py-10 max-w-2xl mx-auto px-4">
        <div className="rounded-2xl border border-white/[0.08] bg-white/[0.02] p-12 text-center">
          <div className="text-5xl mb-6">🔍</div>
          <h2 className="text-2xl font-black mb-3">No open positions right now</h2>
          <p className="text-[15px] text-white/40 leading-relaxed mb-6">
            We don&apos;t have any open roles at the moment — but we&apos;re growing fast.
            <br className="hidden sm:block" />
            Check back soon or drop us a line to stay on our radar.
          </p>
          <a
            href="mailto:careers@loyaltynexus.ng?subject=Staying on your radar"
            className="inline-block px-6 py-3 rounded-lg border border-yellow-400/30 text-yellow-400 font-bold text-[13px] hover:bg-yellow-400/10 transition-colors"
          >
            Get in touch anyway →
          </a>
        </div>
      </section>

      {/* Back CTA */}
      <section className="py-10 text-center border-t border-white/[0.06] max-w-6xl mx-auto px-4 mt-6">
        <p className="text-[14px] text-white/30 mb-4">In the meantime, try the platform for yourself.</p>
        <Link
          href="/"
          className="inline-block px-6 py-3 rounded-lg bg-yellow-400 text-black font-black text-[13px] hover:bg-yellow-300 transition-colors"
        >
          Start Earning Free
        </Link>
      </section>
    </main>
  );
}
