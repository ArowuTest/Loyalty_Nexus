"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import useSWR from "swr";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import { useStore } from "@/store/useStore";
import toast, { Toaster } from "react-hot-toast";
import {
  Send, Bot, User, Loader2, Wand2, Image as ImageIcon, BookOpen,
  Mic, FileText, Music, Globe, ChevronRight, Sparkles,
  AlertTriangle, CheckCircle2, Clock, ExternalLink, RefreshCw,
  Brain, Video, X, Info, Play, LayoutGrid, MessageSquare, History,
  Code2, Copy, Check, Download, RotateCcw,
} from "lucide-react";
import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────────
interface Tool {
  id: string;
  slug: string;
  name: string;
  description: string;
  category: string;
  point_cost: number;
  is_active: boolean;
  provider?: string;
}
interface Message {
  role: "user" | "assistant";
  content: string;
  provider?: string;
  ts?: number;
}
interface Generation {
  id: string;
  tool_name: string;
  tool_slug: string;
  status: "pending" | "processing" | "completed" | "failed";
  output_url?: string;
  output_text?: string;
  prompt?: string;
  created_at: string;
  point_cost?: number;
}

// ─── Tool Meta ─────────────────────────────────────────────────────────────────
const TOOL_META: Record<string, { time: string; output: string; tip: string }> = {
  "ai-chat":            { time: "instant",  output: "Conversational reply",          tip: "Ask follow-ups to go deeper" },
  "web-search-ai":      { time: "~5 sec",   output: "Live internet answer",          tip: "Include 'today' or a date for current info" },
  "ai-photo":           { time: "~8 sec",   output: "1024×1024 image",               tip: "Add style words: 'photorealistic', 'vibrant', 'cinematic'" },
  "ai-photo-pro":       { time: "~12 sec",  output: "Premium 1024×1024 image",       tip: "Describe lighting, mood, and camera angle" },
  "ai-photo-max":       { time: "~20 sec",  output: "Max quality image",             tip: "Be very detailed — every word affects the result" },
  "ai-photo-dream":     { time: "~10 sec",  output: "Creative stylized image",       tip: "Try 'Afrofuturist', 'anime', 'oil painting' styles" },
  "photo-editor":       { time: "~15 sec",  output: "Edited photo",                  tip: "Be specific: 'remove background and replace with beach sunset'" },
  "image-analyser":     { time: "~4 sec",   output: "Detailed description",          tip: "Works with any public image URL" },
  "ask-my-photo":       { time: "~5 sec",   output: "AI answer about image",         tip: "Ask 'What is the brand logo in this image?'" },
  "bg-remover":         { time: "~5 sec",   output: "Transparent PNG",               tip: "Works best with clear subject vs background" },
  "animate-photo":      { time: "~45 sec",  output: "5-second MP4 video",            tip: "Use portraits or scenic photos for best motion" },
  "video-cinematic":    { time: "~90 sec",  output: "Cinematic 5s video",            tip: "Describe motion: 'slow zoom in', 'camera pan left'" },
  "video-premium":      { time: "~2 min",   output: "HD video clip",                 tip: "More detail in prompt = better camera movement" },
  "video-veo":          { time: "~3 min",   output: "Google Veo video",              tip: "Describe the scene like a film director would" },
  "narrate":            { time: "~4 sec",   output: "MP3 audio file",                tip: "Keep text under 500 words for best quality" },
  "narrate-pro":        { time: "~5 sec",   output: "MP3 with premium voice",        tip: "Try 'coral' for warm tone, 'onyx' for deep voice" },
  "transcribe":         { time: "~6 sec",   output: "Text transcript",               tip: "Paste a direct link to an MP3 or WAV file" },
  "transcribe-african": { time: "~8 sec",   output: "African language transcript",   tip: "Select language BEFORE submitting for accuracy" },
  "translate":          { time: "~3 sec",   output: "Translated text",               tip: "Format: type your text, select target language" },
  "bg-music":           { time: "~30 sec",  output: "15-second music clip",          tip: "Describe mood: 'calm', 'energetic', 'corporate'" },
  "jingle":             { time: "~25 sec",  output: "AI music jingle",               tip: "Add brand name and target emotion in prompt" },
  "song-creator":       { time: "~2 min",   output: "Full AI song with vocals",      tip: "Afrobeats, Gospel, Amapiano — be specific about genre" },
  "instrumental":       { time: "~2 min",   output: "Instrumental music track",      tip: "Describe instruments: 'piano, strings, light percussion'" },
  "code-helper":        { time: "~5 sec",   output: "Code + explanation",            tip: "Mention the programming language in your prompt" },
  "study-guide":        { time: "~8 sec",   output: "Structured study guide",        tip: "Add 'for WAEC' or 'for university level' for focus" },
  "quiz":               { time: "~6 sec",   output: "10 multiple-choice questions",  tip: "Specify difficulty: 'easy', 'intermediate', 'expert'" },
  "mindmap":            { time: "~5 sec",   output: "Interactive mind map",          tip: "One topic at a time gives the best results" },
  "research-brief":     { time: "~10 sec",  output: "Structured research report",    tip: "Be specific about industry or location context" },
  "bizplan":            { time: "~12 sec",  output: "Nigerian market business plan", tip: "Include target city and startup budget for relevance" },
  "slide-deck":         { time: "~10 sec",  output: "10-slide presentation outline", tip: "Add audience type: 'investors', 'students', 'clients'" },
  "infographic":        { time: "~8 sec",   output: "Data layout structure",         tip: "Include statistics or data points in your prompt" },
  "podcast":            { time: "~90 sec",  output: "2-host AI podcast audio",       tip: "Give a clear topic — the AI writes the full script" },
};

// ─── Output type label helper ─────────────────────────────────────────────────
function getOutputLabel(slug: string): string {
  if (VIDEO_SLUGS.has(slug))  return "🎬 Video MP4";
  if (AUDIO_SLUGS.has(slug))  return "🎵 Audio MP3";
  if (IMAGE_SLUGS.has(slug))  return "🖼️ Image file";
  if (CODE_SLUGS.has(slug))   return "💻 Code output";
  return "📄 Text output";
}

// ─── Category config ──────────────────────────────────────────────────────────
const CAT = {
  "Knowledge & Research": {
    icon: <BookOpen size={15} />, color: "from-blue-500/20 to-blue-600/10",
    badge: "bg-blue-500/20 text-blue-300 border border-blue-500/30",
    dot: "bg-blue-400",
  },
  "Image & Visual": {
    icon: <ImageIcon size={15} />, color: "from-pink-500/20 to-rose-600/10",
    badge: "bg-pink-500/20 text-pink-300 border border-pink-500/30",
    dot: "bg-pink-400",
  },
  "Audio & Voice": {
    icon: <Mic size={15} />, color: "from-green-500/20 to-emerald-600/10",
    badge: "bg-green-500/20 text-green-300 border border-green-500/30",
    dot: "bg-green-400",
  },
  "Document & Business": {
    icon: <FileText size={15} />, color: "from-orange-500/20 to-amber-600/10",
    badge: "bg-orange-500/20 text-orange-300 border border-orange-500/30",
    dot: "bg-orange-400",
  },
  "Music & Entertainment": {
    icon: <Music size={15} />, color: "from-purple-500/20 to-violet-600/10",
    badge: "bg-purple-500/20 text-purple-300 border border-purple-500/30",
    dot: "bg-purple-400",
  },
  "Language & Translation": {
    icon: <Globe size={15} />, color: "from-cyan-500/20 to-sky-600/10",
    badge: "bg-cyan-500/20 text-cyan-300 border border-cyan-500/30",
    dot: "bg-cyan-400",
  },
  "Video & Animation": {
    icon: <Video size={15} />, color: "from-red-500/20 to-rose-600/10",
    badge: "bg-red-500/20 text-red-300 border border-red-500/30",
    dot: "bg-red-400",
  },
  "Vision": {
    icon: <Brain size={15} />, color: "from-violet-500/20 to-violet-600/10",
    badge: "bg-violet-500/20 text-violet-300 border border-violet-500/30",
    dot: "bg-violet-400",
  },
  "Chat": {
    icon: <MessageSquare size={15} />, color: "from-cyan-500/20 to-teal-600/10",
    badge: "bg-cyan-500/20 text-cyan-300 border border-cyan-500/30",
    dot: "bg-cyan-400",
  },
  "Build": {
    icon: <Code2 size={15} />, color: "from-lime-500/20 to-green-600/10",
    badge: "bg-lime-500/20 text-lime-300 border border-lime-500/30",
    dot: "bg-lime-400",
  },
  "Create": {
    icon: <Sparkles size={15} />, color: "from-amber-500/20 to-orange-600/10",
    badge: "bg-amber-500/20 text-amber-300 border border-amber-500/30",
    dot: "bg-amber-400",
  },
} as const;

// ─── New tool slugs (for badge decoration) ────────────────────────────────────
const NEW_TOOL_SLUGS = new Set([
  "web-search-ai","image-analyser","ask-my-photo","code-helper",
  "narrate-pro","transcribe-african",
  "ai-photo-pro","ai-photo-max","ai-photo-dream","photo-editor",
  "song-creator","instrumental","video-cinematic","video-veo",
]);

// ─── Dual / special input sets ────────────────────────────────────────────────
const DUAL_INPUT_TOOLS = new Set(["ask-my-photo","photo-editor","video-cinematic"]);
const URL_INPUT_TOOLS  = new Set(["image-analyser","transcribe-african"]);
const VOICE_TOOLS      = new Set(["narrate-pro"]);
const LANG_TOOLS       = new Set(["transcribe-african"]);

const VOICES = ["alloy","echo","fable","onyx","nova","shimmer","coral","verse","ballad","ash","sage","amuch","dan"] as const;
const LANGUAGES = [
  { code: "en", label: "English 🇬🇧" },
  { code: "yo", label: "Yoruba 🇳🇬" },
  { code: "ha", label: "Hausa 🇳🇬" },
  { code: "ig", label: "Igbo 🇳🇬" },
  { code: "fr", label: "French 🇫🇷" },
] as const;

const GENRE_CHIPS = ["Afrobeats","Gospel","Hip Hop","Amapiano","Jazz","Classical"] as const;

// ─── Output type helpers ──────────────────────────────────────────────────────
const IMAGE_SLUGS  = new Set(["ai-photo","ai-photo-pro","ai-photo-max","ai-photo-dream","photo-editor","animate-photo","infographic"]);
const AUDIO_SLUGS  = new Set(["narrate","narrate-pro","bg-music","jingle","song-creator","instrumental","transcribe","transcribe-african"]);
const VIDEO_SLUGS  = new Set(["animate-photo","video-premium","video-cinematic","video-veo"]);
const CODE_SLUGS   = new Set(["code-helper"]);
const VISION_SLUGS = new Set(["image-analyser","ask-my-photo"]);
const WEB_SLUGS    = new Set(["web-search-ai"]);
const JSON_SLUGS   = new Set(["quiz","mindmap","slide-deck"]);

// ─── Placeholders ─────────────────────────────────────────────────────────────
const PLACEHOLDERS: Record<string, string> = {
  "ai-photo":           "A vibrant market scene in Lagos at golden hour, photorealistic…",
  "bg-music":           "Uplifting Afrobeats background music, 15 seconds, no vocals…",
  "narrate":            "Paste your text here and I'll convert it to natural speech…",
  "translate":          "Enter the text you want translated…",
  "jingle":             "30-second energetic jingle for a fintech brand called Nexus…",
  "slide-deck":         "Business plan presentation for a mobile loyalty platform…",
  "transcribe":         "Upload voice note link or describe what to transcribe…",
  "business-plan":      "Online grocery delivery startup targeting Abuja residents…",
  "summary":            "Paste the long article or document you want summarized…",
  "research-brief":     "Opportunities in Nigeria's mobile payments sector 2026…",
  "mindmap":            "How blockchain can be applied in African agriculture…",
  "infographic":        "Steps to open a small business in Nigeria — visual format…",
  "animate-photo":      "URL of your image to animate with subtle motion…",
  "bg-remover":         "URL of product image to remove background from…",
  "web-search-ai":      "Ask anything — e.g. 'What is the current price of Bitcoin?' or 'Latest Nigeria news today'…",
  "image-analyser":     "Paste an image URL to analyse…",
  "ask-my-photo":       "Paste your image URL here…",
  "code-helper":        "Describe what code you need — e.g. 'Write a Python function to sort a list of dictionaries by key'…",
  "narrate-pro":        "Type or paste the text you want narrated…",
  "transcribe-african": "Paste the URL of an audio file to transcribe…",
  "ai-photo-pro":       "Describe your photorealistic image — e.g. 'Professional headshot of a Nigerian business executive'…",
  "ai-photo-max":       "Describe your image in detail for maximum quality output…",
  "ai-photo-dream":     "Describe a creative or stylized image — e.g. 'Afrofuturist cityscape at sunset'…",
  "photo-editor":       "Paste your image URL here…",
  "song-creator":       "Describe your song — e.g. 'Upbeat Afrobeats love song, Lagos vibes, female vocals, 120 BPM'…",
  "instrumental":       "Describe the instrumental — e.g. 'Calm piano background music for studying, 60 seconds'…",
  "video-cinematic":    "Paste your image URL here…",
  "video-veo":          "Describe your video — e.g. 'A drone shot over Lagos Island at sunrise, cinematic'…",
};

const SECOND_PLACEHOLDERS: Record<string, string> = {
  "ask-my-photo":    "What do you want to know about this image?",
  "photo-editor":    "Describe the edit — e.g. 'Remove the background', 'Add golden hour lighting'",
  "video-cinematic": "Describe the motion — e.g. 'Slow zoom in with lens flare, cinematic movement'",
};

// ─── Provider labels hidden from users — Nexus AI is the brand ───────────────

// ─── Fetchers ─────────────────────────────────────────────────────────────────
const fetchTools   = () => api.getStudioTools()  as Promise<{ tools: Tool[] }>;
const fetchGallery = () => api.getGallery()       as Promise<{ items: Generation[] }>;

// ─── Utility ──────────────────────────────────────────────────────────────────
function catCfg(category: string) {
  return CAT[category as keyof typeof CAT] ?? {
    icon: <Wand2 size={15} />, color: "from-gray-500/20 to-gray-600/10",
    badge: "bg-white/10 text-white/60 border border-white/10", dot: "bg-gray-400",
  };
}

// ─── Copy-code button ─────────────────────────────────────────────────────────
function CopyButton({ text, label = "Copy Code" }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Copy failed");
    }
  };
  return (
    <button
      onClick={handleCopy}
      className="flex items-center gap-1 text-[10px] font-medium px-2.5 py-1 rounded-lg
                 bg-white/10 hover:bg-white/20 text-white/60 hover:text-white transition-all"
    >
      {copied ? <Check size={11} className="text-green-400" /> : <Copy size={11} />}
      {copied ? "Copied!" : label}
    </button>
  );
}

// ─── Intro / How It Works banner ─────────────────────────────────────────────
function HowItWorksBanner({ onDismiss }: { onDismiss: () => void }) {
  const steps = [
    { icon: "🔍", label: "Choose a tool" },
    { icon: "✏️", label: "Describe what you want" },
    { icon: "⚡", label: "Nexus AI creates it" },
    { icon: "⬇", label: "Download or copy" },
  ];
  return (
    <motion.div
      initial={{ opacity: 0, y: -8 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -8 }}
      className="nexus-card p-4 border border-nexus-500/20 bg-gradient-to-r from-nexus-600/10 to-purple-600/10"
    >
      <div className="flex items-start justify-between gap-2 mb-3">
        <p className="text-white/80 text-xs font-semibold uppercase tracking-wider">How It Works</p>
        <button onClick={onDismiss} className="text-white/30 hover:text-white/70 transition-colors">
          <X size={14} />
        </button>
      </div>
      <div className="grid grid-cols-4 gap-2">
        {steps.map((s, i) => (
          <div key={i} className="flex flex-col items-center gap-1 text-center">
            <span className="text-xl">{s.icon}</span>
            <p className="text-white/50 text-[10px] leading-tight">{s.label}</p>
          </div>
        ))}
      </div>
    </motion.div>
  );
}

// ─── Confirmation modal ───────────────────────────────────────────────────────
function ConfirmModal({
  tool, prompt, onConfirm, onCancel, busy, userPoints,
}: {
  tool: Tool; prompt: string; onConfirm: () => void;
  onCancel: () => void; busy: boolean; userPoints: number;
}) {
  const cfg = catCfg(tool.category);
  const isFree    = tool.point_cost === 0;
  const canAfford = userPoints >= tool.point_cost;

  return (
    <motion.div
      initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
      className="fixed inset-0 bg-black/70 backdrop-blur-md z-50 flex items-end md:items-center justify-center p-4"
      onClick={onCancel}
    >
      <motion.div
        initial={{ y: 60, scale: 0.96 }} animate={{ y: 0, scale: 1 }}
        exit={{ y: 60, scale: 0.96 }} transition={{ type: "spring", damping: 25 }}
        className="w-full max-w-md"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="nexus-card overflow-hidden">
          <div className={cn("h-1.5 w-full bg-gradient-to-r", cfg.color.replace("from-","from-").replace("/20","/60").replace("/10","/40"))} />
          <div className="p-6 space-y-5">
            <div className="flex items-start gap-3">
              <div className={cn("p-2.5 rounded-xl bg-gradient-to-br", cfg.color)}>{cfg.icon}</div>
              <div>
                <h3 className="text-white font-bold text-lg leading-tight">{tool.name}</h3>
                <p className="text-white/50 text-sm mt-0.5">{tool.description}</p>
              </div>
            </div>

            <div className="bg-white/5 border border-white/10 rounded-xl p-3">
              <p className="text-white/40 text-xs uppercase tracking-wider mb-1 font-medium">Your prompt</p>
              <p className="text-white/80 text-sm line-clamp-3">{prompt}</p>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between py-2 border-b border-white/10">
                <span className="text-white/60 text-sm">Tool cost</span>
                <span className={cn("font-bold text-sm", isFree ? "text-green-400" : "text-white")}>
                  {isFree ? "FREE ✓" : `-${tool.point_cost} Pulse Points`}
                </span>
              </div>
              <div className="flex items-center justify-between py-1">
                <span className="text-white/60 text-sm">Your balance after</span>
                <span className={cn("font-semibold text-sm", canAfford ? "text-nexus-300" : "text-red-400")}>
                  {canAfford
                    ? `${(userPoints - tool.point_cost).toLocaleString()} pts remaining`
                    : "⚠ Insufficient points"}
                </span>
              </div>
            </div>

            {!canAfford && !isFree && (
              <div className="flex items-center gap-2.5 bg-red-500/10 border border-red-500/20 rounded-xl p-3">
                <AlertTriangle size={16} className="text-red-400 flex-shrink-0" />
                <p className="text-red-300 text-sm">
                  You need <strong>{(tool.point_cost - userPoints).toLocaleString()} more points</strong>. Recharge to continue.
                </p>
              </div>
            )}

            {canAfford && !isFree && (
              <div className="flex items-start gap-2.5 bg-nexus-600/10 border border-nexus-500/20 rounded-xl p-3">
                <Info size={15} className="text-nexus-400 flex-shrink-0 mt-0.5" />
                <p className="text-nexus-300 text-xs leading-relaxed">
                  Points are deducted when generation starts. If the AI fails, points are automatically refunded within seconds.
                </p>
              </div>
            )}

            <div className="flex gap-2 pt-1">
              <button onClick={onCancel} className="nexus-btn-outline flex-1 text-sm py-3">Cancel</button>
              <button
                onClick={onConfirm}
                disabled={busy || (!canAfford && !isFree)}
                className={cn(
                  "flex-1 py-3 rounded-xl text-sm font-semibold flex items-center justify-center gap-2 transition-all",
                  canAfford || isFree
                    ? "bg-gradient-to-r from-nexus-600 to-purple-600 text-white hover:opacity-90 active:scale-[0.98]"
                    : "bg-white/5 text-white/30 cursor-not-allowed"
                )}
              >
                {busy ? (
                  <><Loader2 size={16} className="animate-spin" /> Starting…</>
                ) : (
                  <><Sparkles size={16} /> {isFree ? "Generate (Free)" : `Use ${tool.point_cost} pts`}</>
                )}
              </button>
            </div>
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
}

// ─── Chat bubble ──────────────────────────────────────────────────────────────
function ChatBubble({ msg }: { msg: Message }) {
  const isUser = msg.role === "user";
  return (
    <div className={cn("flex gap-2.5", isUser && "flex-row-reverse")}>
      <div className={cn(
        "w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5",
        isUser ? "bg-gradient-to-br from-purple-600/40 to-nexus-600/40"
               : "bg-gradient-to-br from-nexus-600/30 to-blue-600/30"
      )}>
        {isUser ? <User size={14} className="text-purple-300" /> : <Brain size={14} className="text-nexus-300" />}
      </div>
      <div className={cn("max-w-[78%] space-y-1", isUser && "items-end flex flex-col")}>
        <div className={cn(
          "px-4 py-2.5 text-sm leading-relaxed",
          isUser
            ? "bg-gradient-to-br from-nexus-600 to-purple-700 text-white rounded-2xl rounded-tr-sm"
            : "bg-[rgb(32_38_68)] text-white/90 rounded-2xl rounded-tl-sm border border-white/5"
        )}>
          {msg.content}
        </div>
      </div>
    </div>
  );
}

// ─── Tool Card ────────────────────────────────────────────────────────────────
function ToolCard({ tool, onClick }: { tool: Tool; onClick: () => void }) {
  const cfg       = catCfg(tool.category);
  const isFree    = tool.point_cost === 0;
  const isPremium = tool.point_cost >= 20;
  const isNew     = NEW_TOOL_SLUGS.has(tool.slug);
  const meta      = TOOL_META[tool.slug];

  return (
    <motion.button
      whileHover={{ scale: 1.015 }} whileTap={{ scale: 0.98 }}
      onClick={onClick}
      className="w-full nexus-card p-4 flex items-center gap-3.5 text-left group hover:border-white/20 transition-all"
    >
      <div className={cn("p-2.5 rounded-xl bg-gradient-to-br flex-shrink-0 transition-transform group-hover:scale-110", cfg.color)}>
        {cfg.icon}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5 flex-wrap">
          <p className="text-white font-semibold text-sm truncate">{tool.name}</p>
          {isNew && (
            <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-purple-500/25 text-purple-300 border border-purple-500/30 leading-none">
              NEW
            </span>
          )}
        </div>
        <p className="text-white/45 text-xs mt-0.5 line-clamp-1 leading-relaxed">{tool.description}</p>
        {meta && (
          <p className="text-white/30 text-[10px] mt-0.5 leading-none">{meta.output}</p>
        )}
      </div>
      <div className="flex flex-col items-end gap-1.5 flex-shrink-0">
        <span className={cn(
          "text-xs font-bold px-2 py-0.5 rounded-full",
          isFree
            ? "bg-green-500/20 text-green-300 border border-green-500/30"
            : isPremium
              ? "bg-amber-500/20 text-amber-300 border border-amber-500/30"
              : "bg-nexus-500/20 text-nexus-300 border border-nexus-500/30"
        )}>
          {isFree ? "Free" : isPremium ? `⭐ ${tool.point_cost} pts` : `${tool.point_cost} pts`}
        </span>
        {meta && (
          <span className="text-[9px] text-white/30 flex items-center gap-0.5 font-medium">
            <Clock size={9} />
            {meta.time}
          </span>
        )}
        <ChevronRight size={13} className="text-white/25 group-hover:text-white/60 transition-colors" />
      </div>
    </motion.button>
  );
}

// ─── Generation status card ───────────────────────────────────────────────────
function GenerationCard({ gen, onRegenerate }: { gen: Generation; onRegenerate?: (gen: Generation) => void }) {
  const isImage  = IMAGE_SLUGS.has(gen.tool_slug);
  const isAudio  = AUDIO_SLUGS.has(gen.tool_slug);
  const isVideo  = VIDEO_SLUGS.has(gen.tool_slug);
  const isCode   = CODE_SLUGS.has(gen.tool_slug);
  const isVision = VISION_SLUGS.has(gen.tool_slug);
  const isWeb    = WEB_SLUGS.has(gen.tool_slug);
  const isJson   = JSON_SLUGS.has(gen.tool_slug);
  const meta     = TOOL_META[gen.tool_slug];

  // ── Quiz card renderer ──
  function renderQuiz(text: string) {
    let parsed: { question?: string; options?: string[]; answer?: string }[] | null = null;
    try {
      const raw = JSON.parse(text);
      if (Array.isArray(raw)) parsed = raw;
    } catch { /* not valid JSON */ }
    if (!parsed) {
      return <p className="text-white/70 text-sm leading-relaxed whitespace-pre-wrap">{text}</p>;
    }
    return (
      <div className="space-y-3">
        {parsed.map((q, i) => (
          <div key={i} className="bg-white/5 border border-white/10 rounded-xl p-3 space-y-2">
            <p className="text-white/90 text-sm font-medium">{i + 1}. {q.question}</p>
            {Array.isArray(q.options) && (
              <ul className="space-y-1">
                {q.options.map((opt: string, oi: number) => (
                  <li key={oi} className={cn(
                    "text-xs px-3 py-1.5 rounded-lg border",
                    q.answer === opt || q.answer === String(oi)
                      ? "border-green-500/40 bg-green-500/10 text-green-300"
                      : "border-white/10 text-white/55"
                  )}>
                    {opt}
                  </li>
                ))}
              </ul>
            )}
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="nexus-card p-4 space-y-3">
      {/* Header row */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-white text-sm font-semibold truncate">{gen.tool_name}</span>
          {isAudio && !isVideo && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-green-500/15 text-green-300 border border-green-500/20">🎵 Audio</span>}
          {isVideo  && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/15 text-red-300 border border-red-500/20">🎬 Video</span>}
          {isCode   && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-lime-500/15 text-lime-300 border border-lime-500/20">💻 Code</span>}
          {isWeb    && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-cyan-500/15 text-cyan-300 border border-cyan-500/20">🔍 Live</span>}
        </div>
        <StatusPill status={gen.status} />
      </div>

      {/* ── Processing state — animated skeleton ── */}
      {gen.status === "processing" && (
        <div className="space-y-3">
          {/* Indeterminate progress bar */}
          <div className="h-1 w-full rounded-full bg-white/10 overflow-hidden">
            <div className="h-full w-1/3 rounded-full bg-gradient-to-r from-nexus-500 to-purple-500 animate-[progress_1.6s_ease-in-out_infinite]" />
          </div>
          {/* Skeleton lines */}
          <div className="space-y-2">
            <div className="h-3 rounded-lg bg-white/10 animate-pulse w-3/4" />
            <div className="h-3 rounded-lg bg-white/8 animate-pulse w-1/2" />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-nexus-400 text-xs">
              <Loader2 size={12} className="animate-spin" />
              <span>Creating your {gen.tool_name}…</span>
            </div>
            {meta && (
              <span className="text-white/30 text-[10px] flex items-center gap-1">
                <Clock size={9} /> ~{meta.time}
              </span>
            )}
          </div>
          {meta && (
            <div className="flex items-start gap-1.5 bg-amber-500/5 border border-amber-500/15 rounded-xl px-3 py-2">
              <span className="text-amber-400 text-xs">💡</span>
              <p className="text-amber-200/60 text-[11px] leading-relaxed">Did you know? {meta.tip}</p>
            </div>
          )}
        </div>
      )}

      {/* ── Completed: URL outputs ── */}
      {gen.status === "completed" && gen.output_url && (
        <div className="space-y-2 rounded-xl overflow-hidden">
          {isImage && !isVideo && (
            <div className="space-y-2">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={gen.output_url} alt={gen.tool_name} className="w-full rounded-xl object-cover" />
              {gen.prompt && (
                <p className="text-white/30 text-[10px] italic px-1 line-clamp-2">"{gen.prompt}"</p>
              )}
              <div className="flex gap-2">
                <a
                  href={gen.output_url}
                  download
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                >
                  <Download size={11} /> Download Image
                </a>
                {onRegenerate && (
                  <button
                    onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                  >
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
          {isAudio && !isVideo && (
            <div className="space-y-2">
              <audio controls className="w-full mt-1" src={gen.output_url}>
                Your browser does not support audio.
              </audio>
              <div className="flex gap-2">
                <a
                  href={gen.output_url}
                  download
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                >
                  <Download size={11} /> Download Audio
                </a>
                {onRegenerate && (
                  <button
                    onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                  >
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
          {isVideo && (
            <div className="space-y-2">
              <video controls className="w-full rounded-xl max-h-64" src={gen.output_url}>
                Your browser does not support video.
              </video>
              <div className="flex gap-2">
                <a
                  href={gen.output_url}
                  download
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                >
                  <Download size={11} /> Download Video
                </a>
                {onRegenerate && (
                  <button
                    onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                  >
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
          {!isImage && !isAudio && !isVideo && (
            <a href={gen.output_url} target="_blank" rel="noreferrer"
              className="flex items-center gap-2 text-nexus-400 text-sm hover:text-nexus-300">
              <ExternalLink size={14} /> View result
            </a>
          )}
        </div>
      )}

      {/* ── Completed: text outputs ── */}
      {gen.status === "completed" && gen.output_text && !gen.output_url && (
        <div className="space-y-2">
          {isWeb && (
            <div className="space-y-2">
              <div className="flex items-center gap-1.5 text-cyan-300 text-xs font-semibold">
                <Globe size={12} /> 🔍 Live Web Result
              </div>
              <p className="text-white/70 text-sm bg-white/5 rounded-xl p-3 leading-relaxed whitespace-pre-wrap">
                {gen.output_text}
              </p>
            </div>
          )}
          {isVision && (
            <p className="text-white/80 text-sm bg-violet-500/5 border border-violet-500/10 rounded-xl p-3 leading-loose whitespace-pre-wrap">
              {gen.output_text}
            </p>
          )}
          {isCode && (
            <div className="relative">
              <div className="flex items-center justify-between bg-gray-900/80 px-3 py-1.5 rounded-t-xl border border-white/10 border-b-0">
                <span className="text-xs text-white/40 font-mono">Code output</span>
                <CopyButton text={gen.output_text} />
              </div>
              <pre className="bg-gray-950 text-green-300 text-xs font-mono p-4 rounded-b-xl border border-white/10 overflow-x-auto whitespace-pre-wrap max-h-64 overflow-y-auto leading-relaxed">
                <code>{gen.output_text}</code>
              </pre>
            </div>
          )}
          {isJson && !isCode && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-white/50 text-xs font-medium uppercase tracking-wider">Result</span>
                <CopyButton text={gen.output_text} label="📋 Copy JSON" />
              </div>
              {gen.tool_slug === "quiz" ? renderQuiz(gen.output_text) : (
                <pre className="bg-gray-950 text-white/60 text-xs font-mono p-3 rounded-xl border border-white/10 overflow-x-auto whitespace-pre-wrap max-h-60 overflow-y-auto leading-relaxed">
                  {gen.output_text}
                </pre>
              )}
              {onRegenerate && (
                <button
                  onClick={() => onRegenerate(gen)}
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                >
                  <RotateCcw size={11} /> Regenerate
                </button>
              )}
            </div>
          )}
          {!isWeb && !isVision && !isCode && !isJson && (
            <div className="space-y-2">
              <p className="text-white/70 text-sm bg-white/5 rounded-xl p-3 leading-relaxed whitespace-pre-wrap">
                {gen.output_text}
              </p>
              <div className="flex gap-2">
                <CopyButton text={gen.output_text} label="📋 Copy Text" />
                {onRegenerate && (
                  <button
                    onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all"
                  >
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Footer badge row ── */}
      {gen.status === "completed" && (
        <div className="flex items-center justify-between pt-1 border-t border-white/5">
          <span className="text-white/25 text-[10px]">Generated by Nexus AI</span>
          {gen.point_cost !== undefined && (
            <span className="text-white/25 text-[10px]">
              {gen.point_cost === 0 ? "✓ Free" : `Used ${gen.point_cost} pts`}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function StatusPill({ status }: { status: Generation["status"] }) {
  const config = {
    pending:    { icon: <Clock size={11} />,        label: "Queued",     cls: "bg-yellow-500/15 text-yellow-300 border-yellow-500/30" },
    processing: { icon: <Loader2 size={11} className="animate-spin" />, label: "Generating", cls: "bg-blue-500/15 text-blue-300 border-blue-500/30" },
    completed:  { icon: <CheckCircle2 size={11} />, label: "Done",       cls: "bg-green-500/15 text-green-300 border-green-500/30" },
    failed:     { icon: <AlertTriangle size={11} />,label: "Failed",     cls: "bg-red-500/15 text-red-300 border-red-500/30" },
  }[status];
  return (
    <span className={cn("flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded-full border flex-shrink-0", config.cls)}>
      {config.icon}{config.label}
    </span>
  );
}

// ─── Tool prompt drawer ───────────────────────────────────────────────────────
function ToolDrawer({
  tool, onClose, userPoints,
}: {
  tool: Tool; onClose: () => void; userPoints: number;
}) {
  const [prompt,       setPrompt]       = useState("");
  const [secondInput,  setSecondInput]  = useState("");
  const [selectedVoice,setSelectedVoice]= useState<string>("nova");
  const [selectedLang, setSelectedLang] = useState<string>("en");
  const [showConfirm,  setShowConfirm]  = useState(false);
  const [generating,   setGenerating]   = useState(false);
  const cfg  = catCfg(tool.category);
  const slug = tool.slug;
  const meta = TOOL_META[slug];

  const isDual    = DUAL_INPUT_TOOLS.has(slug);
  const isURL     = URL_INPUT_TOOLS.has(slug);
  const isVoice   = VOICE_TOOLS.has(slug);
  const isLang    = LANG_TOOLS.has(slug);
  const isFree    = tool.point_cost === 0;
  const isPremium = tool.point_cost >= 20;
  const isNew     = NEW_TOOL_SLUGS.has(slug);
  const canAfford = userPoints >= tool.point_cost;

  // Build final prompt from composed fields
  function buildPrompt(): string {
    if (slug === "ask-my-photo")    return `${prompt.trim()}|||${secondInput.trim()}`;
    if (slug === "photo-editor")    return `${prompt.trim()}|||${secondInput.trim()}`;
    if (slug === "video-cinematic") return `${prompt.trim()}|||${secondInput.trim()}`;
    if (slug === "narrate-pro")     return `${selectedVoice}:${prompt.trim()}`;
    if (slug === "transcribe-african") return `${selectedLang}:${prompt.trim()}`;
    return prompt.trim();
  }

  const finalPrompt = buildPrompt();

  function isValid(): boolean {
    if (!prompt.trim() || prompt.trim().length < 3) return false;
    if (isDual && !secondInput.trim()) return false;
    return true;
  }

  const handleStart = async () => {
    if (!isValid()) return;
    setGenerating(true);
    try {
      await api.generateTool(tool.id, finalPrompt);
      toast.success("✅ Generation started! Check your gallery when ready.");
      onClose();
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Failed to start generation");
    } finally {
      setGenerating(false);
      setShowConfirm(false);
    }
  };

  return (
    <>
      <motion.div
        initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
        className="fixed inset-0 bg-black/60 backdrop-blur-sm z-40"
        onClick={onClose}
      />
      <motion.div
        initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
        transition={{ type: "spring", damping: 30, stiffness: 300 }}
        className="fixed bottom-0 left-0 right-0 z-40 max-h-[90vh] overflow-y-auto
                   md:relative md:inset-auto md:max-h-none"
      >
        <div className="nexus-card m-2 md:m-0 overflow-hidden">
          {/* Gradient top bar */}
          <div className={cn("h-1 w-full bg-gradient-to-r", cfg.color.replace("/20","/70").replace("/10","/50"))} />

          <div className="p-5 space-y-4">
            {/* Header */}
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <div className={cn("p-2.5 rounded-xl bg-gradient-to-br flex-shrink-0", cfg.color)}>{cfg.icon}</div>
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <h3 className="text-white font-bold text-base truncate">{tool.name}</h3>
                    {isNew    && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-purple-500/25 text-purple-300 border border-purple-500/30">NEW</span>}
                    {isFree   && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-green-500/20 text-green-300 border border-green-500/30">FREE</span>}
                    {isPremium && !isFree && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">PREMIUM</span>}
                    {slug === "ai-photo-pro"   && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-nexus-500/25 text-nexus-300 border border-nexus-500/30">⚡ Premium</span>}
                    {slug === "ai-photo-max"   && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-blue-500/20 text-blue-300 border border-blue-500/30">🌟 Max Quality</span>}
                    {slug === "ai-photo-dream" && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-pink-500/20 text-pink-300 border border-pink-500/30">🎨 Creative</span>}
                    {slug === "web-search-ai"  && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">🌐 Live internet</span>}
                    {slug === "video-veo"      && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-blue-500/20 text-blue-300 border border-blue-500/30">Google Veo</span>}
                  </div>
                  <p className="text-white/45 text-xs mt-0.5 leading-relaxed">{tool.description}</p>
                </div>
              </div>
              <button onClick={onClose} className="text-white/40 hover:text-white/80 transition-colors p-1 flex-shrink-0 ml-2">
                <X size={18} />
              </button>
            </div>

            {/* ── Tip box ── */}
            {meta?.tip && (
              <div className="flex items-start gap-2 border border-amber-500/25 bg-amber-500/8 rounded-xl px-3 py-2.5">
                <span className="text-amber-400 text-sm flex-shrink-0">💡</span>
                <p className="text-amber-200/75 text-xs leading-relaxed">
                  <span className="font-semibold">Tip: </span>{meta.tip}
                </p>
              </div>
            )}

            {/* ── Language pills (transcribe-african) ── */}
            {isLang && (
              <div>
                <label className="text-white/60 text-xs font-medium mb-2 block uppercase tracking-wider">
                  Language
                </label>
                <div className="flex flex-wrap gap-1.5">
                  {LANGUAGES.map((lang) => (
                    <button
                      key={lang.code}
                      onClick={() => setSelectedLang(lang.code)}
                      className={cn(
                        "text-xs px-3 py-1.5 rounded-full border font-medium transition-all",
                        selectedLang === lang.code
                          ? "bg-cyan-600 text-white border-cyan-500"
                          : "text-white/55 border-white/15 hover:border-white/30 hover:text-white/80"
                      )}
                    >
                      {lang.label}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* ── Primary input ── */}
            <div>
              <label className="text-white/60 text-xs font-medium mb-1.5 block uppercase tracking-wider">
                {isDual ? "Image URL" : isURL ? "Audio / File URL" : isVoice ? "Text to narrate" : "Describe what you want"}
              </label>
              {isURL || isDual ? (
                <input
                  type="url"
                  placeholder={PLACEHOLDERS[slug] ?? "Paste URL here…"}
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  className="nexus-input w-full text-sm"
                  autoFocus
                />
              ) : (
                <textarea
                  placeholder={PLACEHOLDERS[slug] ?? "Describe what you want to generate…"}
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  rows={4}
                  className="nexus-input resize-none w-full text-sm leading-relaxed"
                  autoFocus
                />
              )}
              {!isDual && !isURL && (
                <p className="text-white/25 text-xs mt-1">{prompt.length}/500 characters</p>
              )}
            </div>

            {/* ── Second input (dual-input tools) ── */}
            {isDual && (
              <div>
                <label className="text-white/60 text-xs font-medium mb-1.5 block uppercase tracking-wider">
                  {slug === "ask-my-photo" ? "Your question" : slug === "photo-editor" ? "Edit instruction" : "Motion prompt"}
                </label>
                <textarea
                  placeholder={SECOND_PLACEHOLDERS[slug] ?? "Enter your instruction…"}
                  value={secondInput}
                  onChange={(e) => setSecondInput(e.target.value)}
                  rows={3}
                  className="nexus-input resize-none w-full text-sm leading-relaxed"
                />
              </div>
            )}

            {/* ── Photo editor suggestions ── */}
            {slug === "photo-editor" && (
              <div>
                <p className="text-white/35 text-xs mb-1.5">Try these edits:</p>
                <div className="flex flex-wrap gap-1.5">
                  {["Remove the background","Add sunset lighting","Make it look like a painting","Add dramatic shadows","Convert to black & white"].map((s) => (
                    <button
                      key={s}
                      onClick={() => setSecondInput(s)}
                      className="text-xs px-2.5 py-1 rounded-full border border-white/15 text-white/50 hover:text-white/80 hover:border-white/30 transition-all"
                    >
                      {s}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* ── Voice selector (narrate-pro) ── */}
            {isVoice && (
              <div>
                <label className="text-white/60 text-xs font-medium mb-2 block uppercase tracking-wider">
                  Choose a voice
                </label>
                <div className="flex flex-wrap gap-1.5">
                  {VOICES.map((v) => (
                    <button
                      key={v}
                      onClick={() => setSelectedVoice(v)}
                      className={cn(
                        "text-xs px-3 py-1.5 rounded-full border font-medium transition-all capitalize",
                        selectedVoice === v
                          ? "bg-green-600 text-white border-green-500"
                          : "text-white/55 border-white/15 hover:border-white/30 hover:text-white/80"
                      )}
                    >
                      {v}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* ── Song / Instrumental genre chips ── */}
            {(slug === "song-creator" || slug === "instrumental") && (
              <div>
                <p className="text-white/35 text-xs mb-1.5">
                  {slug === "song-creator"
                    ? '💡 Tip: Describe genre, mood, tempo — e.g. "upbeat Afrobeats, female vocals, love theme"'
                    : '💡 Tip: Describe genre and mood — e.g. "calm piano background music for studying"'}
                </p>
                <div className="flex flex-wrap gap-1.5">
                  {GENRE_CHIPS.map((g) => (
                    <button
                      key={g}
                      onClick={() => setPrompt((p) => p ? `${p}, ${g}` : g)}
                      className="text-xs px-2.5 py-1 rounded-full border border-white/15 text-white/50 hover:text-white/80 hover:border-white/30 transition-all"
                    >
                      {g}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* ── Point cost confirmation row ── */}
            <div className={cn(
              "flex items-center gap-2 rounded-xl px-3 py-2 border text-xs",
              isFree
                ? "border-green-500/25 bg-green-500/8 text-green-300"
                : canAfford
                  ? "border-white/10 bg-white/5 text-white/55"
                  : "border-red-500/30 bg-red-500/8 text-red-400"
            )}>
              {isFree ? (
                <><CheckCircle2 size={13} className="flex-shrink-0" /> ✓ Free — no points needed</>
              ) : canAfford ? (
                <><Sparkles size={13} className="flex-shrink-0 text-nexus-400" />
                  This will use <span className="font-bold text-white mx-1">{tool.point_cost}</span> Pulse Points from your{" "}
                  <span className="font-bold text-nexus-300 ml-1">{userPoints.toLocaleString()} available</span>
                </>
              ) : (
                <><AlertTriangle size={13} className="flex-shrink-0" />
                  You need <span className="font-bold mx-1">{(tool.point_cost - userPoints).toLocaleString()}</span> more points — recharge to continue
                </>
              )}
            </div>

            {/* ── CTA button ── */}
            <button
              onClick={() => setShowConfirm(true)}
              disabled={!isValid()}
              className={cn(
                "w-full py-3.5 rounded-xl font-semibold flex items-center justify-center gap-2 text-sm transition-all",
                isValid()
                  ? "bg-gradient-to-r from-nexus-600 to-purple-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-nexus-900/30"
                  : "bg-white/5 text-white/20 cursor-not-allowed"
              )}
            >
              <Sparkles size={15} />
              {isFree ? "Generate for free" : `Review & use ${tool.point_cost} pts`}
            </button>

            {/* ── Generation time + output type row ── */}
            {meta && (
              <div className="flex items-center justify-between px-1">
                <span className="text-white/30 text-[11px] flex items-center gap-1">
                  <Clock size={10} /> Usually ready in {meta.time}
                </span>
                <span className="text-white/30 text-[11px]">{getOutputLabel(slug)}</span>
              </div>
            )}
          </div>
        </div>
      </motion.div>

      {/* Confirm modal sits above drawer */}
      <AnimatePresence>
        {showConfirm && (
          <ConfirmModal
            tool={tool}
            prompt={finalPrompt}
            userPoints={userPoints}
            onConfirm={handleStart}
            onCancel={() => setShowConfirm(false)}
            busy={generating}
          />
        )}
      </AnimatePresence>
    </>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export default function StudioPage() {
  const { data: toolsData, isLoading: toolsLoading } = useSWR("/studio/tools",   fetchTools);
  const { data: galleryData, mutate: mutateGallery }  = useSWR("/studio/gallery", fetchGallery, {
    refreshInterval: 8000,
  });
  const user       = useStore((s) => s.user);
  const wallet     = useStore((s) => s.wallet);
  const userPoints = wallet?.pulse_points ?? 0;

  const tools   = toolsData?.tools   ?? [];
  const gallery = galleryData?.items ?? [];
  const recentGens = gallery.slice(0, 6);

  const [activeTab,       setActiveTab]       = useState<"chat" | "tools" | "gallery">("chat");
  const [messages,        setMessages]        = useState<Message[]>([{
    role: "assistant",
    content: "Hey! 👋 I'm Nexus AI — your personal AI assistant. I can help with business ideas, explain anything, draft content, and more. What's on your mind?",
    ts: Date.now(),
  }]);
  const [input,           setInput]           = useState("");
  const [sending,         setSending]         = useState(false);
  const [sessionId]                           = useState(() => `sess_${Date.now()}_${Math.random().toString(36).slice(2)}`);
  const [selectedTool,    setSelectedTool]    = useState<Tool | null>(null);
  const [searchQuery,     setSearchQuery]     = useState("");
  const [activeCategory,  setActiveCategory]  = useState<string | null>(null);
  const [introDismissed,  setIntroDismissed]  = useState<boolean>(true); // start true, check localStorage
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef       = useRef<HTMLInputElement>(null);

  // Check localStorage for intro banner
  useEffect(() => {
    try {
      const dismissed = localStorage.getItem("nexus_studio_intro_dismissed");
      setIntroDismissed(dismissed === "true");
    } catch { /* localStorage may not be available */ }
  }, []);

  const handleDismissIntro = useCallback(() => {
    setIntroDismissed(true);
    try {
      localStorage.setItem("nexus_studio_intro_dismissed", "true");
    } catch { /* ignore */ }
  }, []);

  const categories    = [...new Set(tools.map((t) => t.category))];
  const filteredTools = tools.filter((t) => {
    const matchesSearch = !searchQuery ||
      t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCat = !activeCategory || t.category === activeCategory;
    return matchesSearch && matchesCat;
  });
  const groupedTools = categories.reduce((acc, cat) => {
    const catTools = filteredTools.filter((t) => t.category === cat);
    if (catTools.length > 0) acc[cat] = catTools;
    return acc;
  }, {} as Record<string, Tool[]>);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, sending]);

  const handleChat = useCallback(async () => {
    if (!input.trim() || sending) return;
    const msg = input.trim();
    setInput("");
    setMessages((m) => [...m, { role: "user", content: msg, ts: Date.now() }]);
    setSending(true);
    try {
      const resp = await api.sendChat(msg, sessionId) as { response: string; provider?: string };
      setMessages((m) => [...m, { role: "assistant", content: resp.response, provider: resp.provider, ts: Date.now() }]);
    } catch {
      setMessages((m) => [...m, {
        role: "assistant",
        content: "I'm having trouble connecting right now. Please try again in a moment. 🔄",
        ts: Date.now(),
      }]);
    } finally {
      setSending(false);
    }
  }, [input, sending, sessionId]);

  const handleClearChat = useCallback(() => {
    setMessages([{
      role: "assistant",
      content: "Hey! 👋 I'm Nexus AI — your personal AI assistant. I can help with business ideas, explain anything, draft content, and more. What's on your mind?",
      ts: Date.now(),
    }]);
  }, []);

  const pendingCount = gallery.filter((g) => ["pending","processing"].includes(g.status)).length;

  return (
    <AppShell>
      <Toaster
        position="top-center"
        toastOptions={{
          style: { background: "#1c2038", color: "#fff", border: "1px solid rgba(255,255,255,0.1)" },
          success: { iconTheme: { primary: "#22c55e", secondary: "#fff" } },
        }}
      />

      <div className="max-w-2xl mx-auto px-4 py-5 space-y-4 pb-24">

        {/* ── Header ── */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-2xl bg-gradient-to-br from-nexus-600 to-purple-600 flex items-center justify-center shadow-lg shadow-nexus-900/40">
              <Brain size={20} className="text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold font-display text-white leading-tight">Nexus AI Studio</h1>
              <p className="text-white/40 text-xs">{tools.length} AI-powered tools</p>
            </div>
          </div>
          <div className="flex flex-col items-end">
            <span className="text-nexus-300 font-bold text-sm">{userPoints.toLocaleString()}</span>
            <span className="text-white/35 text-[10px] uppercase tracking-wider">Pulse pts</span>
          </div>
        </div>

        {/* ── How It Works banner (first visit only) ── */}
        <AnimatePresence>
          {!introDismissed && (
            <HowItWorksBanner onDismiss={handleDismissIntro} />
          )}
        </AnimatePresence>

        {/* ── Tab bar ── */}
        <div className="nexus-card p-1 flex gap-1">
          {([
            { key: "chat",    label: "Chat",    icon: <MessageSquare size={14} />, badge: undefined as number | undefined },
            { key: "tools",   label: "Tools",   icon: <LayoutGrid size={14} />,   badge: tools.length as number | undefined },
            { key: "gallery", label: "Gallery", icon: <History size={14} />,      badge: (pendingCount || undefined) as number | undefined },
          ]).map(({ key, label, icon, badge }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key as "chat" | "tools" | "gallery")}
              className={cn(
                "flex-1 py-2.5 px-3 rounded-xl text-xs font-semibold transition-all flex items-center justify-center gap-1.5",
                activeTab === key
                  ? "bg-gradient-to-r from-nexus-600 to-purple-600 text-white shadow"
                  : "text-white/40 hover:text-white/70"
              )}
            >
              {icon}{label}
              {badge !== undefined && (
                <span className={cn(
                  "ml-0.5 text-[9px] font-bold px-1.5 py-0.5 rounded-full min-w-[18px] text-center",
                  activeTab === key ? "bg-white/20 text-white" : "bg-white/10 text-white/50"
                )}>
                  {badge}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* ── Tab content ── */}
        <AnimatePresence mode="wait">

          {/* ── CHAT ── */}
          {activeTab === "chat" && (
            <motion.div key="chat" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }}>
              <div className="nexus-card h-[420px] overflow-y-auto p-4 space-y-4 scroll-smooth">
                {messages.map((msg, i) => <ChatBubble key={i} msg={msg} />)}
                {sending && (
                  <div className="flex gap-2.5">
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-nexus-600/30 to-blue-600/30 flex items-center justify-center flex-shrink-0">
                      <Brain size={14} className="text-nexus-300" />
                    </div>
                    <div className="nexus-card px-4 py-2.5 rounded-2xl rounded-tl-sm border border-white/5 flex items-center gap-1.5">
                      <span className="w-1.5 h-1.5 bg-nexus-400 rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
                      <span className="w-1.5 h-1.5 bg-nexus-400 rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
                      <span className="w-1.5 h-1.5 bg-nexus-400 rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>

              <div className="flex gap-2 mt-2">
                <input
                  ref={inputRef}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleChat()}
                  placeholder="Ask Nexus anything…"
                  className="nexus-input flex-1 text-sm"
                  disabled={sending}
                />
                <button
                  onClick={handleChat}
                  disabled={sending || !input.trim()}
                  className={cn(
                    "px-4 py-3 rounded-xl transition-all",
                    input.trim() && !sending
                      ? "bg-gradient-to-r from-nexus-600 to-purple-600 text-white hover:opacity-90 active:scale-95"
                      : "bg-white/5 text-white/20 cursor-not-allowed"
                  )}
                >
                  {sending ? <Loader2 size={16} className="animate-spin" /> : <Send size={16} />}
                </button>
              </div>
              <div className="flex items-center justify-between mt-2 px-0.5">
                <p className="text-white/25 text-[10px]">
                  💬 Nexus AI Chat is always free · No points used
                </p>
                <button
                  onClick={handleClearChat}
                  className="text-white/25 hover:text-white/55 text-[10px] flex items-center gap-1 transition-colors"
                >
                  <RotateCcw size={9} /> Clear chat
                </button>
              </div>
            </motion.div>
          )}

          {/* ── TOOLS ── */}
          {activeTab === "tools" && (
            <motion.div key="tools" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} className="space-y-4">
              <div className="space-y-2.5">
                <input
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search tools…"
                  className="nexus-input text-sm w-full"
                />
                <div className="flex gap-1.5 overflow-x-auto pb-1 scrollbar-hide">
                  <button
                    onClick={() => setActiveCategory(null)}
                    className={cn(
                      "flex-shrink-0 text-xs px-3 py-1.5 rounded-full border transition-all font-medium",
                      !activeCategory
                        ? "bg-nexus-600 text-white border-nexus-500"
                        : "text-white/50 border-white/10 hover:text-white/80"
                    )}
                  >
                    All
                  </button>
                  {categories.map((cat) => {
                    const cfg = catCfg(cat);
                    return (
                      <button
                        key={cat}
                        onClick={() => setActiveCategory(activeCategory === cat ? null : cat)}
                        className={cn(
                          "flex-shrink-0 text-xs px-3 py-1.5 rounded-full border transition-all font-medium flex items-center gap-1",
                          activeCategory === cat ? cfg.badge : "text-white/50 border-white/10 hover:text-white/80"
                        )}
                      >
                        {cfg.icon}
                        {cat.split(" ")[0]}
                      </button>
                    );
                  })}
                </div>
              </div>

              {toolsLoading ? (
                <div className="space-y-2">
                  {[...Array(6)].map((_, i) => (
                    <div key={i} className="nexus-card h-16 animate-pulse opacity-50" />
                  ))}
                </div>
              ) : tools.length === 0 ? (
                /* ── Empty state: no tools loaded at all ── */
                <div className="text-center py-16 nexus-card space-y-4">
                  <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-nexus-600/20 to-purple-600/20 border border-white/10 flex items-center justify-center mx-auto">
                    <Sparkles size={28} className="text-nexus-400" />
                  </div>
                  <div>
                    <p className="text-white/60 text-base font-semibold">No tools available yet</p>
                    <p className="text-white/30 text-sm mt-1">AI tools will appear here once they&apos;re activated</p>
                  </div>
                  <button
                    onClick={() => setActiveTab("chat")}
                    className="nexus-btn-primary text-sm px-5 py-2.5 mx-auto flex items-center gap-1.5"
                  >
                    <MessageSquare size={14} /> Try AI Chat instead
                  </button>
                </div>
              ) : Object.keys(groupedTools).length === 0 ? (
                /* ── Empty state: search / filter returns nothing ── */
                <div className="text-center py-12 text-white/30 nexus-card space-y-3">
                  <Wand2 size={32} className="mx-auto mb-3 opacity-40" />
                  <p className="text-sm font-medium">No tools match your search</p>
                  <button
                    onClick={() => { setSearchQuery(""); setActiveCategory(null); }}
                    className="text-nexus-400 text-xs hover:text-nexus-300 transition-colors underline underline-offset-2"
                  >
                    Clear filters
                  </button>
                </div>
              ) : (
                Object.entries(groupedTools).map(([cat, catTools]) => {
                  const cfg = catCfg(cat);
                  return (
                    <div key={cat}>
                      <div className="flex items-center gap-2 mb-2 px-1">
                        <span className={cn("flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider px-2.5 py-1 rounded-full", cfg.badge)}>
                          {cfg.icon} {cat}
                        </span>
                        <span className="text-white/20 text-[10px]">{catTools.length} tool{catTools.length !== 1 ? "s" : ""}</span>
                      </div>
                      <div className="space-y-1.5">
                        {catTools.map((tool) => (
                          <ToolCard key={tool.id} tool={tool} onClick={() => setSelectedTool(tool)} />
                        ))}
                      </div>
                    </div>
                  );
                })
              )}

              <div className="flex items-center gap-2 nexus-card p-3">
                <Info size={13} className="text-nexus-400 flex-shrink-0" />
                <p className="text-white/40 text-xs">
                  Points are only deducted after you confirm. Failed generations are automatically refunded.
                </p>
              </div>
            </motion.div>
          )}

          {/* ── GALLERY ── */}
          {activeTab === "gallery" && (
            <motion.div key="gallery" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} className="space-y-3">
              <div className="flex items-center justify-between">
                <p className="text-white/60 text-sm">Recent generations</p>
                <button onClick={() => mutateGallery()} className="text-white/30 hover:text-white/60 transition-colors">
                  <RefreshCw size={14} />
                </button>
              </div>

              {recentGens.length === 0 ? (
                <div className="text-center py-14 nexus-card space-y-3">
                  <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-nexus-600/20 to-purple-600/20 border border-white/10 flex items-center justify-center mx-auto">
                    <Play size={24} className="text-white/30" />
                  </div>
                  <p className="text-white/40 text-sm font-medium">No generations yet</p>
                  <p className="text-white/25 text-xs">Use a tool above to create something amazing</p>
                  <button
                    onClick={() => setActiveTab("tools")}
                    className="nexus-btn-primary text-sm px-5 py-2.5 mx-auto flex items-center gap-1.5"
                  >
                    <Wand2 size={14} /> Browse tools
                  </button>
                </div>
              ) : (
                recentGens.map((gen) => <GenerationCard key={gen.id} gen={gen} />)
              )}

              {gallery.length > 6 && (
                <a href="/studio/gallery" className="nexus-btn-outline w-full py-3 text-sm flex items-center justify-center gap-2">
                  View all {gallery.length} generations <ExternalLink size={13} />
                </a>
              )}
            </motion.div>
          )}

        </AnimatePresence>
      </div>

      {/* ── Tool drawer ── */}
      <AnimatePresence>
        {selectedTool && (
          <ToolDrawer
            tool={selectedTool}
            onClose={() => setSelectedTool(null)}
            userPoints={userPoints}
          />
        )}
      </AnimatePresence>
    </AppShell>
  );
}
