"use client";

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { 
  Sparkles, MessageSquare, Camera, BookOpen, Hammer, Search, 
  Scissors, Video, Music, Clapperboard, HelpCircle, Network, 
  Presentation, PieChart, FileText, Mic2, Languages, Volume2, Mic
} from 'lucide-react';
import { StudioTool, ToolCategory } from '@/types/studio';
import { ToolCard } from './ToolCard';

const CATEGORIES: { name: ToolCategory; icon: any }[] = [
  { name: 'Chat', icon: MessageSquare },
  { name: 'Create', icon: Camera },
  { name: 'Learn', icon: BookOpen },
  { name: 'Build', icon: Hammer },
];

const FULL_CATALOGUE: StudioTool[] = [
  // Chat
  {
    id: 'ask-nexus',
    name: 'Ask Nexus',
    description: 'Ultra-fast AI assistant for brainstorming and everyday help.',
    category: 'Chat',
    pointCost: 0,
    iconName: 'MessageSquare',
    isActive: true,
    examplePrompt: 'How can I register my small business in Lagos?'
  },
  // Create
  {
    id: 'ai-photo',
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
    id: 'bg-remover',
    name: 'Background Remover',
    description: 'Instantly remove backgrounds from your product photos.',
    category: 'Create',
    pointCost: 2,
    iconName: 'Scissors',
    isActive: true,
    examplePrompt: 'Upload a photo of your product to get a transparent PNG.'
  },
  {
    id: 'animate-photo',
    name: 'Animate My Photo',
    description: 'Turn your AI photo into a 5-second living portrait video.',
    category: 'Create',
    pointCost: 65,
    iconName: 'Video',
    isActive: true,
    examplePrompt: 'Select a portrait from your gallery to add motion.'
  },
  {
    id: 'marketing-jingle',
    name: 'Marketing Jingle',
    description: 'Create original 30-second music for your brand.',
    category: 'Create',
    pointCost: 100,
    iconName: 'Music',
    isActive: true,
    examplePrompt: 'Upbeat Afrobeats style jingle for a restaurant called Mama Gold.'
  },
  {
    id: 'video-story',
    name: 'My Video Story',
    description: 'Premium branded video combining AI Photo and custom Jingle.',
    category: 'Create',
    pointCost: 470,
    iconName: 'Clapperboard',
    isActive: true,
    examplePrompt: 'Combine your avatar and jingle into a 15s social media ad.'
  },
  // Learn
  {
    id: 'study-guide',
    name: 'Study Guide',
    description: 'Generate structured learning materials on any topic instantly.',
    category: 'Learn',
    pointCost: 3,
    iconName: 'BookOpen',
    isActive: true,
    examplePrompt: 'WAEC Chemistry: Redox Reactions summary.'
  },
  {
    id: 'quiz-me',
    name: 'Quiz Me',
    description: 'Generate 10 multiple-choice questions to test your knowledge.',
    category: 'Learn',
    pointCost: 2,
    iconName: 'HelpCircle',
    isActive: true,
    examplePrompt: 'Quiz on Nigerian History from 1960 to present.'
  },
  {
    id: 'mind-map',
    name: 'Mind Map',
    description: 'Create a visual mind map from any complex concept.',
    category: 'Learn',
    pointCost: 2,
    iconName: 'Network',
    isActive: true,
    examplePrompt: 'How a POS business operates in Nigeria.'
  },
  {
    id: 'research-brief',
    name: 'Deep Research Brief',
    description: 'Comprehensive multi-angle research coverage on any topic.',
    category: 'Learn',
    pointCost: 3,
    iconName: 'Search',
    isActive: true,
    examplePrompt: 'Opportunities for solar energy SMEs in Northern Nigeria.'
  },
  {
    id: 'my-podcast',
    name: 'My Podcast',
    description: 'Turn any topic into a 5-minute AI-hosted podcast episode.',
    category: 'Learn',
    pointCost: 4,
    iconName: 'Mic',
    isActive: true,
    examplePrompt: 'Topic: The future of 5G in Africa.'
  },
  // Build
  {
    id: 'slide-deck',
    name: 'Slide Deck',
    description: 'Professional PowerPoint presentation generated instantly.',
    category: 'Build',
    pointCost: 4,
    iconName: 'Presentation',
    isActive: true,
    examplePrompt: 'Introduction to my catering business for potential investors.'
  },
  {
    id: 'infographic',
    name: 'Infographic',
    description: 'Visual summary of key facts and complex topics.',
    category: 'Build',
    pointCost: 4,
    iconName: 'PieChart',
    isActive: true,
    examplePrompt: 'Steps to start a small business in Lagos.'
  },
  {
    id: 'business-plan',
    name: 'Business Plan Summary',
    description: 'One-page professional business plan summary.',
    category: 'Build',
    pointCost: 5,
    iconName: 'FileText',
    isActive: true,
    examplePrompt: 'A solar panel cleaning service for homes in Abuja.'
  },
  {
    id: 'voice-to-plan',
    name: 'Voice to Plan',
    description: 'Describe your idea by voice to get a structured plan.',
    category: 'Build',
    pointCost: 6,
    iconName: 'Mic2',
    isActive: true,
    examplePrompt: 'Record: "I want to start a drone photography business for weddings."'
  },
  {
    id: 'local-translation',
    name: 'Local Translation',
    description: 'Translate text to Hausa, Yoruba, Igbo or Pidgin English.',
    category: 'Build',
    pointCost: 2,
    iconName: 'Languages',
    isActive: true,
    examplePrompt: 'Translate my business slogan into Yoruba.'
  },
  {
    id: 'text-to-speech',
    name: 'Text to Speech',
    description: 'Natural audio reading with a professional Nigerian accent.',
    category: 'Build',
    pointCost: 5,
    iconName: 'Volume2',
    isActive: true,
    examplePrompt: 'Read out my marketing script in a confident male voice.'
  }
];

export default function StudioLanding() {
  const router = useRouter();
  const [activeCategory, setActiveCategory] = useState<ToolCategory | 'All'>('All');
  const [searchQuery, setSearchQuery] = useState('');
  const [userPoints] = useState(120); // Demo balance

  const handleSelectTool = (tool: StudioTool) => {
    // Route based on generic logic or specific tool overrides
    if (tool.id === 'ask-nexus') router.push('/studio/chat');
    else if (tool.id === 'ai-photo') router.push('/studio/my-ai-photo');
    else if (tool.id === 'business-plan') router.push('/studio/business-plan');
    else if (tool.id === 'voice-to-plan') router.push('/studio/voice-to-plan');
    else {
      // Generic Tool Interface for the remaining items
      router.push(`/studio/tool/${tool.id}?name=${encodeURIComponent(tool.name)}&cost=${tool.pointCost}`);
    }
  };

  const filteredTools = FULL_CATALOGUE.filter(tool => {
    const matchesCategory = activeCategory === 'All' || tool.category === activeCategory;
    const matchesSearch = tool.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
                          tool.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  return (
    <div className="min-h-screen space-y-10 py-12 px-6 max-w-screen-xl mx-auto bg-black">
      {/* Header Section */}
      <header className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div className="space-y-2 text-left">
          <div className="flex items-center gap-2 text-brand-gold font-black uppercase tracking-[0.2em] text-xs">
            <Sparkles size={14} /> The Creative Engine
          </div>
          <h1 className="text-5xl font-black text-white italic tracking-tighter">NEXUS STUDIO</h1>
          <p className="text-slate-500 font-medium max-w-md">
            Powering your ambition with telco-scale Generative AI. 
            Exchange your <span className="text-brand-gold">Pulse Points</span> for creative computing power.
          </p>
        </div>

        <div className="glass px-6 py-4 rounded-3xl border border-brand-gold/30 flex items-center gap-4 shadow-xl shadow-brand-gold/5">
          <div className="text-right">
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest leading-none">Available Balance</p>
            <p className="text-2xl font-black text-white italic mt-1">{userPoints} <span className="text-brand-gold">PTS</span></p>
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
              ${activeCategory === 'All' ? 'bg-white/10 text-white shadow-lg' : 'text-slate-500 hover:text-white'}
            `}
          >
            All Tools
          </button>
          {CATEGORIES.map(cat => (
            <button
              key={cat.name}
              onClick={() => setActiveCategory(cat.name)}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-black uppercase tracking-tighter transition-all whitespace-nowrap
                ${activeCategory === cat.name ? 'bg-white/10 text-white shadow-lg' : 'text-slate-500 hover:text-white'}
              `}
            >
              <cat.icon size={14} />
              {cat.name}
            </button>
          ))}
        </nav>

        <div className="relative w-full md:w-64 group">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-500 group-focus-within:text-brand-gold transition-colors" size={16} />
          <input 
            type="text" 
            placeholder="Search tools..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-white/5 border border-white/5 rounded-2xl py-3 pl-12 pr-4 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all"
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
      </div>
    </div>
  );
}
