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
    slug:     "what-are-pulse-points",
    date:     "March 20, 2026",
    tag:      "How It Works",
    tagColor: "text-cyan-400 bg-cyan-400/10 border-cyan-400/20",
    title:    "What Are Pulse Points — And How Do You Earn Them?",
    excerpt:
      "Pulse Points are the currency of Loyalty Nexus. Every time you recharge your MTN line, you earn them automatically — no codes, no apps to open. Here's exactly how the system works.",
    readTime: "4 min read",
    featured: true,
    content: `
Pulse Points are the reward currency at the heart of Loyalty Nexus. Every time you recharge your MTN line with ₦250 or more, the system automatically converts a portion of that recharge into Pulse Points — credited to your wallet within seconds.

**The conversion rate is simple: ₦250 = 1 Pulse Point.**

So if you recharge ₦1,000, you earn 4 Pulse Points. Recharge ₦5,000 and you get 20 Pulse Points. The more you recharge, the more you accumulate.

## What Can You Do With Pulse Points?

Pulse Points unlock two things:

1. **AI Studio Tools** — spend your points to access 30+ AI tools including an AI chat assistant, photo generator, video creator, business plan generator, voice-to-plan, and more.

2. **Extra Wheel Spins** — every qualifying recharge (₦1,000+) earns you a free spin. You can also use Pulse Points to buy additional spins beyond your free daily allowance.

## Do Points Ever Expire?

Points are tied to your account and remain valid as long as your account is active. We don't set arbitrary expiry dates — your loyalty is what matters.

## Tier Multipliers

As you earn more points, you move up loyalty tiers — Bronze, Silver, Gold, Platinum, and Diamond. Higher tiers come with point multipliers, so Gold members earn 1.5× points on every recharge, and Diamond members earn 3× points.

Getting started is free. Just recharge your MTN line and your points will be waiting for you in your Loyalty Nexus wallet.
    `,
  },
  {
    slug:     "ai-studio-explained",
    date:     "March 14, 2026",
    tag:      "Product",
    tagColor: "text-yellow-400 bg-yellow-400/10 border-yellow-400/20",
    title:    "AI Studio Explained: 30+ Tools You Unlock With Your Recharge",
    excerpt:
      "From chatting with an AI assistant to generating images and building a full business plan by voice — AI Studio puts frontier AI in the hands of every MTN subscriber.",
    readTime: "6 min read",
    featured: false,
    content: `
AI Studio is the part of Loyalty Nexus where your Pulse Points become creative power. Once you've earned points from your MTN recharges, you can spend them on a growing library of AI tools — no foreign subscription, no credit card.

## The Tools Available Today

**Chat & Search**
- *Ask Nexus* — a conversational AI assistant you can ask anything. Explain a concept, draft a message, research a topic, or just have a conversation. This tool is free for all users.

**Create**
- *AI Photo Creator* — generate images from a text description. Describe what you want to see and the AI produces it.
- *Background Remover* — upload a photo and remove the background instantly.
- *Video Generator* — turn an image or prompt into a short cinematic video clip.

**Build**
- *Business Plan AI* — describe your business idea and receive a full structured plan including market analysis, financial projections, and go-to-market strategy.
- *Voice to Plan* — speak your idea into your phone and receive a written business plan in return.
- *Slide Deck* — generate a presentation from a topic or outline.

**Learn**
- *Study Guide* — turn any topic into a structured study guide.
- *Quiz Maker* — generate practice questions from any subject.
- *AI Podcast* — convert a topic into an audio podcast episode.

## How Much Do Tools Cost?

Pricing is set in Pulse Points and is always displayed before you use any tool. Ask Nexus (AI chat) is free. Most creation tools cost between 10 and 65 Pulse Points per generation.

Because you earn points from recharges you were already going to make, using AI Studio doesn't require any additional spending. It's a benefit you've already paid for.
    `,
  },
  {
    slug:     "spin-and-win-guide",
    date:     "March 7, 2026",
    tag:      "How It Works",
    tagColor: "text-cyan-400 bg-cyan-400/10 border-cyan-400/20",
    title:    "Spin & Win: How the Wheel Works and What You Can Win",
    excerpt:
      "The Loyalty Nexus prize wheel is one of the fastest ways to win real cash, data, and airtime. Here's a full breakdown of eligibility, prizes, and how payouts work.",
    readTime: "5 min read",
    featured: false,
    content: `
Every qualifying MTN recharge earns you a free spin on the Loyalty Nexus prize wheel. Here's everything you need to know about how it works.

## Who Is Eligible?

Any active Loyalty Nexus member who recharges their MTN line with ₦1,000 or more in a single transaction qualifies for a free spin. The spin is credited to your account automatically — no code to enter, no form to fill.

## What Can You Win?

The wheel contains a range of prizes:

- **Cash prizes** — up to ₦5,000 per spin, paid directly to your registered MoMo wallet
- **Data bundles** — 1GB, 2GB, or 5GB added to your MTN line
- **Airtime** — credited directly to your number
- **Bonus Pulse Points** — extra points added to your wallet immediately
- **Free Spin** — win an extra spin to use immediately

## How Are Prizes Delivered?

Cash prizes are sent via Mobile Money (MoMo) to the number you link in your account settings. Data and airtime are delivered directly to your MTN number. Pulse Points land in your wallet instantly.

## Can I Get More Than One Spin?

Yes. Higher loyalty tiers unlock multiple spins per qualifying recharge — Gold members get 3 spins, Platinum gets 5, and Diamond gets 10. You can also purchase additional spins using your Pulse Points from the dashboard.

## Is There a Daily Limit?

Each recharge earns the spins appropriate to your tier. There's no artificial cap beyond that — if you recharge multiple times in a day, each qualifying recharge earns its own spins.
    `,
  },
  {
    slug:     "regional-wars-overview",
    date:     "February 28, 2026",
    tag:      "Features",
    tagColor: "text-purple-400 bg-purple-400/10 border-purple-400/20",
    title:    "Regional Wars: How Your State Can Win ₦250,000",
    excerpt:
      "Regional Wars pits all 36 Nigerian states against each other every month. The state with the highest collective Pulse Points takes home ₦250,000. Here's how to contribute and win.",
    readTime: "5 min read",
    featured: false,
    content: `
Regional Wars is Loyalty Nexus's monthly competition where every active member contributes to their state's ranking — and the top three states share a ₦500,000 prize pool.

## How It Works

Every Pulse Point you earn from your MTN recharges is automatically added to your state's total. Your state is determined by the registration zone of your MTN number. There's nothing extra to do — just recharge as usual.

At the end of each month, the state totals are tallied and the top three states are announced.

## The Prize Pool

- 🥇 **1st Place** — ₦250,000 (50% of the pool)
- 🥈 **2nd Place** — ₦150,000 (30% of the pool)
- 🥉 **3rd Place** — ₦100,000 (20% of the pool)

## The Individual Draw

Beyond the state prize, there is also a personal cash draw within each winning state. One member from each of the top three states is randomly selected to receive a direct Mobile Money payout. You don't need to be the top contributor — just be active.

## How to Check Your State's Ranking

Log in to your dashboard and navigate to the Regional Wars tab. You'll see a live leaderboard showing all 36 states and their current standings.

The competition resets at the start of each calendar month.
    `,
  },
  {
    slug:     "loyalty-tiers-explained",
    date:     "February 20, 2026",
    tag:      "How It Works",
    tagColor: "text-cyan-400 bg-cyan-400/10 border-cyan-400/20",
    title:    "Bronze to Diamond: Understanding Your Loyalty Tier",
    excerpt:
      "There are five loyalty tiers on Loyalty Nexus — and your tier affects how fast you earn points, how many spins you get, and which AI tools you can access.",
    readTime: "4 min read",
    featured: false,
    content: `
Loyalty Nexus has five membership tiers: Bronze, Silver, Gold, Platinum, and Diamond. Your tier is determined by your total accumulated Pulse Points and comes with a set of benefits that grow as you level up.

## The Five Tiers

**🥉 Bronze (0 – 4,999 points)**
The starting tier for all new members. You earn 1× points on every recharge and receive 1 free spin per qualifying recharge.

**🥈 Silver (5,000 – 14,999 points)**
Earn 1.2× points per recharge. Get 2 free spins per qualifying recharge. Unlock all standard AI tools.

**🥇 Gold (15,000 – 39,999 points)**
Earn 1.5× points per recharge. Get 3 free spins per qualifying recharge. Priority AI processing queue.

**💎 Platinum (40,000 – 99,999 points)**
Earn 2× points per recharge. Get 5 free spins per qualifying recharge. Access to exclusive AI tools.

**💠 Diamond (100,000+ points)**
Earn 3× points per recharge. Get 10 free spins per qualifying recharge. Access to all premium tools including video generation at the highest quality tier.

## How Do I Move Up?

Simply recharge. Every naira you put on your MTN line earns you points, and those points accumulate in your wallet permanently. You can see your current tier and progress to the next level right on your dashboard.

Tiers are calculated on lifetime Pulse Points — they don't reset monthly. Once you've reached a tier, you keep it.
    `,
  },
  {
    slug:     "ai-for-nigerians",
    date:     "February 12, 2026",
    tag:      "Vision",
    tagColor: "text-green-400 bg-green-400/10 border-green-400/20",
    title:    "Why We're Making AI Accessible to Every Nigerian",
    excerpt:
      "Access to frontier AI tools in Nigeria has been limited by foreign payment requirements, high subscription costs, and unreliable connectivity. Loyalty Nexus is changing that.",
    readTime: "7 min read",
    featured: false,
    content: `
The best AI tools in the world — the ones capable of generating images, writing business plans, creating videos, and answering complex questions — have largely been out of reach for most Nigerians.

The barriers are real: Stripe-only payment forms that don't accept Nigerian cards, monthly subscription fees priced in dollars, and tools designed for high-bandwidth connections. A Nigerian small business owner who could genuinely benefit from AI has had almost no accessible path to it.

## Our Approach

Loyalty Nexus solves this by decoupling AI access from direct payment. Instead of asking users to pay a subscription, we let you earn access through something you're already doing — recharging your MTN line.

Every ₦250 you recharge earns 1 Pulse Point. Those points unlock AI tools directly. No dollar payment. No foreign card. No subscription.

## What This Means in Practice

A student in Kano can use AI to create a study guide for their JAMB preparation without paying a dollar subscription. A market trader in Onitsha can generate a simple business plan for a loan application. A designer in Lagos can remove backgrounds from product photos for their Instagram shop.

These are real, practical applications that make a difference — and they're all within reach through the phone credits you're already buying.

## The Bigger Picture

We believe that making AI accessible isn't just a product decision — it's an economic one. The businesses, students, and creatives that gain access to these tools will create more value, earn more, and contribute more to the broader economy.

Loyalty Nexus is our attempt to make that access as frictionless and rewarding as possible.
    `,
  },
];

export default function BlogPage() {
  const featured = POSTS[0];
  const rest     = POSTS.slice(1);

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
        <h1 className="text-4xl font-black mb-4">Insights, Updates &amp; Guides</h1>
        <p className="text-[15px] text-white/40">
          How Loyalty Nexus works, what you can unlock, and why we built it for Nigerians.
        </p>
      </section>

      {/* Featured post */}
      <section className="max-w-6xl mx-auto px-4 mb-10">
        <Link href={`/blog/${featured.slug}`}>
          <div className="rounded-2xl border border-yellow-400/20 bg-yellow-400/[0.03] p-8 hover:border-yellow-400/40 transition-colors cursor-pointer">
            <div className="flex flex-wrap items-center gap-3 mb-4">
              <span className={`text-[11px] font-bold px-2.5 py-1 rounded-full border ${featured.tagColor}`}>
                {featured.tag}
              </span>
              <span className="text-[11px] text-white/30">{featured.date}</span>
              <span className="text-[11px] text-white/30">·</span>
              <span className="text-[11px] text-white/30">{featured.readTime}</span>
            </div>
            <h2 className="text-2xl font-black mb-3">{featured.title}</h2>
            <p className="text-[15px] text-white/50 leading-relaxed mb-5">{featured.excerpt}</p>
            <span className="text-[13px] font-bold text-yellow-400 hover:underline">Read full post →</span>
          </div>
        </Link>
      </section>

      {/* Post grid */}
      <section className="max-w-6xl mx-auto px-4 pb-20">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {rest.map((p) => (
            <Link key={p.slug} href={`/blog/${p.slug}`}>
              <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-5 flex flex-col h-full hover:border-white/[0.15] transition-colors cursor-pointer">
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
                  <span className="text-[12px] font-bold text-yellow-400">Read →</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      </section>
    </main>
  );
}
