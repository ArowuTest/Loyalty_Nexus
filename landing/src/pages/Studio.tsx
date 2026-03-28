import React, { useState } from "react";
import { motion } from "framer-motion";
import { Sparkles, Lock, MessageSquare, Search, ChevronRight } from "lucide-react";
import NavBar from "@/components/NavBar";
import Footer from "@/components/Footer";
import AuthModal from "@/components/AuthModal";
import { AI_TOOLS } from "@/data";
import { ROUTES } from "@/lib";
import type { AITool } from "@/lib";

const CATS = ["all", "chat", "create", "learn", "build"] as const;
type Cat = typeof CATS[number];

const CAT_META: Record<string, { color: string; label: string }> = {
  all:    { color: "#F5A623", label: "All Tools" },
  chat:   { color: "#00D4FF", label: "Chat & Search" },
  create: { color: "#F5A623", label: "Create" },
  learn:  { color: "#10B981", label: "Learn" },
  build:  { color: "#8B5CF6", label: "Build" },
};

function ToolModal({ tool, onClose, onLogin }: { tool: AITool; onClose: () => void; onLogin: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 sm:p-6" onClick={onClose}>
      <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" />
      <motion.div
        initial={{ opacity: 0, y: 40, scale: 0.96 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 20 }}
        transition={{ type: "spring", stiffness: 320, damping: 28 }}
        onClick={e => e.stopPropagation()}
        className="relative glass-strong rounded-3xl p-6 sm:p-8 w-full max-w-md border border-white/[0.10]"
      >
        <div className="text-5xl mb-4">{tool.emoji}</div>
        <h2 className="text-2xl font-black text-foreground mb-2">{tool.name}</h2>
        <p className="text-sm text-muted-foreground leading-relaxed mb-5">{tool.description}</p>
        <div className="flex items-center gap-3 mb-6">
          {tool.is_free
            ? <span className="text-xs font-black px-3 py-1 rounded-full text-cyan-grad" style={{ background: "rgba(0,212,255,0.12)", border: "1px solid rgba(0,212,255,0.25)" }}>FREE — No points needed</span>
            : <span className="text-sm font-bold text-primary font-mono">{tool.point_cost} Pulse Points per use</span>
          }
        </div>
        {tool.is_free ? (
          <button className="btn-gold rounded-xl h-12 w-full text-[15px] font-black inline-flex items-center justify-center gap-2 glow-gold">
            <Sparkles className="w-5 h-5" />
            Launch {tool.name}
          </button>
        ) : (
          <button onClick={onLogin} className="btn-gold rounded-xl h-12 w-full text-[15px] font-black inline-flex items-center justify-center gap-2 glow-gold">
            <Lock className="w-4 h-4" />
            Sign In to Unlock
          </button>
        )}
        <button onClick={onClose} className="mt-3 w-full text-center text-sm text-muted-foreground hover:text-foreground transition-colors py-2">
          Close
        </button>
      </motion.div>
    </div>
  );
}

export default function Studio() {
  const [authOpen, setAuthOpen] = useState(false);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [cat, setCat] = useState<Cat>("all");
  const [selected, setSelected] = useState<AITool | null>(null);
  const [query, setQuery] = useState("");

  const filtered = AI_TOOLS.filter(t => {
    const matchCat = cat === "all" || t.category === cat;
    const matchQ = query === "" || t.name.toLowerCase().includes(query.toLowerCase());
    return matchCat && matchQ;
  });

  return (
    <div className="min-h-screen bg-surface-0 dark">
      <NavBar isLoggedIn={isLoggedIn} onLoginClick={() => setAuthOpen(true)} />
      <AuthModal isOpen={authOpen} onClose={() => setAuthOpen(false)} onSuccess={() => setIsLoggedIn(true)} />

      {selected && (
        <ToolModal tool={selected} onClose={() => setSelected(null)} onLogin={() => { setSelected(null); setAuthOpen(true); }} />
      )}

      {/* Header */}
      <div className="pt-20 pb-10 relative overflow-hidden"
        style={{ background: "radial-gradient(ellipse 100% 60% at 50% 0%, rgba(245,166,35,0.08) 0%, transparent 70%)" }}>
        <div className="max-w-5xl mx-auto px-4 sm:px-6 text-center">
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ type: "spring", stiffness: 260, damping: 26 }}>
            <p className="text-[11px] font-black uppercase tracking-[0.22em] text-primary mb-3">AI Studio</p>
            <h1 className="text-4xl sm:text-5xl font-black tracking-tight text-foreground mb-3">
              30+ AI Tools. <span className="text-gold">One Platform.</span>
            </h1>
            <p className="text-base text-muted-foreground max-w-xl mx-auto mb-7">
              Chat is always <span className="font-bold" style={{ color: "#00D4FF" }}>FREE</span>. 
              Spend Pulse Points to unlock image, video, audio and business tools.
            </p>
            {/* Search */}
            <div className="relative max-w-sm mx-auto">
              <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search tools…"
                value={query}
                onChange={e => setQuery(e.target.value)}
                className="w-full glass rounded-2xl h-11 pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground/50 border border-white/[0.09] focus:border-primary/50 focus:outline-none transition-all"
              />
            </div>
          </motion.div>
        </div>
      </div>

      {/* Category filter */}
      <div className="max-w-5xl mx-auto px-4 sm:px-6 mb-8">
        <div className="flex gap-2 overflow-x-auto no-scrollbar pb-1">
          {CATS.map(c => {
            const m = CAT_META[c];
            const active = cat === c;
            return (
              <button
                key={c}
                onClick={() => setCat(c)}
                className={`flex-shrink-0 h-9 px-4 rounded-xl text-[13px] font-semibold transition-all duration-200 ${
                  active ? "text-black font-black" : "glass border border-white/[0.09] text-muted-foreground hover:text-foreground hover:border-white/[0.18]"
                }`}
                style={active ? { background: `linear-gradient(135deg, ${m.color}dd, ${m.color})` } : {}}
              >
                {m.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tool grid */}
      <div className="max-w-5xl mx-auto px-4 sm:px-6 pb-24">
        {/* FREE banner */}
        {(cat === "all" || cat === "chat") && query === "" && (
          <div className="mb-6">
            <p className="text-[11px] font-black uppercase tracking-[0.18em] mb-3" style={{ color: "#00D4FF" }}>
              Always Free — No Points Needed
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              {AI_TOOLS.filter(t => t.is_free).map(tool => (
                <motion.div
                  key={tool.slug}
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  onClick={() => setSelected(tool)}
                  className="glass-gold rounded-2xl p-5 border border-[rgba(0,212,255,0.18)] cursor-pointer group transition-all duration-200 hover:border-[rgba(0,212,255,0.35)]"
                >
                  <div className="flex items-start justify-between mb-3">
                    <span className="text-3xl">{tool.emoji}</span>
                    <span className="text-[10px] font-black px-2 py-0.5 rounded-full text-cyan-grad" style={{ background: "rgba(0,212,255,0.12)", border: "1px solid rgba(0,212,255,0.25)" }}>
                      FREE
                    </span>
                  </div>
                  <h3 className="font-black text-base text-foreground mb-1">{tool.name}</h3>
                  <p className="text-[12px] text-muted-foreground line-clamp-2 leading-relaxed mb-3">{tool.description}</p>
                  <div className="flex items-center gap-1.5 text-[11px] text-cyan-grad font-semibold">
                    Launch <ChevronRight className="w-3 h-3" />
                  </div>
                </motion.div>
              ))}
            </div>
          </div>
        )}

        {/* Paid tools */}
        <div>
          {(cat === "all" || cat !== "chat") && query === "" && (
            <p className="text-[11px] font-black uppercase tracking-[0.18em] text-muted-foreground/50 mb-3">
              {cat === "all" ? "Premium Tools — Spend Pulse Points" : CAT_META[cat].label}
            </p>
          )}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            {filtered.filter(t => !t.is_free || cat !== "all").map(tool => {
              const catColor = CAT_META[tool.category]?.color ?? "#F5A623";
              return (
                <motion.div
                  key={tool.slug}
                  initial={{ opacity: 0, y: 16 }}
                  animate={{ opacity: 1, y: 0 }}
                  whileHover={{ scale: 1.03 }}
                  whileTap={{ scale: 0.97 }}
                  transition={{ type: "spring", stiffness: 300, damping: 24 }}
                  onClick={() => setSelected(tool)}
                  className="glass rounded-xl p-4 border border-white/[0.07] hover:border-white/[0.18] cursor-pointer transition-all duration-200 flex flex-col group relative overflow-hidden"
                >
                  {/* Category stripe */}
                  <div className="absolute top-0 left-0 right-0 h-0.5" style={{ background: catColor, opacity: 0.4 }} />

                  {tool.is_new && (
                    <span className="absolute top-2.5 right-2.5 text-[9px] font-black px-1.5 py-0.5 rounded-full text-black" style={{ background: catColor }}>NEW</span>
                  )}
                  {tool.is_popular && !tool.is_new && (
                    <span className="absolute top-2.5 right-2.5 text-[9px] font-black px-1.5 py-0.5 rounded-full text-black bg-gold">🔥</span>
                  )}

                  <span className="text-2xl mb-3 block">{tool.emoji}</span>
                  <h4 className="text-[13px] font-bold text-foreground leading-snug mb-1">{tool.name}</h4>
                  <p className="text-[11px] text-muted-foreground line-clamp-2 leading-relaxed flex-1 mb-3">{tool.description}</p>

                  <div className="flex items-center justify-between">
                    <span className="text-[12px] font-bold font-mono" style={{ color: catColor }}>
                      {tool.is_free ? "FREE" : `${tool.point_cost} pts`}
                    </span>
                    {!isLoggedIn && !tool.is_free && (
                      <Lock className="w-3 h-3 text-muted-foreground/40" />
                    )}
                  </div>
                </motion.div>
              );
            })}
          </div>
        </div>
      </div>

      <Footer />
    </div>
  );
}
