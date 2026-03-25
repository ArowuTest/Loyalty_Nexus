"use client";

import React from 'react';
import { Lock, Sparkles, ArrowRight, Info } from 'lucide-react';
import * as LucideIcons from 'lucide-react';
import { StudioTool } from '@/types/studio';

interface ToolCardProps {
  tool: StudioTool;
  userPoints: number;
  onSelect: (tool: StudioTool) => void;
}

export const ToolCard: React.FC<ToolCardProps> = ({ tool, userPoints, onSelect }) => {
  const isLocked = userPoints < tool.pointCost;
  const IconComponent = (LucideIcons as any)[tool.iconName] || LucideIcons.Wand2;

  return (
    <div 
      className={`relative group glass rounded-3xl p-5 border transition-all duration-300 flex flex-col h-full
        ${isLocked ? 'border-white/5 opacity-80' : 'border-brand-gold/20 hover:border-brand-gold/50 cursor-pointer hover:shadow-2xl hover:shadow-brand-gold/10'}
      `}
      onClick={() => !isLocked && onSelect(tool)}
    >
      {/* Badge for New Tools */}
      {tool.isNew && (
        <div className="absolute -top-2 -right-2 bg-brand-gold text-black text-[10px] font-black px-2 py-1 rounded-full shadow-lg z-10 animate-pulse">
          NEW
        </div>
      )}

      {/* Header: Icon and Cost */}
      <div className="flex justify-between items-start mb-4">
        <div className={`w-12 h-12 rounded-2xl flex items-center justify-center 
          ${isLocked ? 'bg-white/5 text-slate-500' : 'gold-gradient text-black shadow-lg shadow-yellow-500/20'}
        `}>
          <IconComponent size={24} />
        </div>
        
        <div className={`flex items-center gap-1 px-3 py-1 rounded-full text-[11px] font-black uppercase tracking-tighter
          ${isLocked ? 'bg-red-500/10 text-red-400 border border-red-500/20' : 'bg-brand-gold/10 text-brand-gold border border-brand-gold/20'}
        `}>
          {isLocked && <Lock size={10} />}
          {tool.pointCost} Pulse Points
        </div>
      </div>

      {/* Tool Info */}
      <div className="space-y-2 flex-grow">
        <h3 className="text-lg font-bold text-white group-hover:text-brand-gold transition-colors">
          {tool.name}
        </h3>
        <p className="text-sm text-slate-400 font-medium leading-relaxed line-clamp-2 italic">
          {tool.description}
        </p>
      </div>

      {/* Action Area */}
      <div className="mt-6 pt-4 border-t border-white/5 flex items-center justify-between">
        {isLocked ? (
          <p className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">
            Recharge ₦{Math.ceil((tool.pointCost - userPoints) * 250)} more
          </p>
        ) : (
          <div className="flex items-center gap-2 text-brand-gold text-xs font-black uppercase tracking-widest group-hover:gap-3 transition-all">
            Launch Tool <ArrowRight size={14} />
          </div>
        )}
        
        <button 
          className="p-2 text-slate-500 hover:text-white transition-colors"
          title={`Example: "${tool.examplePrompt}"`}
          onClick={(e) => {
            e.stopPropagation();
            alert(`Example: ${tool.examplePrompt}`);
          }}
        >
          <Info size={16} />
        </button>
      </div>
    </div>
  );
};
