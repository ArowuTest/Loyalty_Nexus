'use client';

import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Send, User, Bot, Sparkles, ArrowLeft, Copy, Check, Download, Paperclip, Link2, X, FileText, Globe, Mic, MicOff, Loader2 } from 'lucide-react';
import Link from 'next/link';
import api from '@/lib/api';
import { cn } from '@/lib/utils';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  provider?: string;
  attachedName?: string; // display name of attached file/link for this message
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

// ─── Accepted file types ─────────────────────────────────────────────────────
const ACCEPTED_FILE_TYPES = '.pdf,.txt,.md,.csv,.doc,.docx';
const ACCEPTED_MIME_TYPES = [
  'application/pdf',
  'text/plain',
  'text/markdown',
  'text/csv',
  'application/msword',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
];

// ─── Mic recording states ────────────────────────────────────────────────────
type MicState = 'idle' | 'recording' | 'transcribing' | 'error';

export default function NexusChat() {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      role: 'assistant',
      content: "Hello! I'm **Nexus**, your personal AI assistant. I can help you with business, education, coding, writing, research, and much more.\n\nYou can also **attach a file** (PDF, TXT, DOCX) or **paste a link** (web page or Google Drive) and I'll read it for you.\n\nWhat would you like to work on today?",
      timestamp: new Date(),
    }
  ]);
  const [input,        setInput]        = useState('');
  const [isLoading,    setIsLoading]    = useState(false);
  const [msgCount,     setMsgCount]     = useState(0);
  const [msgLimit,     setMsgLimit]     = useState(20);
  const [copiedMsg,    setCopiedMsg]    = useState<string | null>(null);

  // File attachment state
  const [attachedFile,    setAttachedFile]    = useState<File | null>(null);
  const [attachedFileURL, setAttachedFileURL] = useState<string>('');
  const [isUploading,     setIsUploading]     = useState(false);
  const [uploadError,     setUploadError]     = useState<string>('');

  // Link attachment state
  const [showLinkInput, setShowLinkInput] = useState(false);
  const [linkInput,     setLinkInput]     = useState('');
  const [attachedLink,  setAttachedLink]  = useState<string>('');

  // ─── Mic / voice-to-text state ───────────────────────────────────────────
  const [micState,    setMicState]    = useState<MicState>('idle');
  const [micError,    setMicError]    = useState<string>('');
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const audioChunksRef   = useRef<Blob[]>([]);
  const micStreamRef     = useRef<MediaStream | null>(null);

  const sessionId      = useRef<string>('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef    = useRef<HTMLTextAreaElement>(null);
  const fileInputRef   = useRef<HTMLInputElement>(null);
  const linkInputRef   = useRef<HTMLInputElement>(null);

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

  // Focus link input when shown
  useEffect(() => {
    if (showLinkInput) linkInputRef.current?.focus();
  }, [showLinkInput]);

  // Cleanup mic stream on unmount
  useEffect(() => {
    return () => {
      micStreamRef.current?.getTracks().forEach(t => t.stop());
    };
  }, []);

  // ─── File upload handler ─────────────────────────────────────────────────
  async function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate MIME type
    if (!ACCEPTED_MIME_TYPES.includes(file.type) && !file.name.endsWith('.md') && !file.name.endsWith('.csv')) {
      setUploadError('Unsupported file type. Please upload a PDF, TXT, DOCX, or CSV file.');
      return;
    }
    if (file.size > 20 * 1024 * 1024) {
      setUploadError('File too large. Maximum size is 20 MB.');
      return;
    }

    setUploadError('');
    setAttachedFile(file);
    setAttachedLink(''); // clear any link attachment
    setIsUploading(true);

    try {
      const result = await api.uploadAsset(file);
      setAttachedFileURL(result.url);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Upload failed';
      setUploadError(msg);
      setAttachedFile(null);
      setAttachedFileURL('');
    } finally {
      setIsUploading(false);
      // Reset file input so same file can be re-selected
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  }

  function clearAttachment() {
    setAttachedFile(null);
    setAttachedFileURL('');
    setAttachedLink('');
    setUploadError('');
    if (fileInputRef.current) fileInputRef.current.value = '';
  }

  function confirmLink() {
    const url = linkInput.trim();
    if (!url) return;
    setAttachedLink(url);
    setAttachedFile(null);
    setAttachedFileURL('');
    setLinkInput('');
    setShowLinkInput(false);
  }

  // ─── Mic recording handlers ──────────────────────────────────────────────
  const startRecording = useCallback(async () => {
    setMicError('');
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      micStreamRef.current = stream;
      audioChunksRef.current = [];

      const mimeType = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
        ? 'audio/webm;codecs=opus'
        : MediaRecorder.isTypeSupported('audio/webm')
          ? 'audio/webm'
          : 'audio/ogg';

      const recorder = new MediaRecorder(stream, { mimeType });
      mediaRecorderRef.current = recorder;

      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) audioChunksRef.current.push(e.data);
      };

      recorder.onstop = async () => {
        // Stop all mic tracks
        stream.getTracks().forEach(t => t.stop());
        micStreamRef.current = null;

        const audioBlob = new Blob(audioChunksRef.current, { type: mimeType });
        if (audioBlob.size < 1000) {
          setMicState('idle');
          setMicError('Recording too short — please try again.');
          return;
        }

        setMicState('transcribing');
        try {
          // Upload audio blob to CDN
          const audioFile = new File([audioBlob], `voice_${Date.now()}.webm`, { type: mimeType });
          const { url: audioURL } = await api.uploadAsset(audioFile);

          // Transcribe using the transcribe-african tool (supports Yoruba, Igbo, Hausa, Pidgin, English)
          const genRes = await api.generateBySlug('transcribe-african', {
            prompt: audioURL,
            language: 'en',
          }) as { generation_id: string };

          // Poll for result
          let transcript = '';
          const deadline = Date.now() + 60_000;
          while (Date.now() < deadline) {
            await new Promise(r => setTimeout(r, 1500));
            const status = await api.getGenerationStatus(genRes.generation_id) as {
              status: string; output_text?: string; error_message?: string;
            };
            if (status.status === 'completed') {
              transcript = status.output_text ?? '';
              break;
            }
            if (status.status === 'failed') {
              throw new Error(status.error_message ?? 'Transcription failed');
            }
          }

          if (!transcript) throw new Error('No transcript returned');

          // Append to existing input (in case user had already typed something)
          setInput(prev => prev ? `${prev} ${transcript}` : transcript);
          setMicState('idle');
          // Focus textarea so user can review / edit before sending
          setTimeout(() => textareaRef.current?.focus(), 100);
        } catch (err: unknown) {
          const msg = err instanceof Error ? err.message : 'Transcription failed';
          setMicError(msg);
          setMicState('error');
          setTimeout(() => { setMicState('idle'); setMicError(''); }, 4000);
        }
      };

      recorder.start(250); // collect chunks every 250 ms
      setMicState('recording');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Microphone access denied';
      setMicError(msg.includes('Permission') || msg.includes('denied') || msg.includes('NotAllowed')
        ? 'Microphone permission denied — please allow access in your browser settings.'
        : msg);
      setMicState('error');
      setTimeout(() => { setMicState('idle'); setMicError(''); }, 4000);
    }
  }, []);

  const stopRecording = useCallback(() => {
    if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
      mediaRecorderRef.current.stop();
    }
  }, []);

  function handleMicClick() {
    if (micState === 'recording') {
      stopRecording();
    } else if (micState === 'idle' || micState === 'error') {
      startRecording();
    }
  }

  // ─── Send message ────────────────────────────────────────────────────────
  const handleSend = async (text?: string) => {
    const msg = (text ?? input).trim();
    if (!msg || isLoading) return;
    if (isUploading) return; // wait for upload to finish

    const displayName = attachedFile?.name ?? (attachedLink ? new URL(attachedLink.startsWith('http') ? attachedLink : 'https://' + attachedLink).hostname : undefined);

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: msg,
      timestamp: new Date(),
      attachedName: displayName,
    };

    setMessages(prev => [...prev, userMsg]);
    setInput('');

    // Capture and clear attachment before async call
    const fileURL  = attachedFileURL;
    const linkURL  = attachedLink;
    const fileName = attachedFile?.name ?? '';
    clearAttachment();

    setIsLoading(true);

    try {
      const res = await api.sendChat(
        msg,
        sessionId.current,
        undefined,   // toolSlug — general AI for standalone chat page
        undefined,   // imageURL
        undefined,   // documentURL
        fileURL  || undefined,
        linkURL  || undefined,
        fileName || undefined,
      ) as {
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

  const remaining      = Math.max(0, msgLimit - msgCount);
  const showSuggestions = messages.length === 1;
  const hasAttachment   = !!attachedFile || !!attachedLink;
  const isBusy          = isLoading || isUploading || micState === 'transcribing';

  // Mic button appearance
  const micIsRecording    = micState === 'recording';
  const micIsTranscribing = micState === 'transcribing';
  const micIsError        = micState === 'error';

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
              {/* Attached file/link chip on user messages */}
              {msg.role === 'user' && msg.attachedName && (
                <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl bg-white/8 border border-white/10 text-[10px] text-white/50 self-end mb-1">
                  <FileText size={10} className="text-brand-gold flex-shrink-0" />
                  <span className="truncate max-w-[180px]">{msg.attachedName}</span>
                </div>
              )}
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

        {/* Attachment preview chip */}
        {hasAttachment && (
          <div className="flex items-center gap-2 mb-2 px-1">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-brand-gold/10 border border-brand-gold/25 text-xs text-brand-gold flex-1 min-w-0">
              {attachedFile
                ? <FileText size={12} className="flex-shrink-0" />
                : <Globe size={12} className="flex-shrink-0" />
              }
              <span className="truncate">
                {attachedFile ? attachedFile.name : attachedLink}
              </span>
              {isUploading && (
                <span className="ml-auto text-[10px] text-white/40 flex-shrink-0 animate-pulse">Uploading…</span>
              )}
              {!isUploading && attachedFileURL && (
                <span className="ml-auto text-[10px] text-green-400 flex-shrink-0">✓ Ready</span>
              )}
            </div>
            <button
              onClick={clearAttachment}
              className="p-1.5 rounded-lg text-white/30 hover:text-white/70 hover:bg-white/8 transition-colors flex-shrink-0"
            >
              <X size={12} />
            </button>
          </div>
        )}

        {/* Upload error */}
        {uploadError && (
          <p className="text-[10px] text-red-400 mb-2 px-1">{uploadError}</p>
        )}

        {/* Mic status banners */}
        {micState === 'recording' && (
          <div className="flex items-center gap-2 mb-2 px-3 py-2 rounded-xl bg-red-500/10 border border-red-500/25">
            <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse flex-shrink-0" />
            <span className="text-[11px] text-red-300 font-medium flex-1">Recording… tap mic to stop</span>
          </div>
        )}
        {micState === 'transcribing' && (
          <div className="flex items-center gap-2 mb-2 px-3 py-2 rounded-xl bg-brand-gold/8 border border-brand-gold/20">
            <Loader2 size={12} className="text-brand-gold animate-spin flex-shrink-0" />
            <span className="text-[11px] text-brand-gold/80 font-medium flex-1">Transcribing your voice…</span>
          </div>
        )}
        {micState === 'error' && micError && (
          <div className="flex items-center gap-2 mb-2 px-3 py-2 rounded-xl bg-red-500/10 border border-red-500/20">
            <MicOff size={12} className="text-red-400 flex-shrink-0" />
            <span className="text-[11px] text-red-300 flex-1">{micError}</span>
          </div>
        )}

        {/* Link input popup */}
        {showLinkInput && (
          <div className="flex items-center gap-2 mb-2">
            <input
              ref={linkInputRef}
              type="url"
              placeholder="Paste a web URL or Google Drive link…"
              value={linkInput}
              onChange={e => setLinkInput(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter') confirmLink();
                if (e.key === 'Escape') { setShowLinkInput(false); setLinkInput(''); }
              }}
              className="flex-1 bg-white/5 border border-brand-gold/30 rounded-xl py-2.5 px-4 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/60 transition-all"
            />
            <button
              onClick={confirmLink}
              disabled={!linkInput.trim()}
              className="px-3 py-2.5 rounded-xl text-xs font-bold bg-brand-gold/20 text-brand-gold border border-brand-gold/30 hover:bg-brand-gold/30 transition-colors disabled:opacity-40"
            >
              Attach
            </button>
            <button
              onClick={() => { setShowLinkInput(false); setLinkInput(''); }}
              className="p-2.5 rounded-xl text-white/30 hover:text-white/70 hover:bg-white/8 transition-colors"
            >
              <X size={14} />
            </button>
          </div>
        )}

        {/* Main input row */}
        <div className="relative group flex items-end gap-2">
          {/* Paperclip — file upload */}
          <input
            ref={fileInputRef}
            type="file"
            accept={ACCEPTED_FILE_TYPES}
            onChange={handleFileSelect}
            className="hidden"
          />
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={isBusy || remaining === 0}
            title="Attach a file (PDF, TXT, DOCX)"
            className={cn(
              'w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 mb-0.5 transition-all',
              hasAttachment && attachedFile
                ? 'bg-brand-gold/20 text-brand-gold border border-brand-gold/40'
                : 'bg-white/5 text-slate-500 hover:text-white/70 hover:bg-white/10 border border-transparent',
              (isBusy || remaining === 0) && 'opacity-40 cursor-not-allowed',
            )}
          >
            <Paperclip size={15} />
          </button>

          {/* Link icon — paste a URL */}
          <button
            onClick={() => setShowLinkInput(v => !v)}
            disabled={isBusy || remaining === 0}
            title="Attach a web link or Google Drive URL"
            className={cn(
              'w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 mb-0.5 transition-all',
              (hasAttachment && attachedLink) || showLinkInput
                ? 'bg-brand-gold/20 text-brand-gold border border-brand-gold/40'
                : 'bg-white/5 text-slate-500 hover:text-white/70 hover:bg-white/10 border border-transparent',
              (isBusy || remaining === 0) && 'opacity-40 cursor-not-allowed',
            )}
          >
            <Link2 size={15} />
          </button>

          {/* ── Mic button — voice to text ─────────────────────────────────── */}
          <button
            onClick={handleMicClick}
            disabled={micIsTranscribing || remaining === 0 || isLoading}
            title={micIsRecording ? 'Stop recording' : 'Speak your message'}
            className={cn(
              'w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 mb-0.5 transition-all',
              micIsRecording
                ? 'bg-red-500/20 text-red-400 border border-red-500/40 animate-pulse'
                : micIsTranscribing
                  ? 'bg-brand-gold/10 text-brand-gold/50 border border-brand-gold/20 cursor-wait'
                  : micIsError
                    ? 'bg-red-500/10 text-red-400/60 border border-red-500/20'
                    : 'bg-white/5 text-slate-500 hover:text-white/70 hover:bg-white/10 border border-transparent',
              (micIsTranscribing || remaining === 0 || isLoading) && 'opacity-40 cursor-not-allowed',
            )}
          >
            {micIsTranscribing
              ? <Loader2 size={15} className="animate-spin" />
              : micIsRecording
                ? <MicOff size={15} />
                : <Mic size={15} />
            }
          </button>

          <textarea
            ref={textareaRef}
            rows={1}
            placeholder={
              micIsRecording    ? 'Listening…' :
              micIsTranscribing ? 'Transcribing…' :
              hasAttachment     ? 'Ask about the attached file or link…' :
              'Ask anything… or tap the mic to speak'
            }
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); }
            }}
            disabled={remaining === 0 || micIsRecording || micIsTranscribing}
            className={cn(
              'flex-1 bg-white/5 border border-white/10 rounded-2xl py-3.5 pl-5 pr-4 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 focus:bg-white/10 transition-all resize-none overflow-hidden disabled:opacity-40',
              micIsRecording && 'border-red-500/30 bg-red-500/5',
            )}
          />
          <button
            onClick={() => handleSend()}
            disabled={!input.trim() || isBusy || remaining === 0}
            className={cn(
              'w-11 h-11 rounded-xl flex items-center justify-center transition-all flex-shrink-0 mb-0.5',
              input.trim() && !isBusy && remaining > 0
                ? 'gold-gradient text-black shadow-lg shadow-yellow-500/20 scale-100'
                : 'bg-white/5 text-slate-600 scale-90 opacity-50',
            )}
          >
            <Send size={17} />
          </button>
        </div>

        {/* Supported formats hint */}
        <p className="mt-1.5 text-[9px] text-center font-medium text-slate-700 uppercase tracking-[0.15em]">
          Supports PDF · TXT · DOCX · Web links · Google Drive · Voice
        </p>
        <p className="mt-1 text-[9px] text-center font-bold text-slate-600 uppercase tracking-[0.2em]">
          {remaining > 0
            ? <>Free Daily Limit: <span className="text-brand-gold">{remaining} messages remaining</span></>
            : <span className="text-red-400">Daily limit reached — recharges reset tomorrow</span>}
        </p>
      </footer>
    </div>
  );
}
