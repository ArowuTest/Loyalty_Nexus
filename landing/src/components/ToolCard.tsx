import React from "react";
import { motion } from "framer-motion";
import { Lock, Sparkles } from "lucide-react";
import { hoverLift } from "@/lib/motion";
import type { AITool } from "@/lib";

interface ToolCardProps {
  tool: AITool;
  isLoggedIn?: boolean;
  onClick?: (tool: AITool) => void;
}

export default function ToolCard({ tool, isLoggedIn = false, onClick }: ToolCardProps) {
  const isLocked = !tool.is_free && !isLoggedIn;
  const categoryColors: Record<string, string> = {
    chat:   "oklch(0.68 0.22 190)",
    create: "oklch(0.75 0.20 45)",
    learn:  "oklch(0.65 0.20 150)",
    build:  "oklch(0.62 0.18 280)",
  };
  const glowColor = categoryColors[tool.category] || categoryColors.chat;

  return (
    <motion.div
      variants={hoverLift}
      initial="rest"
      whileHover={isLocked ? undefined : "hover"}
      whileTap={{ scale: 0.97 }}
      onClick={() => onClick?.(tool)}
      className={`
        relative rounded-xl p-4 cursor-pointer transition-all duration-200
        glass-card border border-white/10
        ${isLocked ? "opacity-70" : "hover:border-primary/30"}
        ${!tool.is_free && !isLoggedIn ? "cursor-default" : "cursor-pointer"}
      `}
      style={{
        boxShadow: isLocked ? "none" : `0 0 0 0 ${glowColor}`,
      }}
    >
      {/* New badge */}
      {tool.is_new && (
        <div className="absolute -top-2 -right-2 bg-cyan-glow text-black text-[10px] font-black px-2 py-0.5 rounded-full uppercase tracking-wider">
          NEW
        </div>
      )}
      {/* Popular badge */}
      {tool.is_popular && !tool.is_new && (
        <div className="absolute -top-2 -right-2 bg-gold-gradient text-black text-[10px] font-black px-2 py-0.5 rounded-full uppercase tracking-wider">
          🔥 HOT
        </div>
      )}

      {/* Emoji icon */}
      <div className="text-3xl mb-3">{tool.emoji}</div>

      {/* Name */}
      <h3 className="font-bold text-sm text-foreground mb-1 leading-tight">{tool.name}</h3>

      {/* Description */}
      <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2 mb-3">
        {tool.description}
      </p>

      {/* Bottom row: cost / free badge + lock */}
      <div className="flex items-center justify-between">
        {tool.is_free ? (
          <span className="inline-flex items-center gap-1 text-xs font-bold px-2 py-0.5 rounded-full bg-cyan-glow/15 text-cyan-glow border border-cyan-glow/30">
            <Sparkles className="w-3 h-3" />
            FREE
          </span>
        ) : (
          <span className="text-xs font-mono font-semibold text-primary">
            {tool.point_cost} pts
          </span>
        )}

        {isLocked ? (
          <div className="flex items-center gap-1 text-xs text-muted-foreground/60">
            <Lock className="w-3 h-3" />
            <span>Login</span>
          </div>
        ) : (
          <span className="text-[10px] font-medium text-muted-foreground/50 uppercase tracking-wider">
            {tool.category}
          </span>
        )}
      </div>
    </motion.div>
  );
}
