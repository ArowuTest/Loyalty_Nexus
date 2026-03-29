import React from "react";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Zap } from "lucide-react";
import type { Metadata } from "next";

// ─── All posts defined here (single source of truth) ───────────────────────
const POSTS = [
  {
    slug:     "what-are-pulse-points",
    date:     "March 20, 2026",
    tag:      "How It Works",
    tagColor: "text-cyan-400 bg-cyan-400/10 border-cyan-400/20",
    title:    "What Are Pulse Points — And How Do You Earn Them?",
    excerpt:  "Pulse Points are the currency of Loyalty Nexus. Every time you recharge your MTN line, you earn them automatically — no codes, no apps to open.",
    readTime: "4 min read",
    content: [
      {
        type: "p",
        text: "Pulse Points are the reward currency at the heart of Loyalty Nexus. Every time you recharge your MTN line with ₦250 or more, the system automatically converts a portion of that recharge into Pulse Points — credited to your wallet within seconds.",
      },
      {
        type: "callout",
        text: "The conversion rate is simple: ₦250 = 1 Pulse Point. Recharge ₦1,000 and earn 4 points. Recharge ₦5,000 and earn 20 points.",
      },
      { type: "h2", text: "What Can You Do With Pulse Points?" },
      {
        type: "p",
        text: "Pulse Points unlock two things on the platform:",
      },
      {
        type: "list",
        items: [
          "AI Studio Tools — spend your points to access 30+ AI tools: an AI chat assistant, photo generator, video creator, business plan generator, voice-to-plan, and more.",
          "Extra Wheel Spins — every qualifying recharge (₦1,000+) earns you a free spin. You can also use Pulse Points to buy additional spins beyond your free daily allowance.",
        ],
      },
      { type: "h2", text: "Do Points Ever Expire?" },
      {
        type: "p",
        text: "Points are tied to your account and remain valid as long as your account is active. There are no arbitrary expiry dates — your loyalty is what matters.",
      },
      { type: "h2", text: "Tier Multipliers" },
      {
        type: "p",
        text: "As you earn more points, you move up loyalty tiers — Bronze, Silver, Gold, Platinum, and Diamond. Higher tiers come with point multipliers. Gold members earn 1.5× points on every recharge. Diamond members earn 3× points. Getting started is free — just recharge your MTN line and your points will be waiting in your wallet.",
      },
    ],
  },
  {
    slug:     "ai-studio-explained",
    date:     "March 14, 2026",
    tag:      "Product",
    tagColor: "text-yellow-400 bg-yellow-400/10 border-yellow-400/20",
    title:    "AI Studio Explained: 30+ Tools You Unlock With Your Recharge",
    excerpt:  "From chatting with an AI assistant to generating images and building a full business plan by voice — AI Studio puts frontier AI in the hands of every MTN subscriber.",
    readTime: "6 min read",
    content: [
      {
        type: "p",
        text: "AI Studio is the part of Loyalty Nexus where your Pulse Points become creative power. Once you've earned points from your MTN recharges, you can spend them on a growing library of AI tools — no foreign subscription, no credit card required.",
      },
      { type: "h2", text: "Chat & Search" },
      {
        type: "list",
        items: [
          "Ask Nexus — a conversational AI assistant you can ask anything. Explain a concept, draft a message, research a topic, or just have a conversation. This tool is free for all users.",
        ],
      },
      { type: "h2", text: "Create" },
      {
        type: "list",
        items: [
          "AI Photo Creator — generate images from a text description. Describe what you want to see and the AI produces it.",
          "Background Remover — upload a photo and remove the background instantly.",
          "Video Generator — turn an image or prompt into a short cinematic video clip.",
        ],
      },
      { type: "h2", text: "Build" },
      {
        type: "list",
        items: [
          "Business Plan AI — describe your business idea and receive a full structured plan including market analysis, financial projections, and go-to-market strategy.",
          "Voice to Plan — speak your idea into your phone and receive a written business plan in return.",
          "Slide Deck — generate a presentation from a topic or outline.",
        ],
      },
      { type: "h2", text: "Learn" },
      {
        type: "list",
        items: [
          "Study Guide — turn any topic into a structured study guide.",
          "Quiz Maker — generate practice questions from any subject.",
          "AI Podcast — convert a topic into an audio podcast episode.",
        ],
      },
      { type: "h2", text: "How Much Do Tools Cost?" },
      {
        type: "p",
        text: "Pricing is set in Pulse Points and is always displayed before you use any tool. Ask Nexus (AI chat) is free. Most creation tools cost between 10 and 65 Pulse Points per generation. Because you earn points from recharges you were already going to make, using AI Studio doesn't require any additional spending. It's a benefit you've already paid for.",
      },
    ],
  },
  {
    slug:     "spin-and-win-guide",
    date:     "March 7, 2026",
    tag:      "How It Works",
    tagColor: "text-cyan-400 bg-cyan-400/10 border-cyan-400/20",
    title:    "Spin & Win: How the Wheel Works and What You Can Win",
    excerpt:  "The Loyalty Nexus prize wheel is one of the fastest ways to win real cash, data, and airtime. Here's a full breakdown of eligibility, prizes, and how payouts work.",
    readTime: "5 min read",
    content: [
      {
        type: "p",
        text: "Every qualifying MTN recharge earns you a free spin on the Loyalty Nexus prize wheel. Here's everything you need to know.",
      },
      { type: "h2", text: "Who Is Eligible?" },
      {
        type: "p",
        text: "Any active Loyalty Nexus member who recharges their MTN line with ₦1,000 or more in a single transaction qualifies for a free spin. The spin is credited to your account automatically — no code to enter, no form to fill.",
      },
      { type: "h2", text: "What Can You Win?" },
      {
        type: "list",
        items: [
          "Cash prizes — up to ₦5,000 per spin, paid directly to your registered MoMo wallet",
          "Data bundles — 1GB, 2GB, or 5GB added to your MTN line",
          "Airtime — credited directly to your number",
          "Bonus Pulse Points — extra points added to your wallet immediately",
          "Free Spin — win an extra spin to use right away",
        ],
      },
      { type: "h2", text: "How Are Prizes Delivered?" },
      {
        type: "p",
        text: "Cash prizes are sent via Mobile Money (MoMo) to the number you link in your account settings. Data and airtime are delivered directly to your MTN number. Pulse Points land in your wallet instantly.",
      },
      { type: "h2", text: "Can I Get More Than One Spin?" },
      {
        type: "p",
        text: "Yes. Higher loyalty tiers unlock multiple spins per qualifying recharge — Gold members get 3 spins, Platinum gets 5, and Diamond gets 10. You can also purchase additional spins using your Pulse Points from the dashboard.",
      },
      { type: "h2", text: "Is There a Daily Limit?" },
      {
        type: "p",
        text: "Each recharge earns the spins appropriate to your tier. There's no artificial cap beyond that — if you recharge multiple times in a day, each qualifying recharge earns its own spins.",
      },
    ],
  },
  {
    slug:     "regional-wars-overview",
    date:     "February 28, 2026",
    tag:      "Features",
    tagColor: "text-purple-400 bg-purple-400/10 border-purple-400/20",
    title:    "Regional Wars: How Your State Can Win ₦250,000",
    excerpt:  "Regional Wars pits all 36 Nigerian states against each other every month. The state with the highest collective Pulse Points takes home ₦250,000.",
    readTime: "5 min read",
    content: [
      {
        type: "p",
        text: "Regional Wars is Loyalty Nexus's monthly competition where every active member contributes to their state's ranking — and the top three states share a ₦500,000 prize pool.",
      },
      { type: "h2", text: "How It Works" },
      {
        type: "p",
        text: "Every Pulse Point you earn from your MTN recharges is automatically added to your state's total. Your state is determined by the registration zone of your MTN number. There's nothing extra to do — just recharge as usual.",
      },
      {
        type: "p",
        text: "At the end of each month, the state totals are tallied and the top three states are announced.",
      },
      { type: "h2", text: "The Prize Pool" },
      {
        type: "list",
        items: [
          "🥇 1st Place — ₦250,000 (50% of the pool)",
          "🥈 2nd Place — ₦150,000 (30% of the pool)",
          "🥉 3rd Place — ₦100,000 (20% of the pool)",
        ],
      },
      { type: "h2", text: "The Individual Draw" },
      {
        type: "p",
        text: "Beyond the state prize, there is also a personal cash draw within each winning state. One member from each of the top three states is randomly selected to receive a direct Mobile Money payout. You don't need to be the top contributor — just be active.",
      },
      { type: "h2", text: "How to Check Your State's Ranking" },
      {
        type: "p",
        text: "Log in to your dashboard and navigate to the Regional Wars tab. You'll see a live leaderboard showing all 36 states and their current standings. The competition resets at the start of each calendar month.",
      },
    ],
  },
  {
    slug:     "loyalty-tiers-explained",
    date:     "February 20, 2026",
    tag:      "How It Works",
    tagColor: "text-cyan-400 bg-cyan-400/10 border-cyan-400/20",
    title:    "Bronze to Diamond: Understanding Your Loyalty Tier",
    excerpt:  "There are five loyalty tiers on Loyalty Nexus — and your tier affects how fast you earn points, how many spins you get, and which AI tools you can access.",
    readTime: "4 min read",
    content: [
      {
        type: "p",
        text: "Loyalty Nexus has five membership tiers: Bronze, Silver, Gold, Platinum, and Diamond. Your tier is determined by your total accumulated Pulse Points and grows as you keep recharging.",
      },
      { type: "h2", text: "The Five Tiers" },
      {
        type: "list",
        items: [
          "🥉 Bronze (0 – 4,999 points) — 1× point multiplier, 1 free spin per qualifying recharge.",
          "🥈 Silver (5,000 – 14,999 points) — 1.2× points, 2 free spins, all standard AI tools.",
          "🥇 Gold (15,000 – 39,999 points) — 1.5× points, 3 free spins, priority AI queue.",
          "💎 Platinum (40,000 – 99,999 points) — 2× points, 5 free spins, exclusive AI tools.",
          "💠 Diamond (100,000+ points) — 3× points, 10 free spins, all premium tools.",
        ],
      },
      { type: "h2", text: "How Do I Move Up?" },
      {
        type: "p",
        text: "Simply recharge. Every naira you put on your MTN line earns you points, and those points accumulate permanently. Tiers are calculated on lifetime Pulse Points — they don't reset monthly. Once you reach a tier, you keep it. Your current tier and progress to the next level are visible right on your dashboard.",
      },
    ],
  },
  {
    slug:     "ai-for-nigerians",
    date:     "February 12, 2026",
    tag:      "Vision",
    tagColor: "text-green-400 bg-green-400/10 border-green-400/20",
    title:    "Why We're Making AI Accessible to Every Nigerian",
    excerpt:  "Access to frontier AI tools in Nigeria has been limited by foreign payment requirements and high subscription costs. Loyalty Nexus is changing that.",
    readTime: "7 min read",
    content: [
      {
        type: "p",
        text: "The best AI tools in the world have largely been out of reach for most Nigerians. The barriers are real: payment forms that don't accept Nigerian cards, monthly fees priced in dollars, and tools designed for high-bandwidth connections.",
      },
      {
        type: "callout",
        text: "A Nigerian small business owner who could genuinely benefit from AI has had almost no accessible path to it — until now.",
      },
      { type: "h2", text: "Our Approach" },
      {
        type: "p",
        text: "Loyalty Nexus solves this by decoupling AI access from direct payment. Instead of asking users to pay a subscription, we let you earn access through something you're already doing — recharging your MTN line. Every ₦250 you recharge earns 1 Pulse Point, and those points unlock AI tools directly. No dollar payment. No foreign card. No subscription.",
      },
      { type: "h2", text: "What This Means in Practice" },
      {
        type: "list",
        items: [
          "A student can use AI to create a study guide for JAMB preparation without paying a dollar subscription.",
          "A market trader can generate a simple business plan for a loan application.",
          "A designer can remove backgrounds from product photos for their online shop.",
        ],
      },
      { type: "h2", text: "The Bigger Picture" },
      {
        type: "p",
        text: "We believe that making AI accessible isn't just a product decision — it's an economic one. The businesses, students, and creatives that gain access to these tools will create more value, earn more, and contribute more to the broader economy. Loyalty Nexus is our attempt to make that access as frictionless and rewarding as possible.",
      },
    ],
  },
];

// ─── Static params ───────────────────────────────────────────────────────────
export function generateStaticParams() {
  return POSTS.map((p) => ({ slug: p.slug }));
}

// ─── Metadata ────────────────────────────────────────────────────────────────
export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }): Promise<Metadata> {
  const { slug } = await params;
  const post = POSTS.find((p) => p.slug === slug);
  if (!post) return { title: "Post Not Found | Loyalty Nexus Blog" };
  return {
    title:       `${post.title} | Loyalty Nexus Blog`,
    description: post.excerpt,
  };
}

// ─── Content renderer ────────────────────────────────────────────────────────
function renderContent(blocks: typeof POSTS[0]["content"]) {
  return blocks.map((block, i) => {
    if (block.type === "h2") {
      return (
        <h2 key={i} className="text-xl sm:text-2xl font-black text-white mt-10 mb-4">
          {block.text}
        </h2>
      );
    }
    if (block.type === "p") {
      return (
        <p key={i} className="text-[15px] sm:text-[16px] text-white/60 leading-relaxed mb-5">
          {block.text}
        </p>
      );
    }
    if (block.type === "callout") {
      return (
        <div key={i} className="my-6 rounded-xl border border-yellow-400/25 bg-yellow-400/[0.05] px-6 py-5">
          <p className="text-[15px] font-bold text-yellow-400 leading-relaxed">{block.text}</p>
        </div>
      );
    }
    if (block.type === "list" && block.items) {
      return (
        <ul key={i} className="mb-5 space-y-2">
          {block.items.map((item, j) => (
            <li key={j} className="flex items-start gap-3 text-[15px] text-white/60 leading-relaxed">
              <span className="mt-[5px] w-1.5 h-1.5 rounded-full bg-yellow-400/60 flex-shrink-0" />
              {item}
            </li>
          ))}
        </ul>
      );
    }
    return null;
  });
}

// ─── Page ────────────────────────────────────────────────────────────────────
export default async function BlogPostPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const post = POSTS.find((p) => p.slug === slug);
  if (!post) notFound();

  const otherPosts = POSTS.filter((p) => p.slug !== slug).slice(0, 3);

  return (
    <main className="min-h-screen" style={{ background: "#0a0b0e", color: "#f0f2ff" }}>
      {/* Nav */}
      <div className="border-b border-white/[0.07]">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-yellow-400 flex items-center justify-center">
              <Zap className="w-3.5 h-3.5 text-black" />
            </div>
            <span className="font-black text-[14px]">
              <span className="text-yellow-400">Loyalty</span><span className="text-white"> Nexus</span>
            </span>
          </Link>
          <Link href="/blog" className="text-[13px] text-white/40 hover:text-white transition-colors">← All posts</Link>
        </div>
      </div>

      {/* Article */}
      <article className="max-w-2xl mx-auto px-4 sm:px-6 py-16">
        {/* Tag + meta */}
        <div className="flex flex-wrap items-center gap-3 mb-6">
          <span className={`text-[11px] font-bold px-2.5 py-1 rounded-full border ${post.tagColor}`}>
            {post.tag}
          </span>
          <span className="text-[12px] text-white/30">{post.date}</span>
          <span className="text-[12px] text-white/30">·</span>
          <span className="text-[12px] text-white/30">{post.readTime}</span>
        </div>

        {/* Title */}
        <h1 className="text-3xl sm:text-4xl font-black text-white leading-tight mb-6">
          {post.title}
        </h1>

        {/* Excerpt */}
        <p className="text-[17px] text-white/50 leading-relaxed mb-10 border-b border-white/[0.06] pb-10">
          {post.excerpt}
        </p>

        {/* Body */}
        <div>{renderContent(post.content)}</div>
      </article>

      {/* More posts */}
      {otherPosts.length > 0 && (
        <section className="max-w-2xl mx-auto px-4 sm:px-6 pb-20 border-t border-white/[0.06] pt-12">
          <h2 className="text-[16px] font-black text-white/60 mb-6 uppercase tracking-widest text-sm">More Articles</h2>
          <div className="space-y-4">
            {otherPosts.map((p) => (
              <Link key={p.slug} href={`/blog/${p.slug}`}>
                <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-4 hover:border-white/[0.15] transition-colors">
                  <div className="flex items-center gap-2 mb-1.5">
                    <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${p.tagColor}`}>{p.tag}</span>
                    <span className="text-[11px] text-white/25">{p.readTime}</span>
                  </div>
                  <p className="text-[14px] font-bold text-white leading-snug">{p.title}</p>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}

      {/* CTA */}
      <div className="max-w-2xl mx-auto px-4 sm:px-6 pb-20 text-center">
        <div className="rounded-2xl border border-yellow-400/20 bg-yellow-400/[0.03] p-8">
          <p className="text-[15px] font-bold text-white mb-2">Ready to start earning?</p>
          <p className="text-[13px] text-white/40 mb-5">Create your free account and earn Pulse Points on your next MTN recharge.</p>
          <Link
            href="/"
            className="inline-block px-6 py-3 rounded-lg bg-yellow-400 text-black font-black text-[13px] hover:bg-yellow-300 transition-colors"
          >
            Get Started Free →
          </Link>
        </div>
      </div>
    </main>
  );
}
