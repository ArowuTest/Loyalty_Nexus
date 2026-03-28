"use client";

import React, { useState, useRef, useEffect } from 'react';
import { Send, User, Bot, Sparkles, ArrowLeft, MoreVertical } from 'lucide-react';
import Link from 'next/link';
import api from '@/lib/api';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  provider?: string;
}

// Session ID persisted in localStorage so the backend can reconstruct memory
function getOrCreateSessionId(): string {
  if (typeof window === 'undefined') return '';
  const key = 'nexus_chat_session';
  const existing = localStorage.getItem(key);
  if (existing) return existing;
  const id = 'sess_' + Date.now().toString(36);
  localStorage.setItem(key, id);
  return id;
}

export default function NexusChat() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      role: 'assistant',
      content: "Hello! I'm Nexus, your personal AI assistant. How can I help you with your business or studies today?",
      timestamp: new Date(),
    }
  ]);
  const [input,     setInput]     = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [msgCount,  setMsgCount]  = useState(0);
  const [msgLimit,  setMsgLimit]  = useState(20);
  const sessionId = useRef<string>('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    sessionId.current = getOrCreateSessionId();
    // load today's usage
    api.getChatUsage().then((res: unknown) => {
      const r = res as { used: number; limit: number };
      setMsgCount(r.used ?? 0);
      setMsgLimit(r.limit ?? 20);
    }).catch(() => {});
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || isLoading) return;

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMsg]);
    setInput('');
    setIsLoading(true);

    try {
      const res = await api.sendChat(input.trim(), sessionId.current) as {
        response: string;
        provider?: string;
        session_id?: string;
        message_count?: number;
      };

      // keep session in sync if backend minted a new one
      if (res.session_id && res.session_id !== sessionId.current) {
        sessionId.current = res.session_id;
        try { localStorage.setItem('nexus_chat_session', res.session_id); } catch { /* ignore */ }
      }
      if (res.message_count !== undefined) setMsgCount(res.message_count);

      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: res.response,
        timestamp: new Date(),
        provider: res.provider?.toUpperCase(),
      }]);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Request failed';
      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: `⚠️ ${msg} — please try again shortly.`,
        timestamp: new Date(),
      }]);
    } finally {
      setIsLoading(false);
    }
  };

  const remaining = Math.max(0, msgLimit - msgCount);

  return (
    <div className="flex flex-col h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5">
      {/* Header */}
      <header className="glass border-b border-brand-gold/20 px-6 py-4 flex items-center justify-between sticky top-0 z-50">
        <div className="flex items-center gap-4">
          <Link href="/studio" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
            <ArrowLeft size={20} />
          </Link>
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-2xl gold-gradient flex items-center justify-center text-black shadow-lg shadow-yellow-500/20">
              <Sparkles size={20} />
            </div>
            <div>
              <h1 className="text-lg font-black tracking-tight italic">ASK NEXUS</h1>
              <div className="flex items-center gap-1.5">
                <div className="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
                <span className="text-[10px] font-black text-slate-500 uppercase tracking-widest">Enterprise Engine Active</span>
              </div>
            </div>
          </div>
        </div>
        <button className="p-2 text-slate-500 hover:text-white transition-colors">
          <MoreVertical size={20} />
        </button>
      </header>

      {/* Chat Canvas */}
      <main className="flex-grow overflow-y-auto p-6 space-y-6 no-scrollbar">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'} animate-in fade-in slide-in-from-bottom-2 duration-300`}
          >
            <div className={`max-w-[85%] flex flex-col ${msg.role === 'user' ? 'items-end' : 'items-start'} space-y-2`}>
              <div className="flex items-center gap-2 mb-1">
                {msg.role === 'assistant' && (
                  <>
                    <div className="w-5 h-5 rounded-lg gold-gradient flex items-center justify-center text-black">
                      <Bot size={12} />
                    </div>
                    <span className="text-[10px] font-black text-brand-gold uppercase tracking-widest">Nexus</span>
                    {msg.provider && (
                      <span className="text-[9px] font-bold text-slate-600 border border-white/5 px-1 rounded uppercase">{msg.provider}</span>
                    )}
                  </>
                )}
                {msg.role === 'user' && (
                  <>
                    <span className="text-[10px] font-black text-slate-500 uppercase tracking-widest">You</span>
                    <div className="w-5 h-5 rounded-lg bg-white/10 flex items-center justify-center text-slate-400">
                      <User size={12} />
                    </div>
                  </>
                )}
              </div>

              <div className={`px-5 py-3.5 rounded-3xl text-sm leading-relaxed font-medium
                ${msg.role === 'user'
                  ? 'bg-brand-gold text-black rounded-tr-none shadow-xl shadow-brand-gold/10'
                  : 'glass border border-white/5 text-slate-200 rounded-tl-none'}
              `}>
                {msg.content}
              </div>

              <span className="text-[9px] font-bold text-slate-600 uppercase">
                {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
              </span>
            </div>
          </div>
        ))}

        {isLoading && (
          <div className="flex justify-start animate-pulse">
            <div className="glass border border-white/5 px-5 py-3 rounded-3xl rounded-tl-none">
              <div className="flex gap-1">
                {[0, 150, 300].map(d => (
                  <div key={d} className="w-1.5 h-1.5 rounded-full bg-brand-gold/50 animate-bounce"
                    style={{ animationDelay: `${d}ms` }} />
                ))}
              </div>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </main>

      {/* Input Hub */}
      <footer className="p-6 pt-2 glass border-t border-white/5 sticky bottom-0">
        <div className="relative group">
          <textarea
            rows={1}
            placeholder="Ask anything…"
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); }
            }}
            className="w-full bg-white/5 border border-white/10 rounded-2xl py-4 pl-5 pr-14 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all resize-none max-h-32"
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || isLoading || remaining === 0}
            className={`absolute right-3 top-1/2 -translate-y-1/2 w-10 h-10 rounded-xl flex items-center justify-center transition-all
              ${input.trim() && !isLoading && remaining > 0
                ? 'gold-gradient text-black shadow-lg shadow-yellow-500/20 scale-100'
                : 'bg-white/5 text-slate-600 scale-90 opacity-50'}
            `}
          >
            <Send size={18} />
          </button>
        </div>
        <p className="mt-3 text-[9px] text-center font-bold text-slate-600 uppercase tracking-[0.2em]">
          {remaining > 0
            ? <>Free Daily Limit: <span className="text-brand-gold">{remaining} messages remaining</span></>
            : <span className="text-red-400">Daily limit reached — recharges reset tomorrow</span>}
        </p>
      </footer>
    </div>
  );
}
