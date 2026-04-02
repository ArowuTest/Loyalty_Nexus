"use client";

import { useState, useRef, useEffect, useCallback, Suspense } from "react";
import { motion, AnimatePresence } from "framer-motion";
import useSWR from "swr";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import { useStore } from "@/store/useStore";
import toast, { Toaster } from "react-hot-toast";
import {
  Send, User, Loader2, Wand2, Image as ImageIcon, BookOpen,
  Mic, FileText, Music, Globe, ChevronRight, Sparkles,
  AlertTriangle, CheckCircle2, Clock, ExternalLink, RefreshCw,
  Brain, Video, X, Info, Play, LayoutGrid, MessageSquare, History,
  Code2, Copy, Check, Download, RotateCcw, Zap, CreditCard,
  TrendingUp, Timer, ChevronDown, Lock, Activity,
  Paperclip, AlertCircle,
} from "lucide-react";
import {
  MusicComposer, ImageCreator, ImageEditor,
  VideoCreator, VideoAnimator, VoiceStudio,
  Transcribe, VisionAsk, KnowledgeDoc,
} from "../../components/studio/templates";
import type { GeneratePayload } from "../../components/studio/templates";
import type { UITemplate, UIConfig } from "../../types/studio";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { useSearchParams } from "next/navigation";

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
  entry_point_cost: number;
  refund_window_mins: number;
  refund_pct: number;
  is_free: boolean;
  ui_template?: UITemplate;
  ui_config?: UIConfig;
}
interface SessionUsage {
  active: boolean;
  session_id?: string;
  total_pts_used: number;
  generation_count: number;
  started_at?: string;
  last_active_at?: string;
}
type ChatMode = 'general' | 'search' | 'code';
interface Message {
  role: "user" | "assistant";
  content: string;
  provider?: string;
  ts?: number;
  mode?: ChatMode;
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
  error_message?: string;
  disputed_at?: string;
  refund_granted?: boolean;
  refund_pts?: number;
  refund_window_mins?: number;
  refund_pct?: number;
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
  // Alias slugs — map to same meta as their canonical counterparts
  "my-ai-photo":         { time: "~8 sec",   output: "1024×1024 image",               tip: "Add style words: 'photorealistic', 'vibrant', 'cinematic'" },
  "background-remover":  { time: "~5 sec",   output: "Transparent PNG",               tip: "Works best with clear subject vs background" },
  "animate-my-photo":    { time: "~45 sec",  output: "5-second MP4 video",            tip: "Use portraits or scenic photos for best motion" },
  "my-video-story":      { time: "~45 sec",  output: "5-second MP4 video",            tip: "Use portraits or scenic photos for best motion" },
  "my-marketing-jingle": { time: "~25 sec",  output: "AI music jingle",               tip: "Add brand name and target emotion in prompt" },
  "my-podcast":          { time: "~90 sec",  output: "2-host AI podcast audio",       tip: "Give a clear topic — the AI writes the full script" },
  "local-translation":   { time: "~3 sec",   output: "Translated text",               tip: "Format: type your text, select target language" },
  "voice-to-text":       { time: "~6 sec",   output: "Text transcript",               tip: "Paste a direct link to an MP3 or WAV file" },
  "text-to-speech":      { time: "~4 sec",   output: "MP3 audio file",                tip: "Keep text under 500 words for best quality" },
};

// ─── Output type helpers ──────────────────────────────────────────────────────
const IMAGE_SLUGS  = new Set(["ai-photo","ai-photo-pro","ai-photo-max","ai-photo-dream","photo-editor","animate-photo","my-ai-photo","background-remover","bg-remover"]);
const AUDIO_SLUGS  = new Set(["narrate","narrate-pro","bg-music","jingle","my-marketing-jingle","song-creator","instrumental","transcribe","transcribe-african","podcast","my-podcast"]);
const VIDEO_SLUGS  = new Set(["animate-photo","video-premium","video-cinematic","video-veo","animate-my-photo","my-video-story"]);
const CODE_SLUGS   = new Set(["code-helper"]);
const VISION_SLUGS = new Set(["image-analyser","ask-my-photo"]);
const WEB_SLUGS    = new Set(["web-search-ai"]);
const JSON_SLUGS   = new Set(["quiz","mindmap","slide-deck"]);

function getOutputType(slug: string): { label: string; emoji: string; noun: string } {
  if (VIDEO_SLUGS.has(slug))  return { label: "Video MP4",  emoji: "🎬", noun: "video" };
  if (AUDIO_SLUGS.has(slug))  return { label: "Audio MP3",  emoji: "🎵", noun: "audio" };
  if (IMAGE_SLUGS.has(slug))  return { label: "Image file", emoji: "🖼️", noun: "image" };
  if (CODE_SLUGS.has(slug))   return { label: "Code output",emoji: "💻", noun: "code" };
  return { label: "Text output", emoji: "📄", noun: "text" };
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

// ─── New tool slugs ──────────────────────────────────────────────────────────
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

// ─── Alias / duplicate slugs hidden from the tool grid ────────────────────────
// These are backend aliases for canonical tools. We hide them to avoid clutter
// and show only the ~28 canonical tools to users.
const HIDDEN_ALIAS_SLUGS = new Set([
  "my-ai-photo",         // alias → ai-photo
  "background-remover",  // alias → bg-remover
  "animate-my-photo",    // alias → animate-photo
  "my-video-story",      // alias → animate-photo
  "my-marketing-jingle", // alias → jingle
  "my-podcast",          // alias → podcast
  "local-translation",   // alias → translate
  "voice-to-text",       // alias → transcribe
  "text-to-speech",      // alias → narrate
  "business-plan",       // alias → bizplan
  "summary",             // alias → research-brief
  "ai-chat",             // handled via Chat tab — not a standalone tool card
]);

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

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins  = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days  = Math.floor(diff / 86400000);
  if (mins  < 1)  return "just now";
  if (mins  < 60) return `${mins}m ago`;
  if (hours < 24) return `${hours}h ago`;
  return `${days}d ago`;
}

// ─── Points cost badge ────────────────────────────────────────────────────────
function PointsBadge({ pointCost, size = "sm" }: { pointCost: number; size?: "xs" | "sm" }) {
  const isFree    = pointCost === 0;
  const isPremium = pointCost >= 20;
  const base      = size === "xs" ? "text-[9px] px-1.5 py-0.5" : "text-xs px-2 py-0.5";
  return (
    <span className={cn(
      "font-bold rounded-full border leading-none",
      base,
      isFree
        ? "bg-green-500/20 text-green-300 border-green-500/30"
        : isPremium
          ? "bg-amber-500/20 text-amber-300 border-amber-500/30"
          : "bg-gold-500/15 text-gold-400 border-gold-500/25"
    )}>
      {isFree ? "Free" : `${pointCost} pts/gen`}
    </span>
  );
}

// ─── Wallet bar ───────────────────────────────────────────────────────────────
function WalletBar({ userPoints }: { userPoints: number }) {
  const isLow = userPoints < 50;
  return (
    <motion.div
      initial={{ opacity: 0, y: -6 }}
      animate={{ opacity: 1, y: 0 }}
      className={cn(
        "glass border border-white/[0.08] p-3 flex items-center justify-between gap-3",
        isLow
          ? "border-amber-500/30 bg-gradient-to-r from-amber-500/8 to-orange-500/5"
          : "border-gold-500/15 bg-gradient-to-r from-gold-500/5 to-amber-600/3"
      )}
    >
      <div className="flex items-center gap-2.5">
        <div className={cn(
          "w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0",
          isLow ? "bg-amber-500/20" : "bg-gold-500/15"
        )}>
          <Zap size={15} className={isLow ? "text-amber-400" : "text-gold-500"} />
        </div>
        <div>
          <div className="flex items-baseline gap-1.5">
            <span className={cn("font-bold text-base leading-none", isLow ? "text-amber-300" : "text-white")}>
              {userPoints.toLocaleString()}
            </span>
            <span className="text-white/40 text-xs">PulsePoints</span>
          </div>
          <p className="text-white/35 text-[10px] mt-0.5 leading-none">Each generation uses points once</p>
        </div>
      </div>
      {isLow ? (
        <Link
          href="/dashboard"
          className="flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-xl
                     bg-amber-500/20 text-amber-300 border border-amber-500/30
                     hover:bg-amber-500/30 transition-all flex-shrink-0"
        >
          <Zap size={12} />  Recharge
        </Link>
      ) : (
        <div className="flex items-center gap-1 text-white/25 text-[10px] flex-shrink-0">
          <TrendingUp size={10} />
          <span>Good balance</span>
        </div>
      )}
    </motion.div>
  );
}

// ─── Session utilisation bar ─────────────────────────────────────────────────
function SessionBar({ userPoints }: { userPoints: number }) {
  const [session, setSession] = useState<{
    active: boolean; total_pts_used: number; generation_count: number;
    started_at?: string; session_id?: string;
  } | null>(null);
  const [chatInfo, setChatInfo] = useState<{ used: number; limit: number } | null>(null);

  useEffect(() => {
    let iv: ReturnType<typeof setInterval>;
    let emptyCount = 0;
    const fetchAll = async () => {
      try {
        const [sess, chat] = await Promise.all([
          (api as any).getSessionUsage(),
          (api as any).getChatUsage(),
        ]);
        const isActive = sess?.active && (sess.total_pts_used > 0 || sess.generation_count > 0);
        if (isActive) {
          setSession(sess);
          emptyCount = 0;
        } else {
          setSession(null);
          emptyCount++;
          // Stop polling after 3 consecutive empty results — no active session
          // The bar re-mounts naturally when the user triggers a generation
          if (emptyCount >= 3) { clearInterval(iv); return; }
        }
        if (chat?.limit != null) setChatInfo(chat);
      } catch { /* silent */ }
    };
    fetchAll();
    iv = setInterval(fetchAll, 10000);
    return () => clearInterval(iv);
  }, []);

  const hasSession = session?.active && (session.total_pts_used > 0 || session.generation_count > 0);
  const hasChat    = chatInfo && chatInfo.used > 0;
  if (!hasSession && !hasChat) return null;

  // Points usage bar
  const pct = userPoints > 0 && session
    ? Math.min(100, (session.total_pts_used / (userPoints + session.total_pts_used)) * 100)
    : 0;
  const barColor = pct < 30 ? "from-green-500 to-emerald-400"
    : pct < 60 ? "from-amber-500 to-yellow-400"
    : "from-red-500 to-rose-400";
  const textColor = pct < 30 ? "text-green-300" : pct < 60 ? "text-amber-300" : "text-red-300";

  return (
    <motion.div
      initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }}
      className="glass border border-white/[0.08] p-2.5 border-white/5 bg-white/[0.02] space-y-2"
    >
      {/* Generation session row */}
      {hasSession && session && (
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <div className="flex items-center gap-1.5">
              <Activity size={11} className={textColor} />
              <span className="text-white/40 text-[10px] uppercase tracking-wider">Session usage</span>
            </div>
            <div className="flex items-center gap-2">
              <span className={cn("text-[10px] font-bold tabular-nums", textColor)}>
                {session.total_pts_used} pts used
              </span>
              <span className="text-white/20 text-[10px]">{session.generation_count} gen{session.generation_count !== 1 ? "s" : ""}</span>
            </div>
          </div>
          <div className="h-1 w-full rounded-full bg-white/8 overflow-hidden">
            <div
              className={cn("h-full rounded-full bg-gradient-to-r transition-all duration-700", barColor)}
              style={{ width: `${pct}%` }}
            />
          </div>
        </div>
      )}
      {/* Chat message row */}
      {hasChat && chatInfo && (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5">
            <MessageSquare size={11} className="text-gold-500" />
            <span className="text-white/40 text-[10px] uppercase tracking-wider">Chat today</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="h-1 w-16 rounded-full bg-white/10 overflow-hidden">
              <div
                className="h-full rounded-full bg-gradient-to-r from-gold-500 to-amber-500 transition-all duration-500"
                style={{ width: `${Math.min(100, (chatInfo.used / chatInfo.limit) * 100)}%` }}
              />
            </div>
            <span className="text-gold-400 text-[10px] font-bold tabular-nums">
              {chatInfo.used}/{chatInfo.limit}
            </span>
          </div>
        </div>
      )}
    </motion.div>
  );
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

// ─── Download text as file button ───────────────────────────────────────────
function DownloadTextButton({ text, filename = "nexus-output.txt", label = "Download .txt" }: { text: string; filename?: string; label?: string }) {
  const handleDownload = () => {
    try {
      const blob = new Blob([text], { type: 'text/plain;charset=utf-8' });
      const url  = URL.createObjectURL(blob);
      const a    = document.createElement('a');
      a.href     = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      toast.success('File downloaded!');
    } catch {
      toast.error('Download failed');
    }
  };
  return (
    <button
      onClick={handleDownload}
      className="flex items-center gap-1 text-[10px] font-medium px-2.5 py-1 rounded-lg
                 bg-white/10 hover:bg-white/20 text-white/60 hover:text-white transition-all"
    >
      <Download size={11} /> {label}
    </button>
  );
}

// ─── Styled audio player ──────────────────────────────────────────────────────
function AudioPlayer({ src, label = "Audio" }: { src: string; label?: string }) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [playing,  setPlaying]  = useState(false);
  const [progress, setProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const [current,  setCurrent]  = useState(0);

  const fmt = (s: number) => {
    if (!isFinite(s)) return '0:00';
    const m = Math.floor(s / 60);
    const sec = Math.floor(s % 60);
    return `${m}:${sec.toString().padStart(2, '0')}`;
  };

  const togglePlay = () => {
    const a = audioRef.current;
    if (!a) return;
    if (playing) { a.pause(); setPlaying(false); }
    else         { a.play().then(() => setPlaying(true)).catch(() => {}); }
  };

  const handleSeek = (e: React.MouseEvent<HTMLDivElement>) => {
    const a = audioRef.current;
    if (!a || !duration) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const pct  = (e.clientX - rect.left) / rect.width;
    a.currentTime = pct * duration;
  };

  return (
    <div className="bg-gradient-to-br from-[#1a1040]/80 to-[#0f0a2a]/60 border border-purple-500/20 rounded-2xl p-4 space-y-3">
      {/* Waveform bars — Suno-style purple/violet gradient */}
      <div className="flex items-end gap-0.5 h-10 px-1">
        {Array.from({ length: 44 }, (_, i) => {
          const h = 15 + Math.abs(Math.sin(i * 0.7 + 0.5) * 55 + Math.cos(i * 1.1) * 25);
          const filled = progress > 0 && (i / 44) * 100 < progress;
          const isActive = playing && filled;
          return (
            <div
              key={i}
              className={cn(
                'flex-1 rounded-full transition-all duration-150',
                isActive
                  ? 'bg-gradient-to-t from-purple-600 to-violet-400'
                  : filled
                  ? 'bg-gradient-to-t from-purple-700/80 to-violet-500/60'
                  : 'bg-white/10',
              )}
              style={{
                height: `${Math.max(8, h)}%`,
                transform: isActive ? 'scaleY(1.1)' : 'scaleY(1)',
              }}
            />
          );
        })}
      </div>
      {/* Progress bar — clickable */}
      <div
        className="h-1.5 w-full rounded-full bg-white/10 cursor-pointer overflow-hidden"
        onClick={handleSeek}
      >
        <div
          className="h-full rounded-full bg-gradient-to-r from-purple-600 to-violet-400 transition-all duration-200"
          style={{ width: `${progress}%` }}
        />
      </div>
      {/* Controls row */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button
            onClick={togglePlay}
            className="w-9 h-9 rounded-full bg-gradient-to-br from-purple-600 to-violet-500 flex items-center justify-center hover:opacity-90 active:scale-95 transition-all shadow-lg shadow-purple-900/40"
          >
            {playing
              ? <span className="flex gap-0.5"><span className="w-1 h-3.5 bg-white rounded-full" /><span className="w-1 h-3.5 bg-white rounded-full" /></span>
              : <Play size={14} className="text-white ml-0.5" />}
          </button>
          <div className="text-xs text-white/40 tabular-nums font-mono">
            {fmt(current)} / {fmt(duration)}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-white/30 text-[10px] font-medium">{label}</span>
          <a
            href={src}
            download
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 text-[10px] px-2.5 py-1 rounded-lg bg-white/10 hover:bg-white/20 text-white/60 hover:text-white transition-all"
          >
            <Download size={10} /> Download
          </a>
        </div>
      </div>
      {/* Hidden native audio element */}
      <audio
        ref={audioRef}
        src={src}
        onTimeUpdate={() => {
          const a = audioRef.current;
          if (!a) return;
          setCurrent(a.currentTime);
          setProgress(a.duration ? (a.currentTime / a.duration) * 100 : 0);
        }}
        onLoadedMetadata={() => { if (audioRef.current) setDuration(audioRef.current.duration); }}
        onEnded={() => { setPlaying(false); setProgress(0); setCurrent(0); }}
        className="hidden"
      />
    </div>
  );
}

// ─── Intro / How It Works banner ─────────────────────────────────────────────
function HowItWorksBanner({ onDismiss }: { onDismiss: () => void }) {
  const steps = [
    { icon: "🔍", label: "Choose a tool" },
    { icon: "✏️", label: "Describe what you want" },
    { icon: "⚡", label: "Points deducted once" },
    { icon: "⬇", label: "Download your output" },
  ];
  return (
    <motion.div
      initial={{ opacity: 0, y: -8 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -8 }}
      className="glass border border-white/[0.08] p-4 border border-gold-500/15 bg-gradient-to-r from-gold-500/8 to-amber-600/5"
    >
      <div className="flex items-start justify-between gap-2 mb-3">
        <div>
          <p className="text-white/80 text-xs font-semibold uppercase tracking-wider">How It Works</p>
          <p className="text-white/35 text-[10px] mt-0.5">One generation = one point deduction. Failures are auto-refunded.</p>
        </div>
        <button onClick={onDismiss} className="text-white/30 hover:text-white/70 transition-colors flex-shrink-0">
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

// ─── Confirm modal ────────────────────────────────────────────────────────────
function ConfirmModal({
  tool, prompt, onConfirm, onCancel, busy, userPoints,
}: {
  tool: Tool; prompt: string; onConfirm: () => void;
  onCancel: () => void; busy: boolean; userPoints: number;
}) {
  const cfg      = catCfg(tool.category);
  const isFree   = tool.point_cost === 0;
  const canAfford= userPoints >= tool.point_cost;
  const after    = userPoints - tool.point_cost;
  const outType  = getOutputType(tool.slug);

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
        <div className="glass border border-white/[0.08] overflow-hidden">
          <div className={cn("h-1.5 w-full bg-gradient-to-r", cfg.color.replace("/20","/60").replace("/10","/40"))} />
          <div className="p-6 space-y-5">
            {/* Tool header */}
            <div className="flex items-start gap-3">
              <div className={cn("p-2.5 rounded-xl bg-gradient-to-br", cfg.color)}>{cfg.icon}</div>
              <div>
                <h3 className="text-white font-bold text-lg leading-tight">{tool.name}</h3>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-white/40 text-xs">{outType.emoji} Outputs 1 {outType.noun}</span>
                </div>
              </div>
            </div>

            {/* Prompt preview */}
            <div className="bg-white/5 border border-white/10 rounded-xl p-3">
              <p className="text-white/40 text-xs uppercase tracking-wider mb-1 font-medium">Your prompt</p>
              <p className="text-white/80 text-sm line-clamp-3">{prompt}</p>
            </div>

            {/* Points summary box */}
            <div className={cn(
              "rounded-xl border p-4 space-y-2",
              !isFree && !canAfford
                ? "border-red-500/30 bg-red-500/8"
                : isFree
                  ? "border-green-500/25 bg-green-500/8"
                  : "border-gold-500/20 bg-gold-500/5"
            )}>
              {isFree ? (
                <div className="flex items-center gap-2 text-green-300">
                  <CheckCircle2 size={15} />
                  <span className="font-semibold text-sm">Free generation — no points needed</span>
                </div>
              ) : (
                <>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-white/55 flex items-center gap-1.5">
                      <Zap size={12} className="text-gold-500" /> Generation cost
                    </span>
                    <span className="font-bold text-white">−{tool.point_cost} pts</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-white/55">Your balance</span>
                    <span className="font-semibold text-gold-400">{userPoints.toLocaleString()} pts</span>
                  </div>
                  <div className={cn(
                    "h-px w-full",
                    canAfford ? "bg-gold-500/15" : "bg-red-500/20"
                  )} />
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-white/55">Balance after</span>
                    <span className={cn("font-bold", canAfford ? "text-gold-400" : "text-red-400")}>
                      {canAfford ? `${after.toLocaleString()} pts remaining` : "⚠ Insufficient"}
                    </span>
                  </div>
                </>
              )}
            </div>

            {/* Insufficient points callout */}
            {!canAfford && !isFree && (
              <div className="flex items-start gap-2.5 bg-red-500/10 border border-red-500/20 rounded-xl p-3">
                <AlertTriangle size={16} className="text-red-400 flex-shrink-0 mt-0.5" />
                <div>
                  <p className="text-red-300 text-sm font-semibold">
                    You need {(tool.point_cost - userPoints).toLocaleString()} more points
                  </p>
                  <p className="text-red-300/60 text-xs mt-0.5">
                    You have {userPoints} pts — need {tool.point_cost} pts. Top up to continue.
                  </p>
                </div>
              </div>
            )}

            {/* Refund notice */}
            {canAfford && !isFree && (
              <div className="flex items-start gap-2.5 bg-gold-500/5 border border-gold-500/15 rounded-xl p-3">
                <Info size={15} className="text-gold-500 flex-shrink-0 mt-0.5" />
                <p className="text-gold-400 text-xs leading-relaxed">
                  {tool.point_cost} pts deducted once when generation starts.
                  If the AI fails, your points are automatically refunded within seconds.
                </p>
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-2 pt-1">
              <button onClick={onCancel} className="glass border border-white/[0.10] text-white/70 hover:text-white hover:border-white/20 transition-all rounded-xl font-black flex-1 text-sm py-3">Cancel</button>
              {!canAfford && !isFree ? (
                <Link
                  href="/dashboard"
                  className="flex-1 py-3 rounded-xl text-sm font-semibold flex items-center justify-center gap-2
                             bg-gradient-to-r from-amber-600 to-orange-600 text-white hover:opacity-90"
                >
                  <CreditCard size={15} /> Recharge MTN
                </Link>
              ) : (
                <button
                  onClick={onConfirm}
                  disabled={busy}
                  className={cn(
                    "flex-1 py-3 rounded-xl text-sm font-semibold flex items-center justify-center gap-2 transition-all",
                    "bg-gradient-to-r from-gold-500/80 to-amber-600 text-white hover:opacity-90 active:scale-[0.98]",
                    busy && "opacity-70 cursor-not-allowed"
                  )}
                >
                  {busy ? (
                    <><Loader2 size={16} className="animate-spin" /> Starting…</>
                  ) : (
                    <><Sparkles size={16} /> {isFree ? "Generate (Free)" : `Use ${tool.point_cost} pts → Generate`}</>
                  )}
                </button>
              )}
            </div>
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
}

// ─── Elapsed timer ────────────────────────────────────────────────────────────
function ElapsedTimer({ startedAt }: { startedAt: number }) {
  const [elapsed, setElapsed] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setElapsed(Math.floor((Date.now() - startedAt) / 1000)), 500);
    return () => clearInterval(id);
  }, [startedAt]);
  return (
    <span className="text-white/30 text-[10px] flex items-center gap-1 tabular-nums">
      <Timer size={9} /> {elapsed}s elapsed
    </span>
  );
}

// ─── Markdown / code renderer for chat bubbles ──────────────────────────────
function RichMessage({ content, mode }: { content: string; mode: ChatMode }) {
  const [copied, setCopied] = useState<number | null>(null);

  function copyCode(text: string, idx: number) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(idx);
      setTimeout(() => setCopied(null), 1800);
    });
  }

  // Split by fenced code blocks ```lang\n...code...\n```
  const parts = content.split(/(```[\s\S]*?```)/g);

  return (
    <div className="space-y-2">
      {parts.map((part, i) => {
        if (part.startsWith('```')) {
          const firstNewline = part.indexOf('\n');
          const lang = part.slice(3, firstNewline).trim() || 'code';
          const code = part.slice(firstNewline + 1, part.lastIndexOf('```')).trim();
          return (
            <div key={i} className="rounded-xl overflow-hidden border border-white/10">
              <div className="flex items-center justify-between px-3 py-1.5 bg-white/5">
                <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider">{lang}</span>
                <button
                  onClick={() => copyCode(code, i)}
                  className="flex items-center gap-1 text-[10px] text-white/40 hover:text-white/70 transition-colors"
                >
                  {copied === i ? <Check size={10} className="text-green-400" /> : <Copy size={10} />}
                  {copied === i ? 'Copied!' : 'Copy'}
                </button>
              </div>
              <pre className="bg-gray-950 text-green-200 text-[11px] font-mono p-4 overflow-x-auto whitespace-pre leading-relaxed max-h-72 overflow-y-auto">
                <code>{code}</code>
              </pre>
            </div>
          );
        }
        // Plain text: render inline bold (**text**) and backtick `code`
        const lines = part.split('\n');
        return (
          <div key={i} className="space-y-1.5">
            {lines.map((line, j) => {
              if (!line.trim()) return <div key={j} className="h-1" />;
              // Inline formatting: **bold** and `code`
              const chunks = line.split(/(`[^`]+`|\*\*[^*]+\*\*)/g);
              return (
                <p key={j} className={cn(
                  'text-sm leading-relaxed',
                  mode === 'code' ? 'text-green-100/90' : 'text-white/85',
                )}>
                  {chunks.map((chunk, k) => {
                    if (chunk.startsWith('**') && chunk.endsWith('**'))
                      return <strong key={k} className="text-white font-semibold">{chunk.slice(2, -2)}</strong>;
                    if (chunk.startsWith('`') && chunk.endsWith('`'))
                      return <code key={k} className="bg-white/10 text-green-300 text-[11px] font-mono px-1.5 py-0.5 rounded">{chunk.slice(1, -1)}</code>;
                    return chunk;
                  })}
                </p>
              );
            })}
          </div>
        );
      })}
    </div>
  );
}

// ─── Chat bubble ──────────────────────────────────────────────────────────────
const MODE_META: Record<ChatMode, { label: string; color: string; icon: React.ReactNode }> = {
  general: { label: 'Nexus AI',    color: 'text-gold-400',  icon: <Brain size={14} className="text-gold-400" /> },
  search:  { label: 'Web Search',  color: 'text-sky-300',    icon: <Globe size={14} className="text-sky-300" /> },
  code:    { label: 'Code Helper', color: 'text-green-300',  icon: <Code2 size={14} className="text-green-300" /> },
};

function ChatBubble({ msg }: { msg: Message }) {
  const isUser = msg.role === "user";
  const mode   = msg.mode ?? 'general';
  const meta   = MODE_META[mode];
  return (
    <div className={cn("flex gap-2.5 group", isUser && "flex-row-reverse")}>
      <div className={cn(
        "w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5",
        isUser ? "bg-gradient-to-br from-gold-500/20 to-amber-600/15"
               : mode === 'search' ? "bg-sky-600/20 border border-sky-500/20"
               : mode === 'code'   ? "bg-green-600/20 border border-green-500/20"
               :                     "bg-gradient-to-br from-gold-500/15 to-amber-600/10"
      )}>
        {isUser ? <User size={14} className="text-purple-300" /> : meta.icon}
      </div>
      <div className={cn("max-w-[80%] space-y-1", isUser && "items-end flex flex-col")}>
        <div className={cn(
          "px-4 py-2.5",
          isUser
            ? "bg-gradient-to-br from-gold-500/80 to-amber-600 text-white rounded-2xl rounded-tr-sm text-sm leading-relaxed"
            : mode === 'code'
              ? "bg-gray-950/80 border border-green-500/15 rounded-2xl rounded-tl-sm"
              : mode === 'search'
              ? "bg-sky-950/40 border border-sky-500/15 rounded-2xl rounded-tl-sm"
              : "bg-[#1c1e2e] rounded-2xl rounded-tl-sm border border-white/[0.07] shadow-sm"
        )}>
          {isUser
            ? <p className="text-sm leading-relaxed">{msg.content}</p>
            : <RichMessage content={msg.content} mode={mode} />
          }
        </div>
        {!isUser && (
          <div className="flex items-center gap-2 px-1">
            <p className="text-white/20 text-[9px] flex items-center gap-1">
              {meta.label}
              {msg.provider && <span>· {msg.provider}</span>}
            </p>
            {/* Quick actions for long AI responses */}
            {msg.content.length > 200 && (
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={() => navigator.clipboard.writeText(msg.content).then(() => toast.success('Copied!'))}
                  className="text-white/20 hover:text-white/50 transition-colors"
                  title="Copy response"
                >
                  <Copy size={9} />
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Tool Card ────────────────────────────────────────────────────────────────
const CHAT_REDIRECT_SLUGS = new Set(["web-search-ai", "code-helper"]);

function ToolCard({ tool, onClick, userPoints = 0 }: { tool: Tool; onClick: () => void; userPoints?: number }) {
  const cfg         = catCfg(tool.category);
  const isFree      = tool.is_free || tool.point_cost === 0;
  const isNew       = NEW_TOOL_SLUGS.has(tool.slug);
  const meta        = TOOL_META[tool.slug];
  const outType     = getOutputType(tool.slug);
  const entryLocked = !tool.is_free && tool.entry_point_cost > 0 && userPoints < tool.entry_point_cost;
  const isChatTool  = CHAT_REDIRECT_SLUGS.has(tool.slug);

  const outputColour =
    VIDEO_SLUGS.has(tool.slug) ? "bg-red-500/15 text-red-300 border-red-500/25"
    : AUDIO_SLUGS.has(tool.slug) ? "bg-green-500/15 text-green-300 border-green-500/25"
    : IMAGE_SLUGS.has(tool.slug) ? "bg-pink-500/15 text-pink-300 border-pink-500/25"
    : CODE_SLUGS.has(tool.slug)  ? "bg-lime-500/15 text-lime-300 border-lime-500/25"
    : WEB_SLUGS.has(tool.slug)   ? "bg-cyan-500/15 text-cyan-300 border-cyan-500/25"
    : "bg-white/8 text-white/40 border-white/10";

  return (
    <motion.button
      whileHover={{ y: -2, scale: 1.01 }}
      whileTap={{ scale: 0.98 }}
      onClick={onClick}
      className="w-full text-left group relative overflow-hidden rounded-2xl border border-white/[0.08] bg-white/[0.04] hover:border-white/20 hover:bg-white/[0.07] hover:shadow-card-hover transition-all duration-200 flex flex-col"
    >
      {/* Locked overlay */}
      {entryLocked && (
        <div className="absolute inset-0 bg-black/60 backdrop-blur-[3px] rounded-2xl flex flex-col items-center justify-center z-20 gap-1.5">
          <Lock size={18} className="text-amber-300/70" />
          <p className="text-amber-300/80 text-xs font-semibold">{tool.entry_point_cost} pts to unlock</p>
        </div>
      )}

      {/* Coloured header strip with icon */}
      <div className={cn(
        "w-full px-4 pt-4 pb-3 bg-gradient-to-br flex items-center justify-between",
        cfg.color
      )}>
        <div className="flex items-center gap-2.5">
          <div className="p-2 rounded-xl bg-white/10 border border-white/10 flex-shrink-0">
            {cfg.icon}
          </div>
          <div className="flex flex-col gap-0.5">
            <p className="text-white font-bold text-sm leading-tight">{tool.name}</p>
            <div className="flex items-center gap-1">
              {isNew && (
                <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-purple-500/30 text-purple-200 border border-purple-400/30 leading-none">
                  NEW
                </span>
              )}
              {isFree && (
                <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-green-500/25 text-green-200 border border-green-400/30 leading-none">
                  FREE
                </span>
              )}
              {isChatTool && (
                <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-cyan-500/25 text-cyan-200 border border-cyan-400/30 leading-none">
                  💬 Chat
                </span>
              )}
            </div>
          </div>
        </div>
        {/* Output type badge top-right */}
        <span className={cn("text-[9px] font-bold px-2 py-1 rounded-full border leading-none flex-shrink-0", outputColour)}>
          {outType.emoji} {outType.label}
        </span>
      </div>

      {/* Body */}
      <div className="px-4 py-3 flex flex-col gap-2.5 flex-1">
        {/* Description — full 2 lines, readable */}
        <p className="text-white/65 text-xs leading-relaxed line-clamp-2">{tool.description}</p>

        {/* What you get row */}
        {meta && (
          <div className="flex items-start gap-1.5 bg-white/[0.04] border border-white/[0.06] rounded-xl px-3 py-2">
            <Sparkles size={10} className="text-gold-400 mt-0.5 flex-shrink-0" />
            <p className="text-white/40 text-[10px] leading-relaxed">
              <span className="text-white/60 font-semibold">You get:</span> {meta.output}
            </p>
          </div>
        )}

        {/* Pro tip */}
        {meta?.tip && (
          <div className="flex items-start gap-1.5">
            <Info size={9} className="text-white/25 mt-0.5 flex-shrink-0" />
            <p className="text-white/30 text-[10px] leading-relaxed italic">{meta.tip}</p>
          </div>
        )}

        {/* Footer: cost + time + CTA */}
        <div className="flex items-center justify-between mt-auto pt-1">
          <div className="flex items-center gap-2">
            <PointsBadge pointCost={tool.point_cost} size="xs" />
            {meta && (
              <span className="text-[9px] text-white/30 flex items-center gap-0.5 font-medium">
                <Clock size={9} /> {meta.time}
              </span>
            )}
          </div>
          <div className={cn(
            "flex items-center gap-1 text-[10px] font-bold px-3 py-1.5 rounded-xl border transition-all",
            isChatTool
              ? "bg-cyan-600/20 text-cyan-300 border-cyan-500/30 group-hover:bg-cyan-600 group-hover:text-white group-hover:border-cyan-500"
              : "bg-gold-500/10 text-gold-400 border-gold-500/25 group-hover:bg-gold-500/25 group-hover:text-white group-hover:border-gold-500"
          )}>
            {isChatTool ? "Open Chat" : "Generate"} <ChevronRight size={11} />
          </div>
        </div>
      </div>
    </motion.button>
  );
}

// ─── Infographic renderer ───────────────────────────────────────────────────
function renderInfographic(text: string) {
  // Try to parse as JSON first
  let data: { title?: string; subtitle?: string; sections?: { heading?: string; icon?: string; points?: string[]; stat?: string; stat_label?: string }[] } | null = null;
  try {
    const raw = JSON.parse(text);
    if (raw && typeof raw === 'object') data = raw;
  } catch { /* not JSON — render as rich text */ }

  if (!data || !data.sections) {
    // Fallback: render as formatted markdown-like text
    return (
      <div className="bg-gradient-to-br from-white/5 to-white/[0.02] border border-white/10 rounded-2xl p-4 space-y-3">
        <div className="flex items-center gap-2 text-amber-300 text-xs font-semibold">
          <LayoutGrid size={12} /> Infographic
        </div>
        <div className="text-white/80 text-sm leading-relaxed whitespace-pre-wrap">{text}</div>
      </div>
    );
  }

  const ICONS: Record<string, string> = {
    chart: '📊', data: '📈', stats: '📉', info: 'ℹ️', tip: '💡', warning: '⚠️',
    check: '✅', star: '⭐', money: '💰', people: '👥', time: '⏱️', globe: '🌍',
    phone: '📱', idea: '🧠', growth: '🚀', default: '🔹',
  };

  return (
    <div className="space-y-3">
      {/* Title block */}
      {(data.title || data.subtitle) && (
        <div className="bg-gradient-to-r from-amber-500/15 to-gold-500/10 border border-amber-500/20 rounded-2xl p-4 text-center">
          {data.title && <h3 className="text-white font-bold text-base">{data.title}</h3>}
          {data.subtitle && <p className="text-white/55 text-xs mt-1">{data.subtitle}</p>}
        </div>
      )}
      {/* Sections grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {data.sections?.map((sec, i) => (
          <div key={i} className="bg-white/[0.04] border border-white/[0.08] rounded-xl p-3 space-y-2">
            <div className="flex items-center gap-2">
              <span className="text-lg">{ICONS[sec.icon ?? ''] ?? ICONS.default}</span>
              {sec.heading && <span className="text-white/90 text-xs font-semibold">{sec.heading}</span>}
            </div>
            {sec.stat && (
              <div className="text-center py-1">
                <div className="text-2xl font-bold text-amber-300">{sec.stat}</div>
                {sec.stat_label && <div className="text-white/40 text-[10px] mt-0.5">{sec.stat_label}</div>}
              </div>
            )}
            {Array.isArray(sec.points) && sec.points.length > 0 && (
              <ul className="space-y-1">
                {sec.points.map((pt, pi) => (
                  <li key={pi} className="text-white/65 text-[11px] flex items-start gap-1.5">
                    <span className="text-amber-400 mt-0.5 flex-shrink-0">•</span>
                    <span>{pt}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Status pill ─────────────────────────────────────────────────────────────
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

// ─── Generation card ──────────────────────────────────────────────────────────
function GenerationCard({ gen, onRegenerate }: { gen: Generation; onRegenerate?: (gen: Generation) => void }) {
  const isImage       = IMAGE_SLUGS.has(gen.tool_slug);
  const isAudio       = AUDIO_SLUGS.has(gen.tool_slug);
  const isVideo       = VIDEO_SLUGS.has(gen.tool_slug);
  const isCode        = CODE_SLUGS.has(gen.tool_slug);
  const isVision      = VISION_SLUGS.has(gen.tool_slug);
  const isWeb         = WEB_SLUGS.has(gen.tool_slug);
  const isJson        = JSON_SLUGS.has(gen.tool_slug);
  const isInfographic = gen.tool_slug === 'infographic';
  const meta     = TOOL_META[gen.tool_slug];
  const outType  = getOutputType(gen.tool_slug);

  // ── Quiz renderer ──
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

  // Colored left border accent per output type
  const accentBorder = gen.status === 'failed'
    ? 'border-l-2 border-l-red-500/50'
    : isVideo  ? 'border-l-2 border-l-red-500/40'
    : isAudio  ? 'border-l-2 border-l-purple-500/50'
    : isImage  ? 'border-l-2 border-l-pink-500/40'
    : isCode   ? 'border-l-2 border-l-lime-500/40'
    : isWeb    ? 'border-l-2 border-l-sky-500/40'
    : 'border-l-2 border-l-gold-500/30';

  return (
    <div className={cn(
      "glass border border-white/[0.08] p-4 space-y-3",
      accentBorder,
      gen.status === "failed" && "border-red-500/15"
    )}>
      {/* Header row */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-white text-sm font-semibold truncate">{gen.tool_name}</span>
          <span className={cn(
            "text-[9px] px-1.5 py-0.5 rounded-full border leading-none flex-shrink-0",
            isVideo  ? "bg-red-500/15 text-red-300 border-red-500/20"
            : isAudio ? "bg-green-500/15 text-green-300 border-green-500/20"
            : isImage ? "bg-pink-500/15 text-pink-300 border-pink-500/20"
            : isCode  ? "bg-lime-500/15 text-lime-300 border-lime-500/20"
            : isWeb   ? "bg-cyan-500/15 text-cyan-300 border-cyan-500/20"
            : "bg-white/8 text-white/35 border-white/10"
          )}>
            {outType.emoji} {outType.noun}
          </span>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          <span className="text-white/25 text-[10px]">{timeAgo(gen.created_at)}</span>
          <StatusPill status={gen.status} />
        </div>
      </div>

      {/* Prompt preview — parse JSON envelope to show human-readable prompt */}
      {gen.prompt && (() => {
        let displayPrompt = gen.prompt;
        try {
          const env = JSON.parse(gen.prompt);
          if (env?.prompt) displayPrompt = env.prompt;
        } catch { /* plain text */ }
        return displayPrompt ? (
          <p className="text-white/30 text-[11px] italic line-clamp-1">"{displayPrompt}"</p>
        ) : null;
      })()}

      {/* ── Processing state ── */}
      {gen.status === "processing" && (
        <div className="space-y-3">
          <div className="h-1 w-full rounded-full bg-white/10 overflow-hidden">
            <div className="h-full w-1/3 rounded-full bg-gradient-to-r from-gold-500 to-amber-500 animate-[progress_1.6s_ease-in-out_infinite]" />
          </div>
          <div className="space-y-2">
            <div className="h-3 rounded-lg bg-white/10 animate-pulse w-3/4" />
            <div className="h-3 rounded-lg bg-white/8 animate-pulse w-1/2" />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-gold-500 text-xs">
              <Loader2 size={12} className="animate-spin" />
              <span>Generating your {outType.noun}…</span>
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
            <div className="space-y-3">
              {/* Full-width image with rounded corners and subtle border — Midjourney-style */}
              <div className="relative group overflow-hidden rounded-2xl border border-white/10">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={gen.output_url} alt={gen.tool_name}
                  className="w-full object-contain max-h-[480px] bg-black/20"
                  loading="lazy" />
                {/* Hover overlay with quick actions — cinematic gradient */}
                <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity flex items-end justify-between p-3">
                  <a href={gen.output_url} target="_blank" rel="noreferrer"
                    className="text-white/80 hover:text-white text-[10px] flex items-center gap-1">
                    <ExternalLink size={10} /> View full size
                  </a>
                  <a href={gen.output_url} download target="_blank" rel="noreferrer"
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-xl bg-gold-500/80 hover:bg-gold-500 text-white font-semibold transition-all shadow-lg">
                    <Download size={12} /> Download
                  </a>
                </div>
              </div>
              {/* Vision tools may also return analysis text */}
              {gen.output_text && isVision && (
                <div className="bg-violet-500/5 border border-violet-500/10 rounded-xl p-3">
                  <p className="text-white/80 text-sm leading-relaxed whitespace-pre-wrap">{gen.output_text}</p>
                </div>
              )}
              <div className="flex gap-2">
                <a href={gen.output_url} download target="_blank" rel="noreferrer"
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                  <Download size={11} /> Download Image
                </a>
                <a href={gen.output_url} target="_blank" rel="noreferrer"
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                  <ExternalLink size={11} /> Open Full Size
                </a>
                {onRegenerate && (
                  <button onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
          {isAudio && !isVideo && (
            <div className="space-y-2">
              <AudioPlayer src={gen.output_url!} label={gen.tool_name} />
              {onRegenerate && (
                <button onClick={() => onRegenerate(gen)}
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                  <RotateCcw size={11} /> Regenerate
                </button>
              )}
            </div>
          )}
          {isVideo && (
            <div className="space-y-3">
              {/* Inline video player — Runway-style */}
              <div className="rounded-2xl overflow-hidden border border-white/10 bg-black">
                <video
                  controls
                  className="w-full max-h-[360px] object-contain"
                  src={gen.output_url}
                  poster=""
                  playsInline
                >
                  Your browser does not support video.
                </video>
              </div>
              <div className="flex gap-2">
                <a href={gen.output_url} download target="_blank" rel="noreferrer"
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                  <Download size={11} /> Download MP4
                </a>
                <a href={gen.output_url} target="_blank" rel="noreferrer"
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                  <ExternalLink size={11} /> Open Video
                </a>
                {onRegenerate && (
                  <button onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
          {!isImage && !isAudio && !isVideo && (
            <a href={gen.output_url} target="_blank" rel="noreferrer"
              className="flex items-center gap-2 text-gold-500 text-sm hover:text-gold-400">
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
              <div className="bg-white/5 rounded-xl p-3">
                <RichMessage content={gen.output_text} mode="search" />
              </div>
              <div className="flex gap-2">
                <CopyButton text={gen.output_text} label="📋 Copy" />
                {gen.output_text.length > 200 && (
                  <DownloadTextButton text={gen.output_text} filename="web-search-result.txt" label="⬇ Download" />
                )}
              </div>
            </div>
          )}
          {isVision && (
            <div className="space-y-2">
              <div className="bg-violet-500/5 border border-violet-500/10 rounded-xl p-3">
                <RichMessage content={gen.output_text} mode="general" />
              </div>
              <div className="flex gap-2">
                <CopyButton text={gen.output_text} label="📋 Copy Analysis" />
                {gen.output_text.length > 200 && (
                  <DownloadTextButton text={gen.output_text} filename="image-analysis.txt" label="⬇ Download" />
                )}
              </div>
            </div>
          )}
          {isCode && (
            <div className="relative">
              <div className="flex items-center justify-between bg-gray-900/80 px-3 py-1.5 rounded-t-xl border border-white/10 border-b-0">
                <span className="text-xs text-white/40 font-mono">Code output</span>
                <div className="flex gap-1.5">
                  <CopyButton text={gen.output_text} />
                  <DownloadTextButton text={gen.output_text} filename="code-output.txt" label="⬇ .txt" />
                </div>
              </div>
              <pre className="bg-gray-950 text-green-300 text-xs font-mono p-4 rounded-b-xl border border-white/10 overflow-x-auto whitespace-pre-wrap max-h-72 overflow-y-auto leading-relaxed">
                <code>{gen.output_text}</code>
              </pre>
            </div>
          )}
          {isInfographic && (
            <div className="space-y-2">
              {renderInfographic(gen.output_text)}
              <div className="flex gap-2">
                <CopyButton text={gen.output_text} label="📋 Copy" />
                {onRegenerate && (
                  <button onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
          {isJson && !isCode && !isInfographic && (
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
                <button onClick={() => onRegenerate(gen)}
                  className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                  <RotateCcw size={11} /> Regenerate
                </button>
              )}
            </div>
          )}
          {!isWeb && !isVision && !isCode && !isJson && !isInfographic && (
            <div className="space-y-2">
              <div className="bg-white/5 rounded-xl p-3">
                <RichMessage content={gen.output_text} mode="general" />
              </div>
              <div className="flex gap-2 flex-wrap">
                <CopyButton text={gen.output_text} label="📋 Copy Text" />
                {gen.output_text.length > 200 && (
                  <DownloadTextButton
                    text={gen.output_text}
                    filename={`${gen.tool_slug || 'nexus'}-output.txt`}
                    label="⬇ Download .txt"
                  />
                )}
                {onRegenerate && (
                  <button onClick={() => onRegenerate(gen)}
                    className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white transition-all">
                    <RotateCcw size={11} /> Regenerate
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Failed state ── */}
      {gen.status === "failed" && (
        <div className="space-y-2">
          <div className="flex items-start gap-2.5 bg-red-500/8 border border-red-500/20 rounded-xl p-3">
            <AlertTriangle size={14} className="text-red-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-red-300 text-xs font-semibold">Generation failed</p>
              {gen.error_message && (
                <p className="text-red-300/60 text-[11px] mt-0.5 leading-relaxed">{gen.error_message}</p>
              )}
            </div>
          </div>
          {gen.point_cost !== undefined && gen.point_cost > 0 && (
            <div className="flex items-center gap-2 text-green-300 text-xs bg-green-500/8 border border-green-500/20 rounded-xl px-3 py-2">
              <CheckCircle2 size={12} />
              <span className="font-semibold">Points Refunded ✓</span>
              <span className="text-green-300/60">{gen.point_cost} pts returned to your balance</span>
            </div>
          )}
        </div>
      )}

      {/* Dispute UI intentionally hidden — admin review flow not yet built.
           Backend endpoints (POST /dispute) are dormant but preserved for phase 2. */}

      {/* ── Footer ── */}
      {gen.status === "completed" && (
        <div className="flex items-center justify-between pt-1 border-t border-white/5">
          <span className="text-white/20 text-[10px]">Generated by Nexus AI</span>
          {gen.point_cost !== undefined && (
            <span className="text-white/25 text-[10px] flex items-center gap-1">
              <Zap size={9} />
              {gen.point_cost === 0
                ? "Free generation"
                : `${gen.point_cost} pts deducted — 1 ${outType.noun} generated`}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

// DisputeButton — removed pending admin review infrastructure (phase 2)
// Backend POST /api/v1/studio/generate/{id}/dispute is dormant but preserved.

// ─── Template router ─────────────────────────────────────────────────────────
// Picks the purpose-built input component based on ui_template from the API.
// Falls back to KnowledgeDoc for any unknown template.
function renderTemplate(
  tool: Tool,
  onSubmit: (p: GeneratePayload) => void,
  isLoading: boolean,
  userPoints: number,
) {
  // Cast Tool → StudioTool-compatible shape (same fields, Tool just omits icon)
  const t = tool as unknown as import("../../types/studio").StudioTool;
  const props = { tool: t, onSubmit, isLoading, userPoints };
  switch (tool.ui_template) {
    case "music-composer":  return <MusicComposer  {...props} />;
    case "image-creator":   return <ImageCreator   {...props} />;
    case "image-editor":    return <ImageEditor    {...props} />;
    case "video-creator":   return <VideoCreator   {...props} />;
    case "video-animator":  return <VideoAnimator  {...props} />;
    case "voice-studio":    return <VoiceStudio    {...props} />;
    case "transcribe":      return <Transcribe     {...props} />;
    case "vision-ask":      return <VisionAsk      {...props} />;
    case "knowledge-doc":
    default:                return <KnowledgeDoc   {...props} />;
  }
}

// ─── Tool drawer ──────────────────────────────────────────────────────────────
function ToolDrawer({
  tool, onClose, userPoints, onGenerated,
}: {
  tool: Tool; onClose: () => void; userPoints: number; onGenerated?: () => void;
}) {
  // pendingPayload holds the GeneratePayload from the template until the user
  // confirms in the ConfirmModal. null = no payload ready yet.
  const [pendingPayload, setPendingPayload] = useState<GeneratePayload | null>(null);
  const [showConfirm,    setShowConfirm]    = useState(false);
  const [generating,     setGenerating]     = useState(false);
  const [genStartedAt,   setGenStartedAt]   = useState<number | null>(null);

  const cfg        = catCfg(tool.category);
  const slug       = tool.slug;
  const meta       = TOOL_META[slug];
  const isFree     = tool.is_free || tool.point_cost === 0;
  const isPremium  = tool.point_cost >= 20;
  const isNew      = NEW_TOOL_SLUGS.has(slug);
  const canAfford  = isFree || userPoints >= tool.point_cost;
  const after      = userPoints - tool.point_cost;
  const entryLocked = !tool.is_free && tool.entry_point_cost > 0 && userPoints < tool.entry_point_cost;
  const outType    = getOutputType(slug);

  // Called by a template component when the user clicks its Generate button.
  // We stash the payload then open the confirmation modal.
  function handleTemplateSubmit(payload: GeneratePayload) {
    if (generating) return;
    setPendingPayload(payload);
    setShowConfirm(true);
  }

  // Called when the user confirms in the ConfirmModal.
  const handleConfirmedGenerate = async () => {
    if (!pendingPayload) return;
    setGenerating(true);
    setGenStartedAt(Date.now());
    try {
      const res = await api.generateTool(tool.id, pendingPayload) as { generation_id: string; status: string };
      toast.success("⚡ Generating… check Gallery tab for result.");
      onClose();
      if (res?.generation_id) {
        const genId = res.generation_id;
        let attempts = 0;
        const poll = setInterval(async () => {
          attempts++;
          if (attempts > 60) { clearInterval(poll); return; }
          try {
            const status = await api.getGenerationStatus(genId) as { status: string };
            if (status?.status === "completed" || status?.status === "failed") {
              clearInterval(poll);
              onGenerated?.();
              if (status.status === "completed") {
                toast.success(`✅ ${tool.name} ready! Check Gallery.`, { duration: 5000 });
              } else {
                toast.error(`${tool.name} failed. Points refunded automatically.`);
              }
            }
          } catch { clearInterval(poll); }
        }, 2000);
      } else {
        setTimeout(() => onGenerated?.(), 5000);
      }
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Failed to start generation");
    } finally {
      setGenerating(false);
      setShowConfirm(false);
    }
  };

  // The prompt shown in the confirm modal — extracted from the pending payload
  // Parse JSON envelope so the confirm modal shows a human-readable summary
  const confirmPrompt = (() => {
    const raw = pendingPayload?.prompt ?? "";
    try {
      const env = JSON.parse(raw);
      if (env?.prompt) return env.prompt as string;
    } catch { /* plain text */ }
    return raw;
  })();

  return (
    <>
      {/* ── Backdrop ── */}
      <motion.div
        initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
        className="fixed inset-0 bg-black/60 backdrop-blur-sm z-40"
        onClick={onClose}
      />

      {/* ── Drawer panel ── */}
      <motion.div
        initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
        transition={{ type: "spring", damping: 30, stiffness: 300 }}
        className="fixed bottom-0 left-0 right-0 z-40 max-h-[92vh] overflow-y-auto
                   md:relative md:inset-auto md:max-h-none"
      >
        <div className="glass border border-white/[0.08] m-2 md:m-0 overflow-hidden">
          {/* Top colour stripe — maps to category colour */}
          <div className={cn("h-1 w-full bg-gradient-to-r", cfg.color.replace("/20","/70").replace("/10","/50"))} />

          {/* ── Entry Gate ── */}
          {entryLocked && (
            <div className="p-6 space-y-5 text-center">
              <div className="flex flex-col items-center gap-3">
                <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-amber-500/20 to-orange-500/10 border border-amber-500/30 flex items-center justify-center">
                  <Lock size={28} className="text-amber-400" />
                </div>
                <div>
                  <h3 className="text-white font-bold text-lg">{tool.name}</h3>
                  <p className="text-white/40 text-sm mt-1">Requires minimum balance to unlock</p>
                </div>
              </div>
              <div className="bg-amber-500/8 border border-amber-500/20 rounded-2xl p-4 space-y-3 text-left">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-white/50">Required balance</span>
                  <span className="font-bold text-amber-300">{tool.entry_point_cost.toLocaleString()} pts</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-white/50">Your balance</span>
                  <span className="font-semibold text-red-400">{userPoints.toLocaleString()} pts</span>
                </div>
                <div className="h-px bg-white/10" />
                <div className="flex items-center justify-between text-sm">
                  <span className="text-white/50">You need</span>
                  <span className="font-bold text-red-300">{(tool.entry_point_cost - userPoints).toLocaleString()} more pts</span>
                </div>
                <div className="h-1.5 w-full rounded-full bg-white/10 overflow-hidden">
                  <div
                    className="h-full rounded-full bg-gradient-to-r from-red-500 to-amber-500 transition-all"
                    style={{ width: `${Math.min(99, (userPoints / tool.entry_point_cost) * 100)}%` }}
                  />
                </div>
              </div>
              <p className="text-white/30 text-xs">Top up your PulsePoints to unlock this tool. Your points never expire.</p>
              <div className="flex gap-2">
                <button onClick={onClose} className="flex-1 nexus-btn-outline text-sm py-3">Back</button>
                <Link href="/dashboard"
                  className="flex-1 py-3 rounded-xl text-sm font-semibold flex items-center justify-center gap-2
                             bg-gradient-to-r from-amber-600 to-orange-600 text-white hover:opacity-90">
                  <CreditCard size={15} /> Recharge MTN
                </Link>
              </div>
            </div>
          )}

          {/* ── Main drawer body ── */}
          {!entryLocked && (
            <div className="p-5 space-y-5">

              {/* Header row */}
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3 flex-1 min-w-0">
                  <div className={cn("p-2.5 rounded-xl bg-gradient-to-br flex-shrink-0", cfg.color)}>{cfg.icon}</div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <h3 className="text-white font-bold text-base truncate">{tool.name}</h3>
                      {isNew     && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-purple-500/25 text-purple-300 border border-purple-500/30">NEW</span>}
                      {isFree    && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-green-500/20 text-green-300 border border-green-500/30">FREE</span>}
                      {isPremium && !isFree && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">PREMIUM</span>}
                      {slug === "web-search-ai" && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">🌐 Live</span>}
                      {slug === "video-veo"     && <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-full bg-blue-500/20 text-blue-300 border border-blue-500/30">Veo</span>}
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-white/40 text-xs">{outType.emoji} Outputs 1 {outType.noun}</span>
                      {meta && <span className="text-white/25 text-[10px] flex items-center gap-0.5"><Clock size={9} /> {meta.time}</span>}
                    </div>
                  </div>
                </div>
                <button onClick={onClose} className="text-white/40 hover:text-white/80 transition-colors p-1 flex-shrink-0 ml-2">
                  <X size={18} />
                </button>
              </div>

              {/* Tip box */}
              {meta?.tip && (
                <div className="flex items-start gap-2 border border-amber-500/25 bg-amber-500/8 rounded-xl px-3 py-2.5">
                  <span className="text-amber-400 text-sm flex-shrink-0">💡</span>
                  <p className="text-amber-200/75 text-xs leading-relaxed">
                    <span className="font-semibold">Tip: </span>{meta.tip}
                  </p>
                </div>
              )}

              {/* ── Purpose-built template input ── */}
              {/* Each template handles its own Generate button. When clicked it calls
                  handleTemplateSubmit(payload) which stages the payload and opens
                  the ConfirmModal before any API call is made.                       */}
              <div className="min-h-0">
                {renderTemplate(tool, handleTemplateSubmit, generating, userPoints)}
              </div>

              {/* ── Points summary bar (always visible below template) ── */}
              <div className={cn(
                "rounded-xl border p-3 space-y-1.5",
                isFree     ? "border-green-500/25 bg-green-500/8"
                : canAfford ? "border-gold-500/15 bg-gold-500/5"
                            : "border-red-500/30 bg-red-500/8",
              )}>
                {isFree ? (
                  <div className="flex items-center gap-2 text-green-300 text-sm">
                    <CheckCircle2 size={13} className="flex-shrink-0" />
                    <span className="font-semibold">Free generation — no points used</span>
                  </div>
                ) : (
                  <>
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-white/50 flex items-center gap-1.5">
                        <Zap size={11} className="text-gold-500" /> Generation cost
                      </span>
                      <span className="font-bold text-white">{tool.point_cost} pts per generation</span>
                    </div>
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-white/50">Your balance</span>
                      <span className="font-semibold text-gold-400">{userPoints.toLocaleString()} pts</span>
                    </div>
                    <div className={cn("h-px w-full", canAfford ? "bg-gold-500/15" : "bg-red-500/20")} />
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-white/50">After generation</span>
                      <span className={cn("font-bold", canAfford ? "text-gold-400" : "text-red-400")}>
                        {canAfford
                          ? `${after.toLocaleString()} pts remaining`
                          : `Need ${(tool.point_cost - userPoints).toLocaleString()} more pts`}
                      </span>
                    </div>
                    {!canAfford && (
                      <Link href="/dashboard"
                        className="mt-1 w-full py-2.5 rounded-xl font-semibold flex items-center justify-center gap-2 text-xs
                                   bg-gradient-to-r from-amber-600 to-orange-600 text-white hover:opacity-90 transition-all">
                        <CreditCard size={13} /> Top Up to Continue
                      </Link>
                    )}
                  </>
                )}
              </div>

              {/* Live generating indicator */}
              {generating && genStartedAt && (
                <motion.div
                  initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }}
                  className="flex items-center justify-between bg-gold-500/5 border border-gold-500/15 rounded-xl px-4 py-3"
                >
                  <div className="flex items-center gap-2 text-gold-400 text-sm">
                    <Loader2 size={14} className="animate-spin" />
                    <span>Generating your {outType.noun}…</span>
                  </div>
                  <ElapsedTimer startedAt={genStartedAt} />
                </motion.div>
              )}

              {/* Time estimate row */}
              {meta && !generating && (
                <div className="flex items-center justify-between px-1">
                  <span className="text-white/25 text-[11px] flex items-center gap-1">
                    <Clock size={10} /> Usually ready in {meta.time}
                  </span>
                  <span className="text-white/25 text-[11px]">{outType.emoji} {outType.label}</span>
                </div>
              )}
            </div>
          )}
        </div>
      </motion.div>

      {/* ── Confirm Modal ── */}
      <AnimatePresence>
        {showConfirm && pendingPayload && (
          <ConfirmModal
            tool={tool}
            prompt={confirmPrompt}
            userPoints={userPoints}
            onConfirm={handleConfirmedGenerate}
            onCancel={() => { setShowConfirm(false); setPendingPayload(null); }}
            busy={generating}
          />
        )}
      </AnimatePresence>
    </>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export default function StudioPage() {
  return (
    <Suspense>
      <StudioPageInner />
    </Suspense>
  );
}

function StudioPageInner() {
  const { data: toolsData, isLoading: toolsLoading } = useSWR("/studio/tools",   fetchTools);
  const { data: galleryData, mutate: mutateGallery }  = useSWR("/studio/gallery", fetchGallery, {
    refreshInterval: 15000,
  });
  const user            = useStore((s) => s.user);
  const wallet          = useStore((s) => s.wallet);
  const isAuthenticated = useStore((s) => s.isAuthenticated);
  const userPoints = wallet?.pulse_points ?? 0;

  const tools   = toolsData?.tools   ?? [];
  const gallery = galleryData?.items ?? [];
  const recentGens = gallery.slice(0, 8);

  // Auto-poll gallery every 4s when there are pending/processing items
  useEffect(() => {
    const hasPending = gallery.some((g) => g.status === 'pending' || g.status === 'processing');
    if (!hasPending) return;
    const iv = setInterval(() => mutateGallery(), 4000);
    return () => clearInterval(iv);
  }, [gallery, mutateGallery]);

  const [activeTab,      setActiveTab]      = useState<"chat" | "tools" | "gallery">("tools");
  const [chatMode,        setChatMode]        = useState<ChatMode>('general');
  // ── Per-mode isolated message histories ──────────────────────────────────
  const WELCOME: Record<ChatMode, string> = {
    general: "Hey! 👋 I'm Nexus AI — your personal AI assistant. I can help with business ideas, explain anything, draft content, and more. What's on your mind?",
    search:  "🔍 Web Search is ready. Ask me anything — current news, prices, facts, or real-time data.",
    code:    "💻 Code Helper is ready. Describe what you need or paste code to explain, debug, or improve.",
  };
  const [modeMessages, setModeMessages] = useState<Record<ChatMode, Message[]>>({
    general: [{ role: "assistant", content: WELCOME.general, ts: Date.now() }],
    search:  [{ role: "assistant", content: WELCOME.search,  ts: Date.now(), mode: 'search' }],
    code:    [{ role: "assistant", content: WELCOME.code,    ts: Date.now(), mode: 'code'   }],
  });
  // Convenience alias for the active mode's messages
  const messages    = modeMessages[chatMode];
  const setMessages = (updater: Message[] | ((prev: Message[]) => Message[])) => {
    setModeMessages((prev) => ({
      ...prev,
      [chatMode]: typeof updater === 'function' ? updater(prev[chatMode]) : updater,
    }));
  };
  const [input,          setInput]          = useState("");
  const [sending,        setSending]        = useState(false);
  // Persist session IDs per chat mode for memory continuity
  const getOrCreateSessionId = (mode: ChatMode): string => {
    const key = `nexus_chat_session_${mode}`;
    try {
      const stored = localStorage.getItem(key);
      if (stored) return stored;
    } catch { /* ignore */ }
    const fresh = `sess_${mode}_${Date.now()}_${Math.random().toString(36).slice(2)}`;
    try { localStorage.setItem(key, fresh); } catch { /* ignore */ }
    return fresh;
  };
  // Initialise all three session IDs once on mount
  const sessionIds = useRef<Record<ChatMode, string>>({ general: '', search: '', code: '' });
  const [historyLoaded, setHistoryLoaded] = useState(false);
  // Reset historyLoaded when user logs out so history reloads after next login
  const prevAuthRef = useRef(false);
  useEffect(() => {
    if (prevAuthRef.current && !isAuthenticated) {
      setHistoryLoaded(false);
    }
    prevAuthRef.current = isAuthenticated;
  }, [isAuthenticated]);
  useEffect(() => {
    sessionIds.current = {
      general: getOrCreateSessionId('general'),
      search:  getOrCreateSessionId('search'),
      code:    getOrCreateSessionId('code'),
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // BUG-05 fix: restore chat history for all 3 modes on page load
  // Depends on isAuthenticated so it re-runs after login
  useEffect(() => {
    if (historyLoaded) return;
    if (!isAuthenticated) return;
    const modes: ChatMode[] = ['general', 'search', 'code'];
    Promise.allSettled(
      modes.map((mode) =>
        api.getChatHistory(mode).then((res) => ({ mode, res }))
      )
    ).then((results) => {
      const updates: Partial<Record<ChatMode, Message[]>> = {};
      for (const result of results) {
        if (result.status !== 'fulfilled') continue;
        const { mode, res } = result.value;
        if (!res?.messages?.length) continue;
        const restored: Message[] = res.messages.map((m) => ({
          role: m.role as 'user' | 'assistant',
          content: m.content,
          ts: new Date(m.created_at).getTime(),
          mode,
        }));
        if (res.session_id) {
          sessionIds.current[mode] = res.session_id;
          try { localStorage.setItem(`nexus_chat_session_${mode}`, res.session_id); } catch { /* ignore */ }
        }
        updates[mode] = restored;
      }
      if (Object.keys(updates).length > 0) {
        setModeMessages((prev) => {
          const next = { ...prev };
          for (const [mode, msgs] of Object.entries(updates) as [ChatMode, Message[]][]) {
            next[mode] = msgs;
          }
          return next;
        });
      }
      setHistoryLoaded(true);
    }).catch(() => setHistoryLoaded(true));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, historyLoaded]);

  // Current mode's session ID
  const sessionId = sessionIds.current[chatMode] || `sess_${chatMode}_${Date.now()}`;
  const searchParams    = useSearchParams();
  const [selectedTool,   setSelectedTool]   = useState<Tool | null>(null);
  const [searchQuery,    setSearchQuery]    = useState("");
  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [introDismissed, setIntroDismissed] = useState<boolean>(true);
  const [chatUsage,      setChatUsage]      = useState<{ used: number; limit: number } | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef       = useRef<HTMLInputElement>(null);
  // FEAT-02: Chat attachment state
  const chatFileInputRef = useRef<HTMLInputElement>(null);
  const [chatAttachFile,    setChatAttachFile]    = useState<File | null>(null);
  const [chatAttachURL,     setChatAttachURL]     = useState<string | null>(null);
  const [chatAttachType,    setChatAttachType]    = useState<'image' | 'document' | null>(null);
  const [chatAttachLoading, setChatAttachLoading] = useState(false);
  const [chatAttachError,   setChatAttachError]   = useState<string | null>(null);

  // FEAT-03: Voice-to-text mic recording state
  const [micRecording,    setMicRecording]    = useState(false);
  const [micTranscribing, setMicTranscribing] = useState(false);
  const mediaRecorderRef  = useRef<MediaRecorder | null>(null);
  const audioChunksRef    = useRef<Blob[]>([]);

  const handleMicToggle = useCallback(async () => {
    if (micRecording) {
      // Stop recording
      mediaRecorderRef.current?.stop();
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const recorder = new MediaRecorder(stream);
      audioChunksRef.current = [];
      recorder.ondataavailable = (e) => { if (e.data.size > 0) audioChunksRef.current.push(e.data); };
      recorder.onstop = async () => {
        stream.getTracks().forEach((t) => t.stop());
        setMicRecording(false);
        setMicTranscribing(true);
        try {
          const blob = new Blob(audioChunksRef.current, { type: 'audio/webm' });
          const file = new File([blob], 'voice.webm', { type: 'audio/webm' });
          const uploaded = await api.uploadAsset(file);
          const resp = await api.sendChat(
            `Please transcribe this audio and use it as my message: ${uploaded.url}`,
            sessionIds.current[chatMode] || `sess_${chatMode}_${Date.now()}`,
            undefined, undefined, undefined
          ) as { response: string };
          // Extract transcription from response and put it in the input
          const transcribed = resp.response.replace(/^(transcription:|here is the transcription:|the transcription is:)/i, '').trim();
          setInput((prev) => prev ? `${prev} ${transcribed}` : transcribed);
        } catch {
          toast.error('Voice transcription failed. Please try again.');
        } finally {
          setMicTranscribing(false);
        }
      };
      mediaRecorderRef.current = recorder;
      recorder.start();
      setMicRecording(true);
    } catch {
      toast.error('Microphone access denied. Please allow mic access in your browser.');
    }
  }, [micRecording, chatMode]);

  useEffect(() => {
    try {
      const dismissed = localStorage.getItem("nexus_studio_intro_dismissed");
      setIntroDismissed(dismissed === "true");
    } catch { /* localStorage may not be available */ }
  }, []);

  // Fetch chat usage on mount
  useEffect(() => {
    api.getChatUsage().then((res) => {
      const r = res as { used?: number; limit?: number };
      if (r?.used !== undefined && r?.limit !== undefined) {
        setChatUsage({ used: r.used, limit: r.limit });
      }
    }).catch(() => { /* silent */ });
  }, []);

  // Deep-link: open a specific tool when ?tool=<slug> is in the URL
  useEffect(() => {
    const slugParam = searchParams?.get("tool");
    if (!slugParam || tools.length === 0) return;
    const match = tools.find((t: Tool) => t.slug === slugParam);
    if (match) {
      setSelectedTool(match);
      setActiveTab("tools");
    }
    // Only run once when tools load and slug is present
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tools, searchParams]);

  const handleDismissIntro = useCallback(() => {
    setIntroDismissed(true);
    try { localStorage.setItem("nexus_studio_intro_dismissed", "true"); } catch { /* ignore */ }
  }, []);

  // Canonical tools — excludes alias/duplicate slugs for a cleaner grid
  const canonicalTools = tools.filter((t) => !HIDDEN_ALIAS_SLUGS.has(t.slug));
  const categories    = [...new Set(canonicalTools.map((t) => t.category))];
  const filteredTools = canonicalTools.filter((t) => {
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

  const handleChatAttachSelect = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const isImage = file.type.startsWith('image/');
    const isDoc   = file.type === 'application/pdf' || file.type === 'text/plain' || file.type === 'text/markdown' || file.name.endsWith('.pdf') || file.name.endsWith('.txt') || file.name.endsWith('.md');
    if (!isImage && !isDoc) {
      setChatAttachError('Only images, PDF, and TXT files are supported.');
      return;
    }
    if (file.size > 50 * 1024 * 1024) { setChatAttachError('File must be under 50 MB.'); return; }
    setChatAttachFile(file);
    setChatAttachType(isImage ? 'image' : 'document');
    setChatAttachError(null);
    setChatAttachLoading(true);
    try {
      const result = await api.uploadAsset(file);
      setChatAttachURL(result.url);
    } catch (err: unknown) {
      setChatAttachError(err instanceof Error ? err.message : 'Upload failed');
      setChatAttachFile(null);
      setChatAttachType(null);
    } finally {
      setChatAttachLoading(false);
      if (chatFileInputRef.current) chatFileInputRef.current.value = '';
    }
  }, []);

  const removeChatAttach = useCallback(() => {
    setChatAttachFile(null);
    setChatAttachURL(null);
    setChatAttachType(null);
    setChatAttachError(null);
    if (chatFileInputRef.current) chatFileInputRef.current.value = '';
  }, []);

  const handleChat = useCallback(async () => {
    if ((!input.trim() && !chatAttachURL) || sending) return;
    const msg = input.trim() || (chatAttachType === 'image' ? 'What is in this image?' : 'Analyse this document.');
    const currentMode = chatMode;
    const attachURL  = chatAttachURL;
    const attachType = chatAttachType;
    setInput("");
    setChatAttachFile(null); setChatAttachURL(null); setChatAttachType(null);
    // Show user message with attachment indicator
    const displayContent = attachURL
      ? `${msg}${attachType === 'image' ? ' 📎 [image attached]' : ' 📎 [document attached]'}`
      : msg;
    setMessages((m) => [...m, { role: "user", content: displayContent, ts: Date.now(), mode: currentMode }]);
    setSending(true);
    try {
      // Route to correct tool based on mode
      let toolSlug: string | undefined;
      if (currentMode === 'search') toolSlug = 'web-search-ai';
      if (currentMode === 'code')   toolSlug = 'code-helper';
      const imageURL    = attachType === 'image'    ? (attachURL ?? undefined) : undefined;
      const documentURL = attachType === 'document' ? (attachURL ?? undefined) : undefined;
      const resp = await api.sendChat(msg, sessionId, toolSlug, imageURL, documentURL) as { response: string; provider?: string; session_id?: string; message_count?: number };
      // If backend returns a new session_id, update localStorage for this mode
      if (resp.session_id && resp.session_id !== sessionId) {
        sessionIds.current[currentMode] = resp.session_id;
        try { localStorage.setItem(`nexus_chat_session_${currentMode}`, resp.session_id); } catch { /* ignore */ }
      }
      setMessages((m) => [...m, { role: "assistant", content: resp.response, provider: resp.provider, ts: Date.now(), mode: currentMode }]);
      // Update chat usage counter from response
      if (resp.message_count != null) {
        setChatUsage((prev) => prev ? { ...prev, used: resp.message_count! } : { used: resp.message_count!, limit: 100 });
      } else {
        setChatUsage((prev) => prev ? { ...prev, used: prev.used + 1 } : null);
      }
    } catch {
      setMessages((m) => [...m, {
        role: "assistant",
        content: "I'm having trouble connecting right now. Please try again in a moment. 🔄",
        ts: Date.now(),
        mode: currentMode,
      }]);
    } finally {
      setSending(false);
    }
  }, [input, sending, sessionId, chatMode]);

  const handleClearChat = useCallback(() => {
    // Clear only the current mode's session from localStorage
    const key = `nexus_chat_session_${chatMode}`;
    try { localStorage.removeItem(key); } catch { /* ignore */ }
    // Mint a fresh session ID for this mode
    const fresh = `sess_${chatMode}_${Date.now()}_${Math.random().toString(36).slice(2)}`;
    sessionIds.current[chatMode] = fresh;
    try { localStorage.setItem(key, fresh); } catch { /* ignore */ }
    setMessages([{
      role: "assistant",
      content: WELCOME[chatMode],
      ts: Date.now(),
      mode: chatMode,
    }]);
  }, [chatMode]);

  const handleSummariseChat = useCallback(async () => {
    if (messages.length < 4) return;
    const userMsgs = messages.filter((m) => m.role === 'user').map((m) => m.content);
    const summary = `Please summarise our conversation so far in 3-5 bullet points. Focus on topics covered and decisions made.`;
    setInput(summary);
    inputRef.current?.focus();
  }, [messages]);

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

      <div className="max-w-5xl mx-auto px-4 md:px-6 py-5 space-y-4 pb-24">

        {/* ── Header ── */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-2xl bg-gradient-to-br from-gold-500/80 to-amber-600 flex items-center justify-center shadow-lg shadow-black/40">
              <Brain size={20} className="text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold font-display text-white leading-tight">Nexus AI Studio</h1>
              <p className="text-white/40 text-xs">{canonicalTools.length || tools.length} AI-powered tools</p>
            </div>
          </div>
        </div>

        {/* ── Wallet Bar ── */}
        <WalletBar userPoints={userPoints} />

        {/* ── Session Utilisation Bar ── */}
        <SessionBar userPoints={userPoints} />

        {/* ── How It Works banner ── */}
        <AnimatePresence>
          {!introDismissed && (
            <HowItWorksBanner onDismiss={handleDismissIntro} />
          )}
        </AnimatePresence>

        {/* ── Tab bar ── */}
        <div className="glass border border-white/[0.08] p-1 flex gap-1">
          {([
            { key: "chat",    label: "Chat",    icon: <MessageSquare size={14} />, badge: undefined as number | undefined },
            { key: "tools",   label: "Tools",   icon: <LayoutGrid size={14} />,   badge: (canonicalTools.length || tools.length) as number | undefined },
            { key: "gallery", label: "Gallery", icon: <History size={14} />,      badge: (pendingCount || undefined) as number | undefined },
          ]).map(({ key, label, icon, badge }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key as "chat" | "tools" | "gallery")}
              className={cn(
                "flex-1 py-2.5 px-3 rounded-xl text-xs font-semibold transition-all flex items-center justify-center gap-1.5",
                activeTab === key
                  ? "bg-gradient-to-r from-gold-500/80 to-amber-600 text-white shadow-gold-glow-sm"
                  : "text-white/40 hover:text-white/70 hover:bg-white/5"
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

              {/* ── Chat mode switcher ── */}
              <div className="flex gap-1.5 mb-3">
                {([
                  { id: 'general', label: 'General',    icon: <Brain size={12} />,  color: 'from-gold-500/80 to-amber-600',    badge: 'Free' },
                  { id: 'search',  label: 'Web Search', icon: <Globe size={12} />,  color: 'from-sky-600 to-blue-600',        badge: 'Free' },
                  { id: 'code',    label: 'Code',       icon: <Code2 size={12} />,  color: 'from-green-600 to-emerald-600',   badge: 'Free' },
                ] as const).map((m) => (
                  <button
                    key={m.id}
                    onClick={() => setChatMode(m.id)}
                    className={cn(
                      'flex-1 flex items-center justify-center gap-1.5 py-2 rounded-xl text-xs font-semibold transition-all',
                      chatMode === m.id
                        ? `bg-gradient-to-r ${m.color} text-white shadow-md`
                        : 'bg-white/[0.04] text-white/40 hover:bg-white/[0.08] hover:text-white/70 border border-white/[0.06]',
                    )}
                  >
                    {m.icon} {m.label}
                    {chatMode === m.id && (
                      <span className="text-[9px] bg-white/20 px-1.5 py-0.5 rounded-full">{m.badge}</span>
                    )}
                  </button>
                ))}
              </div>

              {/* Mode description strip */}
              <div className="flex items-center justify-between mb-2 px-1">
                <p className="text-white/30 text-[10px]">
                  {chatMode === 'general' && '🤖 General assistant — business, ideas, content, advice'}
                  {chatMode === 'search'  && '🔍 Live internet — current news, prices, real-time data'}
                  {chatMode === 'code'    && '💻 Qwen Coder — write, explain, debug any programming language'}
                </p>
                {chatUsage && (
                  <span className="text-white/25 text-[10px] flex items-center gap-1">
                    <MessageSquare size={8} />
                    {chatUsage.used}/{chatUsage.limit}
                  </span>
                )}
              </div>

              {/* Messages window */}
              <div className={cn(
                'glass overflow-y-auto p-4 space-y-4 scroll-smooth border',
                'h-[calc(100vh-520px)] min-h-[400px] max-h-[700px]',
                chatMode === 'code'
                  ? 'bg-gray-950/70 border-green-500/10'
                  : chatMode === 'search'
                  ? 'bg-sky-950/25 border-sky-500/10'
                  : 'border-white/[0.08]',
              )}>
                {/* UX-03: History loading skeleton */}
                {!historyLoaded && (
                  <div className="space-y-3 animate-pulse">
                    <div className="flex gap-2.5">
                      <div className="w-8 h-8 rounded-full bg-white/5 flex-shrink-0" />
                      <div className="space-y-1.5 flex-1">
                        <div className="h-3 bg-white/5 rounded-full w-3/4" />
                        <div className="h-3 bg-white/5 rounded-full w-1/2" />
                      </div>
                    </div>
                    <div className="flex gap-2.5 justify-end">
                      <div className="space-y-1.5">
                        <div className="h-3 bg-white/5 rounded-full w-32" />
                      </div>
                      <div className="w-8 h-8 rounded-full bg-white/5 flex-shrink-0" />
                    </div>
                    <div className="flex gap-2.5">
                      <div className="w-8 h-8 rounded-full bg-white/5 flex-shrink-0" />
                      <div className="space-y-1.5 flex-1">
                        <div className="h-3 bg-white/5 rounded-full w-5/6" />
                        <div className="h-3 bg-white/5 rounded-full w-2/3" />
                        <div className="h-3 bg-white/5 rounded-full w-1/3" />
                      </div>
                    </div>
                    <p className="text-white/20 text-[10px] text-center pt-1">Restoring your conversation…</p>
                  </div>
                )}
                {historyLoaded && messages.map((msg, i) => <ChatBubble key={i} msg={msg} />)}
                {/* Suggested prompts — shown only when chat is empty (welcome message only) */}
                {historyLoaded && messages.length <= 1 && !sending && (() => {
                  const SUGGESTIONS: Record<ChatMode, string[]> = {
                    general: [
                      "Write a business plan for a food delivery startup in Lagos",
                      "Explain blockchain in simple terms",
                      "Give me 5 social media post ideas for a fashion brand",
                      "What are the best ways to save money in Nigeria?",
                    ],
                    search: [
                      "What is the current price of Bitcoin today?",
                      "Latest news in Nigeria today",
                      "Current USD to Naira exchange rate",
                      "What are the best smartphones under ₦200,000?",
                    ],
                    code: [
                      "Write a Python function to validate a Nigerian phone number",
                      "Create a REST API endpoint in Node.js for user login",
                      "Explain the difference between async/await and Promises",
                      "Write a SQL query to find duplicate records in a table",
                    ],
                  };
                  const prompts = SUGGESTIONS[chatMode];
                  const modeColor = chatMode === 'code'
                    ? 'border-green-500/25 bg-green-950/20 hover:border-green-500/50 hover:bg-green-950/40'
                    : chatMode === 'search'
                    ? 'border-sky-500/25 bg-sky-950/20 hover:border-sky-500/50 hover:bg-sky-950/40'
                    : 'border-gold-500/20 bg-amber-950/20 hover:border-gold-500/40 hover:bg-amber-950/40';
                  const modeText = chatMode === 'code' ? 'text-green-300/80 hover:text-green-200'
                    : chatMode === 'search' ? 'text-sky-300/80 hover:text-sky-200'
                    : 'text-white/60 hover:text-white/90';
                  return (
                    <div className="mt-4 space-y-2">
                      <p className="text-white/20 text-[10px] uppercase tracking-wider px-1">Try asking…</p>
                      <div className="grid grid-cols-1 gap-1.5">
                        {prompts.map((p, i) => (
                          <button
                            key={i}
                            onClick={() => { setInput(p); inputRef.current?.focus(); }}
                            className={cn(
                              'text-left text-xs px-3 py-2 rounded-xl border transition-all leading-relaxed',
                              modeColor, modeText,
                            )}
                          >
                            {p}
                          </button>
                        ))}
                      </div>
                    </div>
                  );
                })()}
                {sending && (
                  <div className="flex gap-2.5">
                    <div className={cn(
                      'w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0',
                      chatMode === 'code'   ? 'bg-green-600/20' :
                      chatMode === 'search' ? 'bg-sky-600/20' :
                                             'bg-gradient-to-br from-gold-500/15 to-amber-600/10',
                    )}>
                      {chatMode === 'code'   ? <Code2 size={14} className="text-green-300" /> :
                       chatMode === 'search' ? <Globe size={14} className="text-sky-300" /> :
                                              <Brain size={14} className="text-gold-400" />}
                    </div>
                    <div className="glass border border-white/[0.08] px-4 py-2.5 rounded-2xl rounded-tl-sm border border-white/5 flex items-center gap-1.5">
                      <span className="w-1.5 h-1.5 bg-gold-400 rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
                      <span className="w-1.5 h-1.5 bg-gold-400 rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
                      <span className="w-1.5 h-1.5 bg-gold-400 rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>

              {/* Attachment preview pill (FEAT-02) */}
              {chatAttachFile && (
                <div className={cn(
                  'flex items-center gap-2 px-3 py-2 rounded-xl border text-xs mt-2',
                  chatAttachLoading ? 'border-nexus-500/40 bg-nexus-900/30 text-nexus-300'
                  : chatAttachError  ? 'border-red-500/40 bg-red-900/20 text-red-300'
                  :                    'border-green-500/40 bg-green-900/20 text-green-300',
                )}>
                  {chatAttachLoading
                    ? <Loader2 size={12} className="animate-spin flex-shrink-0" />
                    : chatAttachError
                      ? <AlertCircle size={12} className="flex-shrink-0" />
                      : chatAttachType === 'image'
                        ? <ImageIcon size={12} className="flex-shrink-0" />
                        : <FileText size={12} className="flex-shrink-0" />}
                  <span className="truncate flex-1">
                    {chatAttachLoading ? 'Uploading…' : chatAttachError ? chatAttachError : chatAttachFile.name}
                  </span>
                  {!chatAttachLoading && (
                    <button onClick={removeChatAttach} className="flex-shrink-0 hover:text-white transition-colors">
                      <X size={11} />
                    </button>
                  )}
                </div>
              )}

              {/* Input row */}
              <div className="flex gap-2 mt-2">
                {/* Attachment button (FEAT-02) */}
                <button
                  type="button"
                  onClick={() => chatFileInputRef.current?.click()}
                  disabled={sending || chatAttachLoading}
                  title="Attach image or document"
                  className={cn(
                    'px-3 py-2.5 rounded-xl border transition-all flex-shrink-0',
                    chatAttachURL && !chatAttachError
                      ? 'border-green-500/50 bg-green-900/20 text-green-400'
                      : 'border-white/10 text-white/30 hover:border-white/25 hover:text-white/55',
                    (sending || chatAttachLoading) && 'opacity-40 cursor-not-allowed',
                  )}
                >
                  {chatAttachLoading ? <Loader2 size={15} className="animate-spin" /> : <Paperclip size={15} />}
                </button>

                {/* Mic button (FEAT-03) */}
                <button
                  type="button"
                  onClick={handleMicToggle}
                  disabled={sending || chatAttachLoading || micTranscribing}
                  title={micRecording ? 'Stop recording' : 'Voice input'}
                  className={cn(
                    'px-3 py-2.5 rounded-xl border transition-all flex-shrink-0',
                    micRecording
                      ? 'border-red-500/60 bg-red-900/30 text-red-400 animate-pulse'
                      : micTranscribing
                      ? 'border-amber-500/50 bg-amber-900/20 text-amber-400'
                      : 'border-white/10 text-white/30 hover:border-white/25 hover:text-white/55',
                    (sending || chatAttachLoading || micTranscribing) && !micRecording && 'opacity-40 cursor-not-allowed',
                  )}
                >
                  {micTranscribing ? <Loader2 size={15} className="animate-spin" /> : <Mic size={15} />}
                </button>
                <input
                  ref={chatFileInputRef}
                  type="file"
                  accept="image/*,.pdf,.txt,.md,application/pdf,text/plain,text/markdown"
                  onChange={handleChatAttachSelect}
                  className="hidden"
                />
                <input
                  ref={inputRef}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleChat()}
                  placeholder={
                    chatAttachURL ? 'Ask about the attached file… (or press Enter)' :
                    chatMode === 'search' ? 'Search the web — ask about news, prices, facts…' :
                    chatMode === 'code'   ? 'Describe what code you need or paste code to explain…' :
                                           'Ask Nexus anything…'
                  }
                  className={cn(
                    'glass border border-white/[0.10] rounded-xl px-4 py-2.5 text-white placeholder:text-white/30 focus:outline-none flex-1 text-sm transition-colors',
                    chatMode === 'code'   ? 'font-mono focus:border-green-500/50' :
                    chatMode === 'search' ? 'focus:border-sky-500/50' :
                                           'focus:border-gold-500/40',
                  )}
                  disabled={sending}
                />
                <button
                  onClick={handleChat}
                  disabled={sending || (!input.trim() && !chatAttachURL) || chatAttachLoading}
                  className={cn(
                    'px-4 py-3 rounded-xl transition-all',
                    (input.trim() || chatAttachURL) && !sending && !chatAttachLoading
                      ? chatMode === 'code'   ? 'bg-gradient-to-r from-green-600 to-emerald-600 text-white hover:opacity-90 active:scale-95'
                      : chatMode === 'search' ? 'bg-gradient-to-r from-sky-600 to-blue-600 text-white hover:opacity-90 active:scale-95'
                      :                         'bg-gradient-to-r from-gold-500/80 to-amber-600 text-white hover:opacity-90 active:scale-95'
                      : 'bg-white/5 text-white/20 cursor-not-allowed',
                  )}
                >
                  {sending ? <Loader2 size={16} className="animate-spin" /> : <Send size={16} />}
                </button>
              </div>

              {/* Footer actions */}
              <div className="flex items-center justify-between mt-2 px-0.5">
                <button
                  onClick={handleSummariseChat}
                  disabled={messages.length < 4}
                  className="text-white/25 hover:text-white/55 text-[10px] flex items-center gap-1 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                >
                  <Sparkles size={9} /> Summarise chat
                </button>
                <button
                  onClick={handleClearChat}
                  className="text-white/25 hover:text-white/55 text-[10px] flex items-center gap-1 transition-colors"
                >
                  <RotateCcw size={9} /> New chat
                </button>
              </div>
            </motion.div>
          )}

          {/* ── TOOLS ── */}
          {activeTab === "tools" && (
            <motion.div key="tools" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} className="space-y-4">

              {/* Search + category filters */}
              <div className="space-y-2.5">
                <input
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search tools…"
                  className="glass border border-white/[0.10] rounded-xl px-4 py-2.5 text-white placeholder:text-white/30 focus:outline-none focus:border-gold-500/40 text-sm w-full"
                />
                <div className="flex gap-1.5 overflow-x-auto pb-1 scrollbar-hide">
                  <button
                    onClick={() => setActiveCategory(null)}
                    className={cn(
                      "flex-shrink-0 text-xs px-3 py-1.5 rounded-full border transition-all font-medium",
                      !activeCategory
                        ? "bg-gold-500/20 text-gold-400 border-gold-500/40 shadow-gold-glow-sm"
                        : "text-white/50 border-white/10 hover:text-white/80 hover:border-white/25 hover:bg-white/5"
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
                          activeCategory === cat
                            ? cn(cfg.badge, 'shadow-sm')
                            : "text-white/50 border-white/10 hover:text-white/80 hover:border-white/25 hover:bg-white/5"
                        )}
                      >
                        {cfg.icon}
                        {cat.split(" ")[0]}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Per-generation model clarity banner */}
              <div className="flex items-center gap-2.5 glass border border-white/[0.08] p-3 border-gold-500/15">
                <Zap size={13} className="text-gold-500 flex-shrink-0" />
                <p className="text-white/45 text-xs leading-relaxed">
                  <span className="text-white/70 font-semibold">Per-generation pricing:</span>{" "}
                  Points are deducted once each time you click Generate — you get 1 output.
                  If generation fails, points are automatically refunded.
                </p>
              </div>

              {toolsLoading ? (
                <div className="space-y-2">
                  {[...Array(6)].map((_, i) => (
                    <div key={i} className="glass border border-white/[0.08] h-20 animate-pulse opacity-50" />
                  ))}
                </div>
              ) : tools.length === 0 ? (
                <div className="text-center py-16 glass border border-white/[0.08] space-y-4">
                  <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-gold-500/10 to-amber-600/5 border border-white/10 flex items-center justify-center mx-auto">
                    <Sparkles size={28} className="text-gold-500" />
                  </div>
                  <div>
                    <p className="text-white/60 text-base font-semibold">No tools available yet</p>
                    <p className="text-white/30 text-sm mt-1">AI tools will appear here once they&apos;re activated</p>
                  </div>
                  <button
                    onClick={() => setActiveTab("chat")}
                    className="btn-gold text-sm px-5 py-2.5 mx-auto flex items-center gap-1.5"
                  >
                    <MessageSquare size={14} /> Try AI Chat instead
                  </button>
                </div>
              ) : Object.keys(groupedTools).length === 0 ? (
                <div className="text-center py-12 text-white/30 glass border border-white/[0.08] space-y-3">
                  <Wand2 size={32} className="mx-auto mb-3 opacity-40" />
                  <p className="text-sm font-medium">No tools match your search</p>
                  <button
                    onClick={() => { setSearchQuery(""); setActiveCategory(null); }}
                    className="text-gold-500 text-xs hover:text-gold-400 transition-colors underline underline-offset-2"
                  >
                    Clear filters
                  </button>
                </div>
              ) : (
                <>
                {/* ── Popular Tools spotlight ── */}
                {!searchQuery && !activeCategory && (() => {
                  const POPULAR_SLUGS = ["ai-photo","ai-photo-pro","web-search-ai","narrate-pro","song-creator","code-helper","video-veo","business-plan"];
                  const spotlightTools = tools.filter(t => POPULAR_SLUGS.includes(t.slug)).slice(0, 6);
                  if (spotlightTools.length === 0) return null;
                  return (
                    <div className="mb-2">
                      <div className="flex items-center gap-2 mb-3 px-1">
                        <span className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider px-2.5 py-1 rounded-full bg-gold-500/15 text-gold-400 border border-gold-500/25">
                          <Sparkles size={11} /> Popular Tools
                        </span>
                        <span className="text-white/20 text-[10px]">Quick access to the most-used tools</span>
                      </div>
                      <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                        {spotlightTools.map((tool) => (
                          <ToolCard
                            key={tool.id}
                            tool={tool}
                            userPoints={userPoints}
                            onClick={() => {
                              if (tool.slug === "web-search-ai") { setChatMode("search"); setActiveTab("chat"); }
                              else if (tool.slug === "code-helper") { setChatMode("code"); setActiveTab("chat"); }
                              else { setSelectedTool(tool); }
                            }}
                          />
                        ))}
                      </div>
                      <div className="mt-3 border-b border-white/[0.06]" />
                    </div>
                  );
                })()}
                {Object.entries(groupedTools).map(([cat, catTools]) => {
                  const cfg = catCfg(cat);
                  return (
                    <div key={cat}>
                      <div className="flex items-center gap-2 mb-2 px-1">
                        <span className={cn("flex items-center gap-1.5 text-xs font-bold uppercase tracking-wider px-2.5 py-1 rounded-full", cfg.badge)}>
                          {cfg.icon} {cat}
                        </span>
                        <span className="text-white/20 text-[10px]">{catTools.length} tool{catTools.length !== 1 ? "s" : ""}</span>
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                        {catTools.map((tool) => (
                          <ToolCard
                            key={tool.id}
                            tool={tool}
                            userPoints={userPoints}
                            onClick={() => {
                              // Chat tools switch to Chat tab with correct mode
                              if (tool.slug === "web-search-ai") {
                                setChatMode("search");
                                setActiveTab("chat");
                              } else if (tool.slug === "code-helper") {
                                setChatMode("code");
                                setActiveTab("chat");
                              } else {
                                setSelectedTool(tool);
                              }
                            }}
                          />
                        ))}
                      </div>
                    </div>
                  );
                })}
                </>
              )}
            </motion.div>
          )}

          {/* ── GALLERY ── */}
          {activeTab === "gallery" && (
            <motion.div key="gallery" initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} className="space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-white/70 text-sm font-semibold">Your generations</p>
                  <p className="text-white/30 text-[10px]">Points deducted per generation · Failed items auto-refunded</p>
                </div>
                <button onClick={() => mutateGallery()} className="text-white/30 hover:text-white/60 transition-colors p-1">
                  <RefreshCw size={14} />
                </button>
              </div>

              {recentGens.length === 0 ? (
                <div className="text-center py-14 glass border border-white/[0.08] space-y-3">
                  <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-gold-500/10 to-amber-600/5 border border-white/10 flex items-center justify-center mx-auto">
                    <Play size={24} className="text-white/30" />
                  </div>
                  <p className="text-white/40 text-sm font-medium">No generations yet</p>
                  <p className="text-white/25 text-xs">Use a tool to create something amazing</p>
                  <button
                    onClick={() => setActiveTab("tools")}
                    className="btn-gold text-sm px-5 py-2.5 mx-auto flex items-center gap-1.5"
                  >
                    <Wand2 size={14} /> Browse tools
                  </button>
                </div>
              ) : (
                recentGens.map((gen) => (
                  <GenerationCard
                    key={gen.id}
                    gen={gen}
                    onRegenerate={(g) => {
                      const tool = tools.find((t) => t.slug === g.tool_slug);
                      if (tool) { setSelectedTool(tool); setActiveTab("tools"); }
                    }}
                  />
                ))
              )}

              {gallery.length > 8 && (
                <a href="/studio/gallery" className="glass border border-white/[0.10] text-white/70 hover:text-white hover:border-white/20 transition-all rounded-xl font-black w-full py-3 text-sm flex items-center justify-center gap-2">
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
            onGenerated={() => { mutateGallery(); setActiveTab("gallery"); }}
          />
        )}
      </AnimatePresence>
    </AppShell>
  );
}
