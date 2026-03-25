"use client";

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Sparkles, LayoutGrid, MessageSquare, Camera, BookOpen, Hammer, Search } from 'lucide-react';
import { StudioTool, ToolCategory } from '@/types/studio';
import { ToolCard } from './ToolCard';

const CATEGORIES: { name: ToolCategory; icon: any }[] = [
  { name: 'Chat', icon: MessageSquare },
  { name: 'Create', icon: Camera },
  { name: 'Learn', icon: BookOpen },
  { name: 'Build', icon: Hammer },
];

const MOCK_TOOLS: StudioTool[] = [
  {
    id: '1',
    name: 'Ask Nexus',
    description: 'Ultra-fast AI assistant for brainstorming and everyday help.',
    category: 'Chat',
    pointCost: 0,
    iconName: 'MessageSquare',
    isActive: true,
    examplePrompt: 'How can I register my small business in Lagos?'
  },
  {
    id: '2',
    name: 'My AI Photo',
    description: 'Transform text into professional-grade AI portraits and avatars.',
    category: 'Create',
    pointCost: 10,
    iconName: 'Camera',
    isActive: true,
    isNew: true,
    examplePrompt: 'A professional headshot of a confident tech founder, sunset background.'
  },
  {
    id: '3',
    name: 'Marketing Jingle',
    description: 'Create original 30-second music for your brand or business.',
    category: 'Create',
    pointCost: 100,
    iconName: 'Music',
    isActive: true,
    examplePrompt: 'Upbeat Afrobeats style jingle for a restaurant called Mama Gold.'
  },
  {
    id: '4',
    name: 'Study Guide',
    description: 'Generate structured learning materials on any topic instantly.',
    category: 'Learn',
    pointCost: 3,
    iconName: 'BookOpen',
    isActive: true,
    examplePrompt: 'WAEC Chemistry: Redox Reactions summary.'
  },
  {
    id: '5',
    name: 'Business Plan',
    description: 'Turn your idea into a professional one-page summary.',
    category: 'Build',
    pointCost: 5,
    iconName: 'FileText',
    isActive: true,
    examplePrompt: 'A solar panel cleaning service for homes in Abuja.'
  }
];

export default function StudioLanding() {
  const router = useRouter();
  const [activeCategory, setActiveCategory] = useState<ToolCategory | 'All'>('All');
  const [searchQuery, setSearchQuery] = useState('');
  const [userPoints] = useState(12);

  const handleSelectTool = (tool: StudioTool) => {
    if (tool.id === '1') {
      router.push('/studio/chat');
    } else if (tool.id === '2') {
      router.push('/studio/my-ai-photo');
    } else if (tool.id === '5') {
      router.push('/studio/business-plan');
    } else {
      console.log('Selected:', tool.name);
    }
  };

  const filteredTools = MOCK_TOOLS.filter(tool => {
    const matchesCategory = activeCategory === 'All' || tool.category === activeCategory;
    const matchesSearch = tool.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
                          tool.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  return (
    <div className="min-h-screen space-y-10 py-12 px-6 max-w-screen-xl mx-auto bg-black">
      {/* Header Section */}
      <header className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-brand-gold font-black uppercase tracking-[0.2em] text-xs text-brand-gold">
            <Sparkles size={14} /> The Creative Engine
          </div>
          <h1 className="text-5xl font-black text-white italic tracking-tighter">NEXUS STUDIO</h1>
          <p className="text-slate-500 font-medium max-w-md text-slate-500">
            Powering your ambition with telco-scale Generative AI. 
            Exchange your <span className="text-brand-gold">Pulse Points</span> for creative computing power.
          </p>
        </div>

        <div className="glass px-6 py-4 rounded-3xl border border-brand-gold/30 flex items-center gap-4 shadow-xl shadow-brand-gold/5">
          <div className="text-right">
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-slate-500">Available Balance</p>
            <p className="text-2xl font-black text-white italic">{userPoints} <span className="text-brand-gold text-brand-gold">PTS</span></p>
          </div>
          <div className="w-10 h-10 rounded-2xl gold-gradient flex items-center justify-center text-black">
            <Sparkles size={20} />
          </div>
        </div>
      </header>

      {/* Navigation & Search */}
      <div className="flex flex-col md:flex-row gap-4 items-center justify-between border-b border-white/5 pb-6">
        <nav className="flex gap-2 p-1 glass rounded-2xl border border-white/5 overflow-x-auto w-full md:w-auto no-scrollbar">
          <button 
            onClick={() => setActiveCategory('All')}
            className={`px-4 py-2 rounded-xl text-xs font-black uppercase tracking-tighter transition-all whitespace-nowrap
              ${activeCategory === 'All' ? 'bg-white/10 text-white shadow-lg' : 'text-slate-500 hover:text-white text-slate-500'}
            `}
          >
            All Tools
          </button>
          {CATEGORIES.map(cat => (
            <button
              key={cat.name}
              onClick={() => setActiveCategory(cat.name)}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-black uppercase tracking-tighter transition-all whitespace-nowrap
                ${activeCategory === cat.name ? 'bg-white/10 text-white shadow-lg' : 'text-slate-500 hover:text-white text-slate-500'}
              `}
            >
              <cat.icon size={14} />
              {cat.name}
            </button>
          ))}
        </nav>

        <div className="relative w-full md:w-64 group">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-brand-gold transition-colors text-slate-500" size={16} />
          <input 
            type="text" 
            placeholder="Search tools..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-white/5 border border-white/5 rounded-2xl py-3 pl-12 pr-4 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all placeholder:text-slate-600 text-white"
          />
        </div>
      </div>

      {/* Tool Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filteredTools.map(tool => (
          <ToolCard 
            key={tool.id} 
            tool={tool} 
            userPoints={userPoints} 
            onSelect={handleSelectTool}
          />
        ))}
        
        {/* Placeholder for "Coming Soon" impact */}
        <div className="glass rounded-3xl p-5 border border-white/5 opacity-40 flex flex-col justify-center items-center text-center space-y-3 grayscale opacity-40">
          <div className="w-12 h-12 rounded-2xl bg-white/5 flex items-center justify-center text-slate-500 text-slate-500">
            <Sparkles size={24} />
          </div>
          <h3 className="text-lg font-bold text-white italic text-white">More Coming...</h3>
          <p className="text-xs text-slate-500 font-bold uppercase tracking-widest leading-tight text-slate-500">
            New AI tools added weekly
          </p>
        </div>
      </div>
    </div>
  );
}
