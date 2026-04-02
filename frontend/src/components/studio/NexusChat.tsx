'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Send, User, Bot, Sparkles, ArrowLeft, Copy, Check, Download } from 'lucide-react';
import Link from 'next/link';
import api from '@/lib/api';
import { cn } from '@/lib/utils';

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

// ─── Language colour map for code blocks ────────────────────────────────────
const LANG_COLORS: Record<string, { bg: string; text: string; dot: string }> = {
  python:     { bg: 'bg-blue-500/20',   text: 'text-blue-300',   dot: 'bg-blue-400' },
  javascript: { bg: 'bg-yellow-500/20', text: 'text-yellow-300', dot: 'bg-yellow-400' },
  typescript: { bg: 'bg-blue-600/20',   text: 'text-blue-200',   dot: 'bg-blue-300' },
  js:         { bg: 'bg-yellow-500/20', text: 'text-yellow-300', dot: 'bg-yellow-400' },
  ts:         { bg: 'bg-blue-600/20',   text: 'text-blue-200',   dot: 'bg-blue-300' },
  html:       { bg: 'bg-orange-500/20', text: 'text-orange-300', dot: 'bg-orange-400' },
  css:        { bg: 'bg-sky-500/20',    text: 'text-sky-300',    dot: 'bg-sky-400' },
  sql:        { bg: 'bg-cyan-500/20',   text: 'text-cyan-300',   dot: 'bg-cyan-400' },
  bash:       { bg: 'bg-green-500/20',  text: 'text-green-300',  dot: 'bg-green-400' },
  sh:         { bg: 'bg-green-500/20',  text: 'text-green-300',  dot: 'bg-green-400' },
  go:         { bg: 'bg-teal-500/20',   text: 'text-teal-300',   dot: 'bg-teal-400' },
  rust:       { bg: 'bg-orange-600/20', text: 'text-orange-200', dot: 'bg-orange-300' },
  java:       { bg: 'bg-red-500/20',    text: 'text-red-300',    dot: 'bg-red-400' },
  kotlin:     { bg: 'bg-purple-500/20', text: 'text-purple-300', dot: 'bg-purple-400' },
  swift:      { bg: 'bg-orange-500/20', text: 'text-orange-300', dot: 'bg-orange-400' },
  dart:       { bg: 'bg-sky-600/20',    text: 'text-sky-200',    dot: 'bg-sky-300' },
  json:       { bg: 'bg-amber-500/20',  text: 'text-amber-300',  dot: 'bg-amber-400' },
  yaml:       { bg: 'bg-pink-500/20',   text: 'text-pink-300',   dot: 'bg-pink-400' },
  markdown:   { bg: 'bg-white/10',      text: 'text-white/60',   dot: 'bg-white/40' },
  md:         { bg: 'bg-white/10',      text: 'text-white/60',   dot: 'bg-white/40' },
};
const DEFAULT_LANG_COLOR = { bg: 'bg-white/8', text: 'text-white/40', dot: 'bg-white/30' };

// ─── Language → file extension map ──────────────────────────────────────────
function getExt(lang: string): string {
  const map: Record<string, string> = {
    python: 'py', javascript: 'js', typescript: 'ts', js: 'js', ts: 'ts',
    html: 'html', css: 'css', sql: 'sql', bash: 'sh', sh: 'sh',
    go: 'go', rust: 'rs', java: 'java', kotlin: 'kt', swift: 'swift',
    dart: 'dart', json: 'json', yaml: 'yml', markdown: 'md', md: 'md',
    c: 'c', cpp: 'cpp', 'c++': 'cpp', ruby: 'rb', php: 'php',
    r: 'r', scala: 'scala', haskell: 'hs',
  };
  return map[lang.toLowerCase()] ?? 'txt';
}

// ─── Rich Markdown renderer ──────────────────────────────────────────────────
function RichMessage({ content }: { content: string }) {
  const [copied, setCopied] = useState<number | null>(null);
  const [downloaded, setDownloaded] = useState<number | null>(null);

  function copyCode(text: string, idx: number) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(idx);
      setTimeout(() => setCopied(null), 1800);
    });
  }

  function downloadCode(code: string, lang: string, idx: number) {
    const ext = getExt(lang);
    const blob = new Blob([code], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `code.${ext}`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    setDownloaded(idx);
    setTimeout(() => setDownloaded(null), 2000);
  }

  const parts = content.split(/(```[\s\S]*?```)/g);

  return (
    <div className="space-y-2.5">
      {parts.map((part, i) => {
        if (part.startsWith('```')) {
          const firstNewline = part.indexOf('\n');
          const lang = part.slice(3, firstNewline).trim().toLowerCase() || 'code';
          const code = part.slice(firstNewline + 1, part.lastIndexOf('```')).trim();
          const lc   = LANG_COLORS[lang] ?? DEFAULT_LANG_COLOR;
          const lines = code.split('\n');
          const ext  = getExt(lang);
          return (
            <div key={i} className="rounded-xl overflow-hidden border border-white/[0.12] shadow-lg shadow-black/30">
              {/* VS Code-style header bar */}
              <div className="flex items-center justify-between px-3 py-2 bg-[#0d0d14] border-b border-white/[0.08]">
                <div className="flex items-center gap-2">
                  <div className="flex gap-1">
                    <span className="w-2.5 h-2.5 rounded-full bg-red-500/60" />
                    <span className="w-2.5 h-2.5 rounded-full bg-yellow-500/60" />
                    <span className="w-2.5 h-2.5 rounded-full bg-green-500/60" />
                  </div>
                  <span className={cn('text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider', lc.bg, lc.text)}>
                    <span className={cn('inline-block w-1.5 h-1.5 rounded-full mr-1 align-middle', lc.dot)} />
                    {lang}
                  </span>
                  <span className="text-white/20 text-[10px]">{lines.length} line{lines.length !== 1 ? 's' : ''}</span>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => copyCode(code, i)}
                    className="flex items-center gap-1 text-[10px] text-white/35 hover:text-white/70 transition-colors px-2 py-1 rounded-lg hover:bg-white/[0.06]"
                  >
                    {copied === i ? <Check size={10} className="text-green-400" /> : <Copy size={10} />}
                    {copied === i ? 'Copied!' : 'Copy'}
                  </button>
                  <button
                    onClick={() => downloadCode(code, lang, i)}
                    className="flex items-center gap-1 text-[10px] text-white/35 hover:text-white/70 transition-colors px-2 py-1 rounded-lg hover:bg-white/[0.06]"
                    title={`Download as .${ext}`}
                  >
                    {downloaded === i ? <Check size={10} className="text-green-400" /> : <Download size={10} />}
                    {downloaded === i ? 'Saved!' : `.${ext}`}
                  </button>
                </div>
              </div>
              {/* Code body with line numbers */}
              <div className="bg-[#0a0a10] overflow-x-auto max-h-80 overflow-y-auto">
                <table className="w-full text-[11px] font-mono leading-relaxed">
                  <tbody>
                    {lines.map((line, li) => (
                      <tr key={li} className="hover:bg-white/[0.02]">
                        <td className="select-none text-right pr-4 pl-3 py-0 text-white/20 w-8 border-r border-white/[0.05] align-top pt-0.5">
                          {li + 1}
                        </td>
                        <td className="pl-4 pr-3 py-0 text-green-100/85 whitespace-pre align-top pt-0.5">
                          {line || ' '}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          );
        }

        // Plain text: render inline bold, `code`, headings, bullets, numbered lists
        const lines = part.split('\n');
        return (
          <div key={i} className="space-y-1">
            {lines.map((line, j) => {
              if (!line.trim()) return <div key={j} className="h-1" />;
              if (line.startsWith('### ')) return <p key={j} className="text-white font-bold text-sm mt-2 mb-0.5">{line.slice(4)}</p>;
              if (line.startsWith('## '))  return <p key={j} className="text-white font-bold text-base mt-3 mb-1">{line.slice(3)}</p>;
              if (line.startsWith('# '))   return <p key={j} className="text-white font-bold text-lg mt-3 mb-1">{line.slice(2)}</p>;

              const isBullet   = /^[-*•]\s/.test(line);
              const isNumbered = /^\d+\.\s/.test(line);
              const textContent = isBullet ? line.replace(/^[-*•]\s/, '') : isNumbered ? line.replace(/^\d+\.\s/, '') : line;

              const chunks = textContent.split(/(`[^`]+`|\*\*[^*]+\*\*)/g);
              const rendered = chunks.map((chunk, k) => {
                if (chunk.startsWith('**') && chunk.endsWith('**'))
                  return <strong key={k} className="text-white font-semibold">{chunk.slice(2, -2)}</strong>;
                if (chunk.startsWith('`') && chunk.endsWith('`'))
                  return <code key={k} className="text-[11px] font-mono px-1.5 py-0.5 rounded bg-white/10 text-amber-200">{chunk.slice(1, -1)}</code>;
                return chunk;
              });

              if (isBullet) return (
                <div key={j} className="flex items-start gap-2">
                  <span className="mt-1.5 w-1.5 h-1.5 rounded-full flex-shrink-0 bg-brand-gold" />
                  <p className="text-sm leading-relaxed flex-1 text-white/85">{rendered}</p>
                </div>
              );
              if (isNumbered) {
                const num = line.match(/^(\d+)\./)?.[1];
                return (
                  <div key={j} className="flex items-start gap-2">
                    <span className="flex-shrink-0 w-5 h-5 rounded-full text-[10px] font-bold flex items-center justify-center mt-0.5 bg-gold-500/15 text-gold-400">{num}</span>
                    <p className="text-sm leading-relaxed flex-1 text-white/85">{rendered}</p>
                  </div>
                );
              }
              return <p key={j} className="text-sm leading-relaxed text-white/85">{rendered}</p>;
            })}
          </div>
        );
      })}
    </div>
  );
}

// ─── Quick suggestion chips ──────────────────────────────────────────────────
const SUGGESTIONS = [
  "Write a business plan for a food delivery startup in Lagos",
  "Explain how blockchain works in simple terms",
  "Help me write a professional email to a client",
  "What are the best investment options in Nigeria right now?",
  "Write a Python script to sort a list of names",
  "Summarise the key differences between WAEC and JAMB",
];

export default function NexusChat() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      role: 'assistant',
      content: "Hello! I'm **Nexus**, your personal AI assistant. I can help you with business, education, coding, writing, research, and much more.\n\nWhat would you like to work on today?",
      timestamp: new Date(),
    }
  ]);
  const [input,     setInput]     = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [msgCount,  setMsgCount]  = useState(0);
  const [msgLimit,  setMsgLimit]  = useState(20);
  const [copiedMsg, setCopiedMsg] = useState<string | null>(null);
  const sessionId = useRef<string>('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef    = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    sessionId.current = getOrCreateSessionId();
    api.getChatUsage().then((res: unknown) => {
      const r = res as { used: number; limit: number };
      setMsgCount(r.used ?? 0);
      setMsgLimit(r.limit ?? 20);
    }).catch(() => {});
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Auto-resize textarea
  useEffect(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.style.height = 'auto';
    ta.style.height = Math.min(ta.scrollHeight, 128) + 'px';
  }, [input]);

  const handleSend = async (text?: string) => {
    const msg = (text ?? input).trim();
    if (!msg || isLoading) return;

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: msg,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMsg]);
    setInput('');
    setIsLoading(true);

    try {
      const res = await api.sendChat(msg, sessionId.current) as {
        response: string;
        provider?: string;
        session_id?: string;
        message_count?: number;
      };

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
      const errMsg = err instanceof Error ? err.message : 'Request failed';
      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: `⚠️ ${errMsg} — please try again shortly.`,
        timestamp: new Date(),
      }]);
    } finally {
      setIsLoading(false);
    }
  };

  function copyMessage(content: string, id: string) {
    navigator.clipboard.writeText(content).then(() => {
      setCopiedMsg(id);
      setTimeout(() => setCopiedMsg(null), 1800);
    });
  }

  const remaining = Math.max(0, msgLimit - msgCount);
  const showSuggestions = messages.length === 1;

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
        <div className="text-[10px] text-white/25 font-mono tabular-nums">
          {remaining}/{msgLimit} msgs
        </div>
      </header>

      {/* Chat Canvas */}
      <main className="flex-grow overflow-y-auto p-4 space-y-5 no-scrollbar">

        {/* Quick suggestion chips — only shown before first user message */}
        {showSuggestions && (
          <div className="pt-2 pb-1">
            <p className="text-white/25 text-[11px] uppercase tracking-widest font-semibold mb-3 text-center">Try asking…</p>
            <div className="flex flex-wrap gap-2 justify-center">
              {SUGGESTIONS.map((s, i) => (
                <button
                  key={i}
                  onClick={() => handleSend(s)}
                  className="text-xs px-3 py-2 rounded-xl border border-white/10 text-white/50 hover:border-brand-gold/40 hover:text-white/80 hover:bg-brand-gold/5 transition-all text-left max-w-[280px] leading-snug"
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
        )}

        {messages.map((msg) => (
          <div
            key={msg.id}
            className={cn(
              'flex gap-3 group',
              msg.role === 'user' && 'flex-row-reverse',
            )}
          >
            {/* Avatar */}
            <div className={cn(
              'w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5',
              msg.role === 'user'
                ? 'bg-gradient-to-br from-gold-500/20 to-amber-600/15'
                : 'bg-gradient-to-br from-gold-500/15 to-amber-600/10',
            )}>
              {msg.role === 'user'
                ? <User size={14} className="text-purple-300" />
                : <Bot size={14} className="text-brand-gold" />
              }
            </div>

            {/* Bubble */}
            <div className={cn('max-w-[82%] space-y-1', msg.role === 'user' && 'items-end flex flex-col')}>
              <div className={cn(
                'px-4 py-3 rounded-2xl',
                msg.role === 'user'
                  ? 'bg-gradient-to-br from-gold-500/80 to-amber-600 text-white rounded-tr-sm text-sm leading-relaxed'
                  : 'bg-[#1c1e2e] rounded-tl-sm border border-white/[0.07] shadow-sm',
              )}>
                {msg.role === 'user'
                  ? <p className="text-sm leading-relaxed">{msg.content}</p>
                  : <RichMessage content={msg.content} />
                }
              </div>

              {/* Meta row */}
              <div className={cn(
                'flex items-center gap-2 px-1 opacity-0 group-hover:opacity-100 transition-opacity',
                msg.role === 'user' && 'flex-row-reverse',
              )}>
                <span className="text-white/20 text-[9px] tabular-nums">
                  {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </span>
                {msg.role === 'assistant' && msg.provider && (
                  <span className="text-[9px] font-bold text-white/20 border border-white/10 px-1.5 py-0.5 rounded uppercase">{msg.provider}</span>
                )}
                {/* Copy button — appears on hover */}
                <button
                  onClick={() => copyMessage(msg.content, msg.id)}
                  className="flex items-center gap-1 text-[10px] text-white/25 hover:text-white/60 transition-colors px-1.5 py-0.5 rounded"
                >
                  {copiedMsg === msg.id ? <Check size={9} className="text-green-400" /> : <Copy size={9} />}
                  {copiedMsg === msg.id ? 'Copied' : 'Copy'}
                </button>
              </div>
            </div>
          </div>
        ))}

        {/* Typing indicator */}
        {isLoading && (
          <div className="flex gap-3 justify-start">
            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-gold-500/15 to-amber-600/10 flex items-center justify-center flex-shrink-0">
              <Bot size={14} className="text-brand-gold" />
            </div>
            <div className="bg-[#1c1e2e] border border-white/[0.07] px-5 py-3.5 rounded-2xl rounded-tl-sm">
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
      <footer className="p-4 pt-2 glass border-t border-white/5 sticky bottom-0">
        <div className="relative group flex items-end gap-2">
          <textarea
            ref={textareaRef}
            rows={1}
            placeholder="Ask anything…"
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); }
            }}
            disabled={remaining === 0}
            className="flex-1 bg-white/5 border border-white/10 rounded-2xl py-3.5 pl-5 pr-4 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all resize-none overflow-hidden disabled:opacity-40"
          />
          <button
            onClick={() => handleSend()}
            disabled={!input.trim() || isLoading || remaining === 0}
            className={cn(
              'w-11 h-11 rounded-xl flex items-center justify-center transition-all flex-shrink-0 mb-0.5',
              input.trim() && !isLoading && remaining > 0
                ? 'gold-gradient text-black shadow-lg shadow-yellow-500/20 scale-100'
                : 'bg-white/5 text-slate-600 scale-90 opacity-50',
            )}
          >
            <Send size={17} />
          </button>
        </div>
        <p className="mt-2 text-[9px] text-center font-bold text-slate-600 uppercase tracking-[0.2em]">
          {remaining > 0
            ? <>Free Daily Limit: <span className="text-brand-gold">{remaining} messages remaining</span></>
            : <span className="text-red-400">Daily limit reached — recharges reset tomorrow</span>}
        </p>
      </footer>
    </div>
  );
}
