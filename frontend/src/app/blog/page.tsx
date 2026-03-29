import React from "react";
import Link from "next/link";
import { Zap } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Blog | Loyalty Nexus",
  description: "AI insights, product updates, and loyalty programme stories from the Loyalty Nexus team.",
};

const POSTS = [
  {
    date:    "March 20, 2026",
    tag:     "Product Update",
    tagColor:"text-cyan-400 bg-cyan-400/10 border-cyan-400/20",
    title:   "Introducing AI Studio Max: Generate Cinematic Videos with a Single Prompt",
    excerpt: "Our most powerful AI generation tier yet — full HD video from text, powered by Google Veo 2 and exclusively available to Diamond tier members.",
    readTime:"4 min read",
  },
  {
    date:    "March 12, 2026",
    tag:     "Loyalty Insights",
    tagColor:"text-yellow-400 bg-yellow-400/10 border-yellow-400/20",
    title:   "How ₦250 Changed Everything: The Economics of Pulse Points",
    excerpt: "We explain why we priced Pulse Points at ₦250 per point, how it compares to global loyalty standards, and why it gives the most value to everyday Nigerian subscribers.",
    readTime:"6 min read",
  },
  {
    date:    "March 5, 2026",
    tag:     "Community",
    tagColor:"text-green-400 bg-green-400/10 border-green-400/20",
    title:   "Regional Wars Season 1 Results: Lagos Takes the Crown",
    excerpt: "Over 200,000 subscribers competed in the first-ever Regional Wars. Lagos, Abuja, Port Harcourt, and Kano battled for ₦2M in prizes. Here's how it unfolded.",
    readTime:"5 min read",
  },
  {
    date:    "February 25, 2026",
    tag:     "AI Deep Dive",
    tagColor:"text-purple-400 bg-purple-400/10 border-purple-400/20",
    title:   "Why We Chose Groq Over OpenAI for Real-Time AI in Nigeria",
    excerpt: "Latency, cost, and reliability matter more in Nigeria than anywhere else. We tested 6 AI inference providers for 3 months. Here's exactly why we chose Groq.",
    readTime:"8 min read",
  },
  {
    date:    "February 14, 2026",
    tag:     "Partnership",
    tagColor:"text-orange-400 bg-orange-400/10 border-orange-400/20",
    title:   "Loyalty Nexus × MTN Nigeria: One Year of Building Together",
    excerpt: "We reflect on 12 months of joint engineering, go-to-market collaboration, and the ARPU data that convinced MTN to deepen our partnership.",
    readTime:"7 min read",
  },
  {
    date:    "January 30, 2026",
    tag:     "How-To",
    tagColor:"text-pink-400 bg-pink-400/10 border-pink-400/20",
    title:   "10 AI Studio Tools Every Nigerian Small Business Should Be Using in 2026",
    excerpt: "From business plan generation to social media jingles, these 10 Nexus AI tools can replace ₦200,000/month in software subscriptions for any Lagos SME.",
    readTime:"10 min read",
  },
];

export default function BlogPage() {
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

      {/* Header */}
      <section className="py-16 text-center max-w-2xl mx-auto px-4">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-[11px] font-semibold uppercase tracking-widest mb-5 border border-yellow-400/20 text-yellow-400 bg-yellow-400/5">
          Nexus Blog
        </div>
        <h1 className="text-4xl font-black mb-4">Insights, Updates &amp; Stories</h1>
        <p className="text-[15px] text-white/40">
          From AI deep dives to loyalty programme economics — the thinking behind Africa&apos;s most rewarding platform.
        </p>
      </section>

      {/* Featured post */}
      <section className="max-w-6xl mx-auto px-4 mb-10">
        <div className="rounded-2xl border border-yellow-400/20 bg-yellow-400/[0.03] p-8">
          <div className="flex flex-wrap items-center gap-3 mb-4">
            <span className={`text-[11px] font-bold px-2.5 py-1 rounded-full border ${POSTS[0].tagColor}`}>
              {POSTS[0].tag}
            </span>
            <span className="text-[11px] text-white/30">{POSTS[0].date}</span>
            <span className="text-[11px] text-white/30">·</span>
            <span className="text-[11px] text-white/30">{POSTS[0].readTime}</span>
          </div>
          <h2 className="text-2xl font-black mb-3">{POSTS[0].title}</h2>
          <p className="text-[15px] text-white/50 leading-relaxed mb-5">{POSTS[0].excerpt}</p>
          <button className="text-[13px] font-bold text-yellow-400 hover:underline">Read full post →</button>
        </div>
      </section>

      {/* Post grid */}
      <section className="max-w-6xl mx-auto px-4 pb-20">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {POSTS.slice(1).map((p) => (
            <div key={p.title} className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-5 flex flex-col">
              <div className="flex flex-wrap items-center gap-2 mb-3">
                <span className={`text-[11px] font-bold px-2.5 py-1 rounded-full border ${p.tagColor}`}>
                  {p.tag}
                </span>
                <span className="text-[11px] text-white/30">{p.readTime}</span>
              </div>
              <h3 className="text-[15px] font-bold mb-2 leading-snug">{p.title}</h3>
              <p className="text-[12px] text-white/40 leading-relaxed flex-1 mb-4">{p.excerpt}</p>
              <div className="flex items-center justify-between mt-auto">
                <span className="text-[11px] text-white/25">{p.date}</span>
                <button className="text-[12px] font-bold text-yellow-400 hover:underline">Read →</button>
              </div>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}
