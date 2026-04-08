"use client";

/**
 * NexusChatUI — Best-in-class conversational interface for all Nexus AI chat tools.
 *
 * Used by: Ask Nexus, Code Helper, Code Pro, Research Brief, Deep Research,
 *          Mind Map, Study Guide, Quiz Me, Business Plan, Doc Analyser,
 *          Voice to Plan, Local Translation, Web Search AI, Nexus Agent,
 *          Localize UI, Image Analyser (vision mode)
 *
 * Design goals:
 *  • Clean, content-first — no gold gradients inside message bubbles
 *  • Streaming simulation — word-by-word reveal makes responses feel alive
 *  • Tool-aware — each tool gets its own welcome message + suggestion chips
 *  • Full markdown + VS Code-style code blocks
 *  • File, image, link attachments
 *  • Voice input (Web Speech API)
 *  • Mobile-first, buttery scroll
 */

import React, {
  useState, useRef, useEffect, useCallback, useMemo,
} from "react";
import {
  ArrowLeft, Send, Paperclip, Link2, Mic, MicOff,
  Copy, Check, Download, X, FileText, Globe, Loader2,
  RotateCcw, ChevronDown, Sparkles, Zap,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useSpeechToText } from "@/hooks/useSpeechToText";
import api from "@/lib/api";
import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  displayContent: string; // streamed-in portion
  isStreaming: boolean;
  timestamp: Date;
  provider?: string;
  attachedName?: string;
}

// ─── Tool config ─────────────────────────────────────────────────────────────

interface ToolConfig {
  label: string;
  icon: string;
  color: string;         // accent hex
  gradient: string;      // tailwind gradient classes for avatar
  welcome: string;
  placeholder: string;
  suggestions: string[];
}

const TOOL_CONFIGS: Record<string, ToolConfig> = {
  "ask-nexus": {
    label: "Ask Nexus",
    icon: "✦",
    color: "#a78bfa",
    gradient: "from-violet-500/30 to-purple-600/20",
    welcome: "Hi! I'm **Nexus**, your personal AI assistant.\n\nI can help with business, research, writing, coding, education — anything really. You can also attach a file or paste a link and I'll read it for you.\n\nWhat's on your mind?",
    placeholder: "Ask me anything…",
    suggestions: [
      "Write a business plan for a food delivery startup in Lagos",
      "Explain blockchain in simple terms",
      "Help me write a professional email to a client",
      "What are the best investment options in Nigeria right now?",
    ],
  },
  "nexus-chat": {
    label: "Ask Nexus",
    icon: "✦",
    color: "#a78bfa",
    gradient: "from-violet-500/30 to-purple-600/20",
    welcome: "Hi! I'm **Nexus**, your personal AI assistant.\n\nWhat's on your mind?",
    placeholder: "Ask me anything…",
    suggestions: [
      "Summarise the key differences between WAEC and JAMB",
      "Write a Python script to sort a list of names",
      "What are the best investment options in Nigeria?",
      "Help me write a speech for my business launch",
    ],
  },
  "code-helper": {
    label: "Code Helper",
    icon: "⌥",
    color: "#86efac",
    gradient: "from-green-500/30 to-emerald-600/20",
    welcome: "Hey! I'm your **code assistant**.\n\nPaste your code, describe a bug, or tell me what you want to build — I'll write it, explain it, and help you debug it in any language.\n\nWhat are we building?",
    placeholder: "Describe what you want to build, or paste your code…",
    suggestions: [
      "Write a REST API in Go that handles user authentication",
      "Debug this React component — it's re-rendering too often",
      "Explain the difference between async/await and Promises",
      "Write a SQL query to find the top 10 customers by revenue",
    ],
  },
  "code-pro": {
    label: "Code Pro",
    icon: "⌥",
    color: "#86efac",
    gradient: "from-green-500/30 to-emerald-600/20",
    welcome: "**Code Pro** here — I do deeper analysis, architecture reviews, and can read screenshots of errors or UI bugs.\n\nAttach an image of your error message or describe your problem.",
    placeholder: "Describe a bug, paste code, or attach a screenshot…",
    suggestions: [
      "Review this architecture and suggest improvements",
      "I'm getting a CORS error in my Next.js app — here's the error",
      "Write a complete TypeScript type system for this API response",
      "Optimise this database query — it's too slow",
    ],
  },
  "research-brief": {
    label: "Research Brief",
    icon: "◎",
    color: "#67e8f9",
    gradient: "from-cyan-500/30 to-sky-600/20",
    welcome: "I'm your **research analyst**.\n\nGive me a topic, question, or industry and I'll produce a structured, well-sourced research brief — with key findings, data points, and a clear summary.\n\nWhat do you need researched?",
    placeholder: "Enter a topic, question, or industry to research…",
    suggestions: [
      "Research the fintech landscape in West Africa 2024",
      "What are the main causes of youth unemployment in Nigeria?",
      "Analyse the competitive landscape for e-commerce in Lagos",
      "Research best practices for mobile app onboarding",
    ],
  },
  "deep-research-brief": {
    label: "Deep Research",
    icon: "◎",
    color: "#67e8f9",
    gradient: "from-cyan-500/30 to-sky-600/20",
    welcome: "**Deep Research mode** — I go broader and deeper than a standard brief.\n\nGive me your topic and I'll produce a comprehensive, structured report with multiple perspectives.\n\nWhat's your research question?",
    placeholder: "Enter a complex topic for deep research…",
    suggestions: [
      "Deep research on the impact of AI on Nigerian banking",
      "Comprehensive analysis of renewable energy in Sub-Saharan Africa",
      "Research the full history and future of the Naira",
      "Deep dive into mobile money adoption patterns in Africa",
    ],
  },
  "mind-map": {
    label: "Mind Map",
    icon: "⬡",
    color: "#fbbf24",
    gradient: "from-amber-500/30 to-yellow-600/20",
    welcome: "I'll help you **map out any topic** — breaking it into branches, sub-topics, and connections.\n\nGive me a topic and I'll structure a detailed mind map you can explore and expand.",
    placeholder: "Enter a topic to mind-map…",
    suggestions: [
      "Mind map: Starting a restaurant business in Lagos",
      "Mind map: How the internet works",
      "Mind map: Keys to a healthy lifestyle",
      "Mind map: Digital marketing strategies",
    ],
  },
  "mindmap": {
    label: "Mind Map",
    icon: "⬡",
    color: "#fbbf24",
    gradient: "from-amber-500/30 to-yellow-600/20",
    welcome: "I'll help you **map out any topic**.\n\nGive me a topic and I'll build a structured mind map.",
    placeholder: "Enter a topic to mind-map…",
    suggestions: [
      "Mind map: Building a startup",
      "Mind map: Studying for WAEC",
      "Mind map: Personal finance basics",
      "Mind map: Social media marketing",
    ],
  },
  "study-guide": {
    label: "Study Guide",
    icon: "📖",
    color: "#f472b6",
    gradient: "from-pink-500/30 to-rose-600/20",
    welcome: "I'm your **study companion**.\n\nGive me any subject, topic, or exam you're preparing for and I'll create a structured study guide — with key concepts, explanations, examples, and tips.\n\nWhat are you studying?",
    placeholder: "Enter a subject, topic, or exam name…",
    suggestions: [
      "Study guide for WAEC Economics",
      "Explain photosynthesis for a secondary school student",
      "Study guide: Introduction to programming concepts",
      "Help me understand how the Nigerian government works",
    ],
  },
  "quiz": {
    label: "Quiz Me",
    icon: "❓",
    color: "#f97316",
    gradient: "from-orange-500/30 to-amber-600/20",
    welcome: "Ready to **test your knowledge**? 🎯\n\nTell me a subject and difficulty level and I'll generate a quiz for you. I'll give you questions one by one and keep score.\n\nWhat do you want to be quizzed on?",
    placeholder: "Enter a subject and difficulty (easy / medium / hard)…",
    suggestions: [
      "Quiz me on Nigerian history — medium difficulty",
      "10 questions on basic Python programming",
      "Test my knowledge of human anatomy",
      "Quiz on JAMB Economics — hard",
    ],
  },
  "quiz-me": {
    label: "Quiz Me",
    icon: "❓",
    color: "#f97316",
    gradient: "from-orange-500/30 to-amber-600/20",
    welcome: "Ready to **test your knowledge**? 🎯\n\nWhat subject and difficulty?",
    placeholder: "Enter a subject and difficulty…",
    suggestions: [
      "Quiz me on Nigerian history — medium",
      "10 Python programming questions",
      "Human anatomy quiz — hard",
      "WAEC Biology — easy warm-up",
    ],
  },
  "bizplan": {
    label: "Business Plan",
    icon: "📊",
    color: "#34d399",
    gradient: "from-emerald-500/30 to-teal-600/20",
    welcome: "I'll help you write a **professional business plan** tailored for the Nigerian market.\n\nTell me about your business idea — what it is, who it's for, and your startup budget if you have one.\n\nLet's build something great.",
    placeholder: "Describe your business idea, target market, and location…",
    suggestions: [
      "Business plan for a laundry delivery service in Abuja",
      "Online tutoring platform for secondary school students in Lagos",
      "Agritech startup connecting farmers to buyers in rural Nigeria",
      "Fashion e-commerce store targeting young Nigerians",
    ],
  },
  "business-plan-summary": {
    label: "Business Plan",
    icon: "📊",
    color: "#34d399",
    gradient: "from-emerald-500/30 to-teal-600/20",
    welcome: "Let's build your **business plan**.\n\nTell me your idea, target market, and city.",
    placeholder: "Describe your business idea…",
    suggestions: [
      "Business plan for a food delivery startup in Port Harcourt",
      "Online fashion store for Nigerian youth",
      "Logistics company connecting SMEs in Lagos",
      "EdTech platform for primary school children",
    ],
  },
  "doc-analyzer": {
    label: "Doc Analyser",
    icon: "📄",
    color: "#a5b4fc",
    gradient: "from-indigo-500/30 to-violet-600/20",
    welcome: "I can **analyse any document** — contracts, invoices, reports, PDFs, and more.\n\nUpload a file or paste text and tell me what you want to know. I can summarise, extract key points, compare sections, or answer specific questions.\n\nWhat would you like to analyse?",
    placeholder: "Upload a document or paste text and ask your question…",
    suggestions: [
      "Summarise the key points of this contract",
      "What are the payment terms in this invoice?",
      "Find all risk clauses in this document",
      "Compare section 3 and section 7 of this report",
    ],
  },
  "voice-to-plan": {
    label: "Voice to Plan",
    icon: "🎙️",
    color: "#fb7185",
    gradient: "from-rose-500/30 to-pink-600/20",
    welcome: "**Voice to Plan** — speak or type your rough idea and I'll turn it into a structured action plan.\n\nRamble, brainstorm, think out loud — I'll organise it into clear steps.\n\nWhat's your idea or goal?",
    placeholder: "Describe your idea, goal, or plan in any way…",
    suggestions: [
      "I want to start a business but don't know where to begin",
      "I have a YouTube channel idea about Nigerian street food",
      "Plan a community event for 500 people in Ikeja",
      "I want to save ₦500,000 in 6 months — help me plan",
    ],
  },
  "local-translation": {
    label: "Local Translation",
    icon: "🌍",
    color: "#22d3ee",
    gradient: "from-cyan-500/30 to-teal-600/20",
    welcome: "I can translate between **English, Hausa, Yoruba, Igbo** and other Nigerian languages.\n\nJust type your text and tell me the target language. I can also explain idioms and cultural context.\n\nWhat would you like to translate?",
    placeholder: "Type your text and specify the target language…",
    suggestions: [
      "Translate to Yoruba: Welcome to our store, how can I help you?",
      "Translate to Hausa: Good morning, I'd like to make a payment",
      "Translate to Igbo: Thank you for your business",
      "How do you say 'I love you' in Yoruba with context?",
    ],
  },
  "web-search-ai": {
    label: "Web Search AI",
    icon: "⚡",
    color: "#38bdf8",
    gradient: "from-sky-500/30 to-blue-600/20",
    welcome: "I search the **live internet** to give you accurate, up-to-date answers — not just training data.\n\nAsk me anything that needs current information: news, prices, events, trends, people.\n\nWhat do you want to search?",
    placeholder: "Ask anything that needs current information…",
    suggestions: [
      "What's the current USD to NGN exchange rate?",
      "Latest news about the Nigerian tech industry today",
      "Current price of Dangote Cement shares",
      "What events are happening in Lagos this weekend?",
    ],
  },
  "nexus-agent": {
    label: "Nexus Agent",
    icon: "⬡",
    color: "#c084fc",
    gradient: "from-purple-500/30 to-fuchsia-600/20",
    welcome: "I'm **Nexus Agent** — I can handle complex, multi-step tasks that regular chat can't.\n\nDescribe a complex goal: research + summarise + format, scrape + organise + compare, or any multi-part workflow. I'll break it down and execute each step.\n\nWhat complex task can I tackle for you?",
    placeholder: "Describe a complex, multi-step task…",
    suggestions: [
      "Research top 5 competitors in Nigerian fintech and compare their features",
      "Find the best logistics providers in Lagos, compare rates and reviews",
      "Research, summarise and structure a report on Nigerian agriculture 2024",
      "Analyse the market for electric vehicles in Nigeria",
    ],
  },
  "localize-ui": {
    label: "Localise UI",
    icon: "🔤",
    color: "#6ee7b7",
    gradient: "from-emerald-500/30 to-green-600/20",
    welcome: "I'll help you **localise your app or website** into Nigerian languages.\n\nUpload a screenshot of your UI or paste your text strings and I'll produce a translation table with Hausa, Yoruba, and Igbo equivalents — plus cultural notes.\n\nWhat would you like to localise?",
    placeholder: "Paste your UI text strings or describe your app…",
    suggestions: [
      "Translate this app navigation: Home, Profile, Settings, Notifications, Help",
      "Localise onboarding screens for a fintech app",
      "Translate error messages for a Nigerian audience",
      "Help me localise marketing copy for Lagos users",
    ],
  },
  "image-analyser": {
    label: "Image Analyser",
    icon: "👁",
    color: "#e879f9",
    gradient: "from-fuchsia-500/30 to-pink-600/20",
    welcome: "I can **see and analyse any image**.\n\nPaste an image URL or attach a file and ask me anything about it — describe it, read text in it, identify objects, analyse charts, check designs.\n\nWhat would you like me to look at?",
    placeholder: "Paste an image URL or attach a file, then ask your question…",
    suggestions: [
      "Describe everything you see in this image",
      "Read all the text in this screenshot",
      "What's wrong with this UI design?",
      "Identify the products in this photo",
    ],
  },
};

const DEFAULT_CONFIG: ToolConfig = TOOL_CONFIGS["ask-nexus"];

// ─── Session ID ───────────────────────────────────────────────────────────────

function getOrCreateSessionId(toolSlug: string): string {
  if (typeof window === "undefined") return "";
  const key = `nexus_chat_session_${toolSlug}`;
  const existing = localStorage.getItem(key);
  if (existing) return existing;
  const id = "sess_" + Date.now().toString(36) + "_" + toolSlug;
  localStorage.setItem(key, id);
  return id;
}

// ─── Streaming simulation ─────────────────────────────────────────────────────

function useStreamText(
  full: string,
  enabled: boolean,
  onDone: () => void,
): string {
  const [displayed, setDisplayed] = useState("");
  const frameRef = useRef<number | null>(null);
  const indexRef = useRef(0);

  useEffect(() => {
    if (!enabled || !full) return;
    indexRef.current = 0;
    setDisplayed("");

    // Variable speed — faster for code blocks, slower for prose
    const tick = () => {
      if (indexRef.current >= full.length) {
        setDisplayed(full);
        onDone();
        return;
      }
      // Chunk: 2-6 chars per frame for prose feel
      const isCode = full.indexOf("```", indexRef.current) === indexRef.current;
      const step = isCode ? 8 : 3;
      indexRef.current = Math.min(indexRef.current + step, full.length);
      setDisplayed(full.slice(0, indexRef.current));
      frameRef.current = requestAnimationFrame(tick);
    };
    frameRef.current = requestAnimationFrame(tick);

    return () => {
      if (frameRef.current) cancelAnimationFrame(frameRef.current);
    };
  }, [full, enabled]); // eslint-disable-line react-hooks/exhaustive-deps

  return enabled ? displayed : full;
}

// ─── Code language colours ────────────────────────────────────────────────────

const LANG_COLORS: Record<string, { bg: string; text: string; dot: string }> = {
  python:     { bg: "bg-blue-500/20",   text: "text-blue-300",   dot: "bg-blue-400" },
  javascript: { bg: "bg-yellow-500/20", text: "text-yellow-300", dot: "bg-yellow-400" },
  typescript: { bg: "bg-blue-600/20",   text: "text-blue-200",   dot: "bg-blue-300" },
  js:         { bg: "bg-yellow-500/20", text: "text-yellow-300", dot: "bg-yellow-400" },
  ts:         { bg: "bg-blue-600/20",   text: "text-blue-200",   dot: "bg-blue-300" },
  html:       { bg: "bg-orange-500/20", text: "text-orange-300", dot: "bg-orange-400" },
  css:        { bg: "bg-sky-500/20",    text: "text-sky-300",    dot: "bg-sky-400" },
  sql:        { bg: "bg-cyan-500/20",   text: "text-cyan-300",   dot: "bg-cyan-400" },
  bash:       { bg: "bg-green-500/20",  text: "text-green-300",  dot: "bg-green-400" },
  go:         { bg: "bg-teal-500/20",   text: "text-teal-300",   dot: "bg-teal-400" },
  rust:       { bg: "bg-orange-600/20", text: "text-orange-200", dot: "bg-orange-300" },
  java:       { bg: "bg-red-500/20",    text: "text-red-300",    dot: "bg-red-400" },
  python3:    { bg: "bg-blue-500/20",   text: "text-blue-300",   dot: "bg-blue-400" },
  json:       { bg: "bg-amber-500/20",  text: "text-amber-300",  dot: "bg-amber-400" },
  yaml:       { bg: "bg-pink-500/20",   text: "text-pink-300",   dot: "bg-pink-400" },
};
const LANG_DEFAULT = { bg: "bg-white/8", text: "text-white/40", dot: "bg-white/30" };

function getLangExt(lang: string): string {
  const m: Record<string, string> = {
    python: "py", javascript: "js", typescript: "ts", html: "html",
    css: "css", sql: "sql", bash: "sh", go: "go", rust: "rs",
    java: "java", json: "json", yaml: "yml", markdown: "md",
  };
  return m[lang.toLowerCase()] ?? "txt";
}

// ─── Rich markdown renderer ───────────────────────────────────────────────────

function RichMessage({ content }: { content: string }) {
  const [copied, setCopied] = useState<number | null>(null);

  const copyCode = (code: string, idx: number) => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(idx);
      setTimeout(() => setCopied(null), 1800);
    });
  };

  const parts = content.split(/(```[\s\S]*?```)/g);

  return (
    <div className="space-y-3">
      {parts.map((part, i) => {
        if (part.startsWith("```")) {
          const firstNl = part.indexOf("\n");
          const lang = part.slice(3, firstNl).trim().toLowerCase() || "code";
          const code = part.slice(firstNl + 1, part.lastIndexOf("```")).trim();
          const lc = LANG_COLORS[lang] ?? LANG_DEFAULT;
          const lines = code.split("\n");
          return (
            <div key={i} className="rounded-xl overflow-hidden border border-white/10 shadow-xl shadow-black/40 my-2">
              {/* Header bar */}
              <div className="flex items-center justify-between px-3 py-2 bg-[#0d0d14] border-b border-white/[0.07]">
                <div className="flex items-center gap-2">
                  <div className="flex gap-1.5">
                    <span className="w-2.5 h-2.5 rounded-full bg-red-500/50" />
                    <span className="w-2.5 h-2.5 rounded-full bg-yellow-500/50" />
                    <span className="w-2.5 h-2.5 rounded-full bg-green-500/50" />
                  </div>
                  <span className={cn("text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider flex items-center gap-1", lc.bg, lc.text)}>
                    <span className={cn("inline-block w-1.5 h-1.5 rounded-full", lc.dot)} />
                    {lang}
                  </span>
                  <span className="text-white/20 text-[10px]">{lines.length}L</span>
                </div>
                <button
                  onClick={() => copyCode(code, i)}
                  className="flex items-center gap-1 text-[10px] text-white/30 hover:text-white/70 transition-colors px-2 py-1 rounded-lg hover:bg-white/[0.06]"
                >
                  {copied === i ? <Check size={10} className="text-green-400" /> : <Copy size={10} />}
                  {copied === i ? "Copied!" : "Copy"}
                </button>
              </div>
              {/* Code body */}
              <div className="bg-[#080810] overflow-x-auto max-h-72 overflow-y-auto">
                <table className="w-full text-[11.5px] font-mono leading-[1.7]">
                  <tbody>
                    {lines.map((line, li) => (
                      <tr key={li} className="hover:bg-white/[0.025] transition-colors">
                        <td className="select-none text-right pr-4 pl-3 text-white/15 w-8 border-r border-white/[0.04] align-top">
                          {li + 1}
                        </td>
                        <td className="pl-4 pr-3 text-emerald-100/80 whitespace-pre align-top">
                          {line || " "}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          );
        }

        // Plain text with markdown
        const lines = part.split("\n");
        return (
          <div key={i} className="space-y-[5px]">
            {lines.map((line, j) => {
              if (!line.trim()) return <div key={j} className="h-2" />;

              if (line.startsWith("### ")) return <h3 key={j} className="text-white font-semibold text-[13px] mt-3 mb-0.5 leading-snug">{renderInline(line.slice(4))}</h3>;
              if (line.startsWith("## "))  return <h2 key={j} className="text-white font-semibold text-sm mt-4 mb-1 leading-snug">{renderInline(line.slice(3))}</h2>;
              if (line.startsWith("# "))   return <h1 key={j} className="text-white font-bold text-base mt-4 mb-1.5 leading-snug">{renderInline(line.slice(2))}</h1>;
              if (line.startsWith("---"))  return <hr key={j} className="border-white/10 my-3" />;

              const isBullet = /^[-*•]\s/.test(line);
              const isNum    = /^\d+\.\s/.test(line);
              const text     = isBullet ? line.replace(/^[-*•]\s/, "") : isNum ? line.replace(/^\d+\.\s/, "") : line;
              const num      = line.match(/^(\d+)\./)?.[1];

              if (isBullet) return (
                <div key={j} className="flex items-start gap-2.5 pl-1">
                  <span className="mt-[7px] w-[5px] h-[5px] rounded-full flex-shrink-0 bg-white/30" />
                  <p className="text-[13.5px] leading-[1.7] text-white/80 flex-1">{renderInline(text)}</p>
                </div>
              );
              if (isNum) return (
                <div key={j} className="flex items-start gap-2.5 pl-1">
                  <span className="mt-[3px] flex-shrink-0 text-[11px] font-bold text-white/30 w-5 text-right">{num}.</span>
                  <p className="text-[13.5px] leading-[1.7] text-white/80 flex-1">{renderInline(text)}</p>
                </div>
              );
              return <p key={j} className="text-[13.5px] leading-[1.7] text-white/80">{renderInline(line)}</p>;
            })}
          </div>
        );
      })}
    </div>
  );
}

function renderInline(text: string) {
  const chunks = text.split(/(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*)/g);
  return chunks.map((c, k) => {
    if (c.startsWith("**") && c.endsWith("**"))
      return <strong key={k} className="text-white font-semibold">{c.slice(2, -2)}</strong>;
    if (c.startsWith("*") && c.endsWith("*") && c.length > 2)
      return <em key={k} className="text-white/70 italic">{c.slice(1, -1)}</em>;
    if (c.startsWith("`") && c.endsWith("`"))
      return <code key={k} className="text-[11.5px] font-mono px-1.5 py-0.5 rounded-md bg-white/10 text-amber-200/90">{c.slice(1, -1)}</code>;
    return c;
  });
}

// ─── Streaming message wrapper ────────────────────────────────────────────────

function StreamingMessage({
  content, isStreaming, onStreamDone,
}: {
  content: string; isStreaming: boolean; onStreamDone: () => void;
}) {
  const displayed = useStreamText(content, isStreaming, onStreamDone);
  return <RichMessage content={displayed} />;
}

// ─── Typing indicator ─────────────────────────────────────────────────────────

function TypingDots({ color }: { color: string }) {
  return (
    <div className="flex items-center gap-1 py-1">
      {[0, 120, 240].map((d) => (
        <motion.div
          key={d}
          className="w-[7px] h-[7px] rounded-full"
          style={{ backgroundColor: color + "80" }}
          animate={{ scale: [1, 1.4, 1], opacity: [0.4, 1, 0.4] }}
          transition={{ duration: 0.9, delay: d / 1000, repeat: Infinity }}
        />
      ))}
    </div>
  );
}

// ─── Suggestion cards ─────────────────────────────────────────────────────────

function SuggestionCards({
  suggestions, onSelect, color,
}: {
  suggestions: string[]; onSelect: (s: string) => void; color: string;
}) {
  return (
    <div className="grid grid-cols-2 gap-2 px-4 pb-2">
      {suggestions.map((s, i) => (
        <motion.button
          key={i}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: i * 0.07 }}
          onClick={() => onSelect(s)}
          className="text-left p-3 rounded-2xl border border-white/[0.08] bg-white/[0.03] hover:bg-white/[0.07] hover:border-white/20 transition-all duration-150 active:scale-[0.97] group"
        >
          <p className="text-[12px] leading-snug text-white/55 group-hover:text-white/80 transition-colors line-clamp-3">{s}</p>
        </motion.button>
      ))}
    </div>
  );
}

// ─── Props ────────────────────────────────────────────────────────────────────

interface NexusChatUIProps {
  toolSlug?: string;
  onClose?: () => void;
  /** Called when user clicks back — if not provided, onClose is used */
  onBack?: () => void;
  /** If opened as a full page (no back button needed from drawer) */
  fullPage?: boolean;
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function NexusChatUI({
  toolSlug = "ask-nexus",
  onClose,
  onBack,
  fullPage = false,
}: NexusChatUIProps) {
  const cfg: ToolConfig = TOOL_CONFIGS[toolSlug] ?? DEFAULT_CONFIG;

  const [messages, setMessages] = useState<Message[]>([
    {
      id: "welcome",
      role: "assistant",
      content: cfg.welcome,
      displayContent: cfg.welcome,
      isStreaming: false,
      timestamp: new Date(),
    },
  ]);

  const [input, setInput]           = useState("");
  const [isLoading, setIsLoading]   = useState(false);
  const [msgCount, setMsgCount]     = useState(0);
  const [msgLimit, setMsgLimit]     = useState(20);
  const [copiedId, setCopiedId]     = useState<string | null>(null);
  const [showScrollBtn, setShowScrollBtn] = useState(false);
  const [streamingId, setStreamingId] = useState<string | null>(null);

  // Attachment state
  const [attachedFile,    setAttachedFile]    = useState<File | null>(null);
  const [attachedFileURL, setAttachedFileURL] = useState("");
  const [isUploading,     setIsUploading]     = useState(false);
  const [uploadError,     setUploadError]     = useState("");
  const [showLinkInput,   setShowLinkInput]   = useState(false);
  const [linkInput,       setLinkInput]       = useState("");
  const [attachedLink,    setAttachedLink]    = useState("");

  const sessionId      = useRef("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollAreaRef  = useRef<HTMLDivElement>(null);
  const textareaRef    = useRef<HTMLTextAreaElement>(null);
  const fileInputRef   = useRef<HTMLInputElement>(null);
  const linkInputRef   = useRef<HTMLInputElement>(null);

  const { speechState, speechError, interimText, handleMicClick, isMicBusy } =
    useSpeechToText({
      onTranscript: (t) => {
        setInput((p) => (p ? `${p} ${t}` : t));
        setTimeout(() => textareaRef.current?.focus(), 100);
      },
      language: "en-US",
    });

  // ── Init ──────────────────────────────────────────────────────────────────
  useEffect(() => {
    sessionId.current = getOrCreateSessionId(toolSlug);
    api.getChatUsage().then((r: unknown) => {
      const d = r as { used: number; limit: number };
      setMsgCount(d.used ?? 0);
      setMsgLimit(d.limit ?? 20);
    }).catch(() => {});
  }, [toolSlug]);

  // ── Scroll management ─────────────────────────────────────────────────────
  const scrollToBottom = useCallback((smooth = true) => {
    messagesEndRef.current?.scrollIntoView({ behavior: smooth ? "smooth" : "instant" });
  }, []);

  useEffect(() => { scrollToBottom(); }, [messages, scrollToBottom]);

  useEffect(() => {
    const el = scrollAreaRef.current;
    if (!el) return;
    const handler = () => {
      const diff = el.scrollHeight - el.scrollTop - el.clientHeight;
      setShowScrollBtn(diff > 120);
    };
    el.addEventListener("scroll", handler);
    return () => el.removeEventListener("scroll", handler);
  }, []);

  // ── Textarea auto-resize ──────────────────────────────────────────────────
  useEffect(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.style.height = "auto";
    ta.style.height = Math.min(ta.scrollHeight, 140) + "px";
  }, [input]);

  useEffect(() => {
    if (showLinkInput) linkInputRef.current?.focus();
  }, [showLinkInput]);

  // ── File upload ───────────────────────────────────────────────────────────
  async function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 20 * 1024 * 1024) { setUploadError("Max file size is 20 MB"); return; }
    setUploadError("");
    setAttachedFile(file);
    setAttachedLink("");
    setIsUploading(true);
    try {
      const r = await api.uploadAsset(file);
      setAttachedFileURL(r.url);
    } catch (err: unknown) {
      setUploadError(err instanceof Error ? err.message : "Upload failed");
      setAttachedFile(null);
      setAttachedFileURL("");
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  function clearAttachment() {
    setAttachedFile(null); setAttachedFileURL(""); setAttachedLink(""); setUploadError("");
    if (fileInputRef.current) fileInputRef.current.value = "";
  }

  function confirmLink() {
    const u = linkInput.trim();
    if (!u) return;
    setAttachedLink(u); setAttachedFile(null); setAttachedFileURL("");
    setLinkInput(""); setShowLinkInput(false);
  }

  // ── Send ──────────────────────────────────────────────────────────────────
  const handleSend = useCallback(async (text?: string) => {
    const msg = (text ?? input).trim();
    if (!msg || isLoading || isUploading) return;

    const displayName = attachedFile?.name
      ?? (attachedLink ? (() => { try { return new URL(attachedLink.startsWith("http") ? attachedLink : "https://" + attachedLink).hostname; } catch { return attachedLink; } })() : undefined);

    const userMsg: Message = {
      id: Date.now().toString(),
      role: "user",
      content: msg,
      displayContent: msg,
      isStreaming: false,
      timestamp: new Date(),
      attachedName: displayName,
    };

    setMessages((p) => [...p, userMsg]);
    setInput("");
    const fileURL = attachedFileURL;
    const linkURL = attachedLink;
    const fileName = attachedFile?.name ?? "";
    clearAttachment();
    setIsLoading(true);

    try {
      const res = await api.sendChat(
        msg, sessionId.current, toolSlug,
        undefined, undefined,
        fileURL || undefined,
        linkURL || undefined,
        fileName || undefined,
      ) as { response: string; provider?: string; session_id?: string; message_count?: number };

      if (res.session_id) {
        sessionId.current = res.session_id;
        try { localStorage.setItem(`nexus_chat_session_${toolSlug}`, res.session_id); } catch { /**/ }
      }
      if (res.message_count !== undefined) setMsgCount(res.message_count);

      const aiId = (Date.now() + 1).toString();
      setStreamingId(aiId);
      setMessages((p) => [...p, {
        id: aiId,
        role: "assistant",
        content: res.response,
        displayContent: "",
        isStreaming: true,
        timestamp: new Date(),
        provider: res.provider?.toUpperCase(),
      }]);
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : "Request failed";
      setMessages((p) => [...p, {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content: `⚠️ ${errMsg} — please try again shortly.`,
        displayContent: `⚠️ ${errMsg} — please try again shortly.`,
        isStreaming: false,
        timestamp: new Date(),
      }]);
    } finally {
      setIsLoading(false);
    }
  }, [input, isLoading, isUploading, attachedFile, attachedFileURL, attachedLink, toolSlug]);

  function copyMsg(content: string, id: string) {
    navigator.clipboard.writeText(content).then(() => {
      setCopiedId(id); setTimeout(() => setCopiedId(null), 1800);
    });
  }

  const remaining      = Math.max(0, msgLimit - msgCount);
  const showSuggestions = messages.length === 1 && !isLoading;
  const hasAttachment   = !!attachedFile || !!attachedLink;
  const isBusy          = isLoading || isUploading || isMicBusy;
  const canSend         = input.trim().length > 0 && !isBusy && remaining > 0;
  const micRecording    = speechState === "listening";
  const micProcessing   = speechState === "processing";

  const handleBack = onBack ?? onClose;

  return (
    <div className="flex flex-col h-full bg-[#0c0c10] text-white overflow-hidden">

      {/* ── Header ───────────────────────────────────────────────────────── */}
      <div className="flex items-center gap-3 px-4 pt-4 pb-3 border-b border-white/[0.06] flex-shrink-0">
        {handleBack && (
          <button
            onClick={handleBack}
            className="w-8 h-8 rounded-xl flex items-center justify-center text-white/40 hover:text-white/80 hover:bg-white/[0.07] transition-all flex-shrink-0"
          >
            <ArrowLeft size={18} />
          </button>
        )}

        {/* Tool avatar */}
        <div
          className={cn("w-9 h-9 rounded-2xl flex items-center justify-center flex-shrink-0 text-base font-bold bg-gradient-to-br", cfg.gradient)}
          style={{ color: cfg.color }}
        >
          {cfg.icon}
        </div>

        <div className="flex-1 min-w-0">
          <h1 className="text-[15px] font-semibold text-white leading-none truncate">{cfg.label}</h1>
          <div className="flex items-center gap-1.5 mt-0.5">
            <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
            <span className="text-[10px] text-white/30 font-medium">Online</span>
          </div>
        </div>

        {/* Message counter — only show when close to limit */}
        {remaining <= 5 && remaining > 0 && (
          <div className="text-[10px] text-amber-400/70 font-medium px-2 py-1 rounded-lg bg-amber-400/10 border border-amber-400/20">
            {remaining} left
          </div>
        )}
        {remaining === 0 && (
          <div className="text-[10px] text-red-400/80 font-medium px-2 py-1 rounded-lg bg-red-400/10 border border-red-400/20">
            Limit reached
          </div>
        )}
      </div>

      {/* ── Messages ─────────────────────────────────────────────────────── */}
      <div ref={scrollAreaRef} className="flex-1 overflow-y-auto overscroll-contain px-4 py-3 space-y-5 scroll-smooth"
        style={{ scrollbarWidth: "none" }}>

        {/* Suggestion cards */}
        <AnimatePresence>
          {showSuggestions && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              className="pt-2"
            >
              <p className="text-center text-[11px] text-white/20 font-medium uppercase tracking-widest mb-3">Try asking</p>
              <SuggestionCards suggestions={cfg.suggestions} onSelect={handleSend} color={cfg.color} />
            </motion.div>
          )}
        </AnimatePresence>

        {/* Message list */}
        {messages.map((msg) => (
          <motion.div
            key={msg.id}
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
            className={cn("flex gap-2.5 group", msg.role === "user" && "flex-row-reverse")}
          >
            {/* Avatar — only assistant */}
            {msg.role === "assistant" && (
              <div className={cn(
                "w-7 h-7 rounded-xl flex items-center justify-center flex-shrink-0 mt-0.5 text-[11px] font-bold bg-gradient-to-br flex-shrink-0",
                cfg.gradient,
              )} style={{ color: cfg.color }}>
                {cfg.icon}
              </div>
            )}

            {/* Bubble */}
            <div className={cn("max-w-[85%]", msg.role === "user" && "items-end flex flex-col")}>

              {/* Attached pill */}
              {msg.role === "user" && msg.attachedName && (
                <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl bg-white/[0.07] border border-white/10 text-[10px] text-white/40 self-end mb-1.5">
                  <FileText size={9} className="flex-shrink-0" />
                  <span className="truncate max-w-[160px]">{msg.attachedName}</span>
                </div>
              )}

              {/* Message body */}
              <div className={cn(
                "px-4 py-3 rounded-2xl text-[13.5px] leading-[1.7]",
                msg.role === "user"
                  ? "rounded-tr-sm text-white/90 bg-white/[0.09] border border-white/[0.08]"
                  : "rounded-tl-sm text-white/80",
              )}>
                {msg.role === "user" ? (
                  <p>{msg.content}</p>
                ) : msg.isStreaming && streamingId === msg.id ? (
                  <StreamingMessage
                    content={msg.content}
                    isStreaming={true}
                    onStreamDone={() => {
                      setStreamingId(null);
                      setMessages((p) => p.map((m) => m.id === msg.id
                        ? { ...m, isStreaming: false, displayContent: m.content }
                        : m));
                    }}
                  />
                ) : (
                  <RichMessage content={msg.content} />
                )}
              </div>

              {/* Hover actions */}
              <div className={cn(
                "flex items-center gap-1.5 mt-1 px-1 opacity-0 group-hover:opacity-100 transition-opacity duration-150",
                msg.role === "user" && "flex-row-reverse",
              )}>
                <span className="text-white/15 text-[9px] tabular-nums">
                  {msg.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                </span>
                {msg.role === "assistant" && msg.provider && (
                  <span className="text-[9px] font-semibold text-white/15 border border-white/10 px-1.5 py-0.5 rounded-md uppercase tracking-wide">{msg.provider}</span>
                )}
                <button
                  onClick={() => copyMsg(msg.content, msg.id)}
                  className="flex items-center gap-1 text-[10px] text-white/20 hover:text-white/60 transition-colors px-1.5 py-0.5 rounded-lg hover:bg-white/[0.05]"
                >
                  {copiedId === msg.id ? <Check size={9} className="text-green-400" /> : <Copy size={9} />}
                  {copiedId === msg.id ? "Copied" : "Copy"}
                </button>
              </div>
            </div>
          </motion.div>
        ))}

        {/* Typing indicator */}
        <AnimatePresence>
          {isLoading && (
            <motion.div
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              className="flex gap-2.5"
            >
              <div className={cn(
                "w-7 h-7 rounded-xl flex items-center justify-center flex-shrink-0 text-[11px] font-bold bg-gradient-to-br",
                cfg.gradient,
              )} style={{ color: cfg.color }}>
                {cfg.icon}
              </div>
              <div className="px-4 py-3 rounded-2xl rounded-tl-sm bg-white/[0.04] border border-white/[0.06]">
                <TypingDots color={cfg.color} />
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        <div ref={messagesEndRef} className="h-1" />
      </div>

      {/* Scroll-to-bottom button */}
      <AnimatePresence>
        {showScrollBtn && (
          <motion.button
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.8 }}
            onClick={() => scrollToBottom()}
            className="absolute bottom-28 right-4 w-8 h-8 rounded-full bg-white/10 border border-white/15 flex items-center justify-center text-white/60 hover:text-white hover:bg-white/20 transition-all shadow-lg"
            style={{ zIndex: 10 }}
          >
            <ChevronDown size={15} />
          </motion.button>
        )}
      </AnimatePresence>

      {/* ── Input area ───────────────────────────────────────────────────── */}
      <div className="flex-shrink-0 px-4 pb-5 pt-2 border-t border-white/[0.06]">

        {/* Attachment chip */}
        <AnimatePresence>
          {hasAttachment && (
            <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} exit={{ opacity: 0, height: 0 }}
              className="flex items-center gap-2 mb-2 overflow-hidden">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-white/[0.06] border border-white/10 text-xs text-white/50 flex-1 min-w-0">
                {attachedFile ? <FileText size={11} className="flex-shrink-0 text-indigo-400" /> : <Globe size={11} className="flex-shrink-0 text-sky-400" />}
                <span className="truncate">{attachedFile ? attachedFile.name : attachedLink}</span>
                {isUploading && <span className="ml-auto text-[10px] text-white/30 animate-pulse flex-shrink-0">Uploading…</span>}
                {!isUploading && attachedFileURL && <span className="ml-auto text-[10px] text-green-400/80 flex-shrink-0">✓</span>}
              </div>
              <button onClick={clearAttachment} className="w-7 h-7 flex items-center justify-center text-white/25 hover:text-white/60 hover:bg-white/[0.06] rounded-lg transition-all">
                <X size={12} />
              </button>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Upload error */}
        {uploadError && <p className="text-[11px] text-red-400/80 mb-2">{uploadError}</p>}

        {/* Voice banners */}
        <AnimatePresence>
          {micRecording && (
            <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} exit={{ opacity: 0, height: 0 }}
              className="flex items-center gap-2 mb-2 px-3 py-2 rounded-xl bg-red-500/10 border border-red-500/20 overflow-hidden">
              <span className="w-2 h-2 rounded-full bg-red-400 animate-pulse flex-shrink-0" />
              <span className="text-[11px] text-red-300/90 font-medium">Recording — tap mic to stop</span>
            </motion.div>
          )}
          {micProcessing && (
            <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} exit={{ opacity: 0, height: 0 }}
              className="flex items-center gap-2 mb-2 px-3 py-2 rounded-xl bg-white/[0.05] border border-white/10 overflow-hidden">
              <Loader2 size={11} className="animate-spin flex-shrink-0" style={{ color: cfg.color }} />
              <span className="text-[11px] text-white/50">{interimText || "Transcribing…"}</span>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Link input */}
        <AnimatePresence>
          {showLinkInput && (
            <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} exit={{ opacity: 0, height: 0 }}
              className="flex items-center gap-2 mb-2 overflow-hidden">
              <input
                ref={linkInputRef}
                type="url"
                placeholder="Paste a URL or Google Drive link…"
                value={linkInput}
                onChange={(e) => setLinkInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") confirmLink();
                  if (e.key === "Escape") { setShowLinkInput(false); setLinkInput(""); }
                }}
                className="flex-1 bg-white/[0.05] border border-white/10 rounded-xl py-2 px-3 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/25 transition-all"
              />
              <button onClick={confirmLink} disabled={!linkInput.trim()}
                className="px-3 py-2 rounded-xl text-xs font-semibold bg-white/[0.08] hover:bg-white/[0.14] text-white/70 transition-all disabled:opacity-30">
                Attach
              </button>
              <button onClick={() => { setShowLinkInput(false); setLinkInput(""); }}
                className="w-8 h-8 flex items-center justify-center text-white/25 hover:text-white/60 transition-colors">
                <X size={13} />
              </button>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Main input row */}
        <div className="flex items-end gap-2">
          {/* Toolbar — left side */}
          <div className="flex flex-col gap-1.5 mb-0.5 flex-shrink-0">
            {/* File */}
            <input ref={fileInputRef} type="file" accept=".pdf,.txt,.md,.csv,.doc,.docx" onChange={handleFileSelect} className="hidden" />
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={isBusy || remaining === 0}
              className={cn(
                "w-8 h-8 rounded-xl flex items-center justify-center transition-all",
                hasAttachment && attachedFile
                  ? "bg-indigo-500/20 text-indigo-300 border border-indigo-500/30"
                  : "text-white/25 hover:text-white/60 hover:bg-white/[0.07]",
                (isBusy || remaining === 0) && "opacity-30 cursor-not-allowed",
              )}
            >
              <Paperclip size={14} />
            </button>
            {/* Link */}
            <button
              onClick={() => setShowLinkInput((v) => !v)}
              disabled={isBusy || remaining === 0}
              className={cn(
                "w-8 h-8 rounded-xl flex items-center justify-center transition-all",
                showLinkInput || (hasAttachment && attachedLink)
                  ? "bg-sky-500/20 text-sky-300 border border-sky-500/30"
                  : "text-white/25 hover:text-white/60 hover:bg-white/[0.07]",
                (isBusy || remaining === 0) && "opacity-30 cursor-not-allowed",
              )}
            >
              <Link2 size={14} />
            </button>
          </div>

          {/* Textarea */}
          <div className="flex-1 relative">
            <textarea
              ref={textareaRef}
              rows={1}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleSend(); }
              }}
              disabled={remaining === 0 || isMicBusy}
              placeholder={
                micRecording ? "Listening…" :
                micProcessing ? "Transcribing…" :
                hasAttachment ? "Ask about this file or link…" :
                cfg.placeholder
              }
              className={cn(
                "w-full bg-white/[0.06] border border-white/[0.09] rounded-2xl py-3 px-4 text-[13.5px] text-white placeholder:text-white/20 focus:outline-none focus:border-white/20 focus:bg-white/[0.08] transition-all resize-none overflow-hidden disabled:opacity-30",
                micRecording && "border-red-500/30 bg-red-500/[0.04]",
              )}
            />
          </div>

          {/* Right-side buttons */}
          <div className="flex flex-col gap-1.5 mb-0.5 flex-shrink-0">
            {/* Mic */}
            <button
              onClick={handleMicClick}
              disabled={micProcessing || isLoading || remaining === 0}
              className={cn(
                "w-8 h-8 rounded-xl flex items-center justify-center transition-all",
                micRecording
                  ? "bg-red-500/20 text-red-400 border border-red-500/30 animate-pulse"
                  : micProcessing
                    ? "bg-white/[0.05] text-white/30 cursor-wait"
                    : "text-white/25 hover:text-white/60 hover:bg-white/[0.07]",
                (micProcessing || isLoading || remaining === 0) && "opacity-30 cursor-not-allowed",
              )}
            >
              {micProcessing ? <Loader2 size={14} className="animate-spin" /> : micRecording ? <MicOff size={14} /> : <Mic size={14} />}
            </button>

            {/* Send */}
            <button
              onClick={() => handleSend()}
              disabled={!canSend}
              className={cn(
                "w-8 h-8 rounded-xl flex items-center justify-center transition-all",
                canSend
                  ? "text-white shadow-sm"
                  : "text-white/15 cursor-not-allowed",
              )}
              style={canSend ? { backgroundColor: cfg.color + "30", border: `1px solid ${cfg.color}50` } : {}}
            >
              {isLoading
                ? <Loader2 size={14} className="animate-spin" style={{ color: cfg.color }} />
                : <Send size={13} style={{ color: canSend ? cfg.color : undefined }} />
              }
            </button>
          </div>
        </div>

        {/* Footer hint — only shown after hitting the limit */}
        {remaining === 0 && (
          <p className="text-center text-[10px] text-red-400/60 mt-2">
            Daily limit reached — recharges reset tomorrow
          </p>
        )}
      </div>
    </div>
  );
}
