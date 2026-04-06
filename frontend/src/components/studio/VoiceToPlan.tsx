"use client";

import React, { useState, useRef } from 'react';
import { Mic, Sparkles, Send, Download, ArrowLeft, RefreshCw, StopCircle, Upload } from 'lucide-react';
import Link from 'next/link';
import api from '@/lib/api';

type Step = 'idle' | 'uploading' | 'transcribing' | 'generating' | 'done' | 'error';

// Poll generation status until completed or failed
async function pollGeneration(genId: string, maxAttempts = 90): Promise<{ status: string; output_text?: string; output_url?: string }> {
  for (let i = 0; i < maxAttempts; i++) {
    await new Promise(r => setTimeout(r, 3000));
    const s = await api.getGenerationStatus(genId) as { status: string; output_text?: string; output_url?: string };
    if (s.status === 'completed' || s.status === 'failed') return s;
  }
  throw new Error('Timed out waiting for result');
}

export default function VoiceToBusinessPlan() {
  // UI state
  const [step,        setStep]        = useState<Step>('idle');
  const [error,       setError]       = useState('');
  const [audioFile,   setAudioFile]   = useState<File | null>(null);
  const [audioPreview,setAudioPreview]= useState('');
  const [transcript,  setTranscript]  = useState('');
  const [planURL,     setPlanURL]     = useState('');
  const [planText,    setPlanText]    = useState('');
  const [isRecording, setIsRecording] = useState(false);

  // Manual text fallback when no audio
  const [textInput,   setTextInput]   = useState('');

  // MediaRecorder for in-browser recording
  const mediaRef  = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<BlobPart[]>([]);

  // ── Recording ──────────────────────────────────────────────────────────────
  const toggleRecording = async () => {
    if (isRecording) {
      // Stop recording
      mediaRef.current?.stop();
      setIsRecording(false);
      return;
    }

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mr = new MediaRecorder(stream, { mimeType: 'audio/webm' });
      chunksRef.current = [];
      mr.ondataavailable = e => { if (e.data.size > 0) chunksRef.current.push(e.data); };
      mr.onstop = () => {
        stream.getTracks().forEach(t => t.stop());
        const blob = new Blob(chunksRef.current, { type: 'audio/webm' });
        const file = new File([blob], 'voice-note.webm', { type: 'audio/webm' });
        setAudioFile(file);
        setAudioPreview(URL.createObjectURL(blob));
      };
      mr.start();
      mediaRef.current = mr;
      setIsRecording(true);
    } catch {
      setError('Microphone access denied. Please upload an audio file instead.');
    }
  };

  // ── File upload picker ─────────────────────────────────────────────────────
  const handleFilePick = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    setAudioFile(f);
    setAudioPreview(URL.createObjectURL(f));
  };

  // ── Reset ──────────────────────────────────────────────────────────────────
  const reset = () => {
    setStep('idle');
    setError('');
    setAudioFile(null);
    setAudioPreview('');
    setTranscript('');
    setPlanURL('');
    setPlanText('');
    setTextInput('');
  };

  // ── Main pipeline ──────────────────────────────────────────────────────────
  const handleGenerate = async () => {
    if (step !== 'idle') return;
    setError('');

    let description = textInput.trim();

    // Step 1: if we have an audio file, upload it and transcribe it
    if (audioFile && !description) {
      try {
        setStep('uploading');
        const { url: audioURL } = await api.uploadAsset(audioFile);

        setStep('transcribing');
        const tr = await api.generateBySlug('transcribe-african', { prompt: audioURL, language: 'en' }) as { generation_id: string };
        const trResult = await pollGeneration(tr.generation_id);
        if (trResult.status === 'failed') throw new Error('Transcription failed — please check your audio or type your idea instead.');
        description = trResult.output_text ?? '';
        setTranscript(description);
      } catch (e: unknown) {
        setStep('error');
        setError(e instanceof Error ? e.message : 'Transcription failed');
        return;
      }
    }

    if (!description) {
      setError('Please record your voice, upload an audio file, or type your business idea.');
      setStep('idle');
      return;
    }

    // Step 2: generate the business plan
    try {
      setStep('generating');
      const bizPrompt = `Create a comprehensive one-page Nigerian market business plan for the following idea:\n\n${description}\n\nInclude: Executive Summary, Market Opportunity, Revenue Model, Target Customers, Key Risks, and Next Steps. Format clearly with headings.`;
      const biz = await api.generateBySlug('bizplan', { prompt: bizPrompt }) as { generation_id: string };
      const bizResult = await pollGeneration(biz.generation_id);
      if (bizResult.status === 'failed') throw new Error('Business plan generation failed — please try again.');

      setPlanURL(bizResult.output_url ?? '');
      setPlanText(bizResult.output_text ?? '');
      setStep('done');
    } catch (e: unknown) {
      setStep('error');
      setError(e instanceof Error ? e.message : 'Plan generation failed');
    }
  };

  // ── Status labels ──────────────────────────────────────────────────────────
  const statusLabel: Record<Step, string> = {
    idle:         '',
    uploading:    'Uploading audio…',
    transcribing: 'Transcribing with AI…',
    generating:   'Writing your business plan…',
    done:         '',
    error:        '',
  };

  const isProcessing = ['uploading', 'transcribing', 'generating'].includes(step);

  return (
    <div className="min-h-screen bg-black text-white max-w-screen-md mx-auto border-x border-white/5 flex flex-col">
      <header className="glass border-b border-brand-gold/20 px-6 py-4 flex items-center gap-4 sticky top-0 z-50">
        <Link href="/studio" className="p-2 -ml-2 text-slate-400 hover:text-brand-gold transition-colors">
          <ArrowLeft size={20} />
        </Link>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-2xl gold-gradient flex items-center justify-center text-black shadow-lg shadow-yellow-500/20">
            <Mic size={20} />
          </div>
          <div>
            <h1 className="text-lg font-black tracking-tight italic uppercase leading-none">Voice to Plan</h1>
            <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-brand-gold mt-1 italic">
              AI Voice Transcription + Business Plan Generator
            </p>
          </div>
        </div>
      </header>

      <main className="flex-grow p-6 space-y-8 overflow-y-auto no-scrollbar">

        {/* ── Result ── */}
        {step === 'done' && (
          <div className="glass rounded-3xl p-8 border border-brand-gold/30 space-y-6 animate-in zoom-in-95">
            <div className="flex flex-col items-center text-center space-y-3">
              <div className="w-16 h-16 rounded-3xl gold-gradient flex items-center justify-center text-black shadow-2xl">
                <Sparkles size={32} />
              </div>
              <h3 className="text-xl font-black text-white tracking-tight italic">Plan Orchestrated</h3>
              <p className="text-xs text-slate-500 font-bold uppercase tracking-widest leading-relaxed">
                Your idea was transcribed, analysed, and structured into a business plan.
              </p>
            </div>

            {planText && (
              <div className="bg-white/5 rounded-2xl p-5 text-sm text-slate-300 leading-relaxed whitespace-pre-wrap font-medium max-h-96 overflow-y-auto">
                {planText}
              </div>
            )}

            <div className="flex gap-3 justify-center flex-wrap">
              {planURL && (
                <a href={planURL} download target="_blank" rel="noreferrer"
                  className="gold-gradient text-black px-8 py-3.5 rounded-2xl font-black text-sm uppercase tracking-[0.2em] flex items-center gap-2 shadow-xl hover:scale-105 transition-transform">
                  <Download size={16} /> Download PDF
                </a>
              )}
              <button onClick={reset}
                className="border border-white/10 text-slate-400 px-6 py-3.5 rounded-2xl font-black text-sm uppercase tracking-widest hover:text-white transition-colors flex items-center gap-2">
                <RefreshCw size={14} /> Start Over
              </button>
            </div>
          </div>
        )}

        {/* ── Error ── */}
        {step === 'error' && (
          <div className="glass rounded-3xl p-6 border border-red-500/20 space-y-4">
            <p className="text-red-400 font-bold text-sm">⚠ {error}</p>
            <button onClick={reset}
              className="border border-white/10 text-slate-400 px-6 py-3 rounded-2xl font-bold text-sm flex items-center gap-2">
              <RefreshCw size={14} /> Try Again
            </button>
          </div>
        )}

        {/* ── Processing ── */}
        {isProcessing && (
          <div className="flex flex-col items-center justify-center gap-6 py-16">
            <div className="relative">
              <div className="w-20 h-20 rounded-3xl gold-gradient flex items-center justify-center text-black shadow-2xl">
                <Sparkles size={36} className="animate-pulse" />
              </div>
              <div className="absolute inset-0 rounded-3xl ring-4 ring-brand-gold/20 animate-ping" />
            </div>
            <div className="text-center space-y-1">
              <p className="text-white font-black tracking-tight">{statusLabel[step]}</p>
              <p className="text-xs text-slate-500 uppercase tracking-widest">This may take 30–90 seconds</p>
            </div>
          </div>
        )}

        {/* ── Input ── */}
        {step === 'idle' && (
          <div className="space-y-10">

            {/* Voice recorder */}
            <div className="flex flex-col items-center space-y-6">
              <div className={`relative p-1 rounded-full transition-all duration-500 ${isRecording ? 'bg-red-500/20' : 'bg-brand-gold/5'}`}>
                <button onClick={toggleRecording}
                  className={`w-32 h-32 rounded-full flex flex-col items-center justify-center gap-2 transition-all duration-500 border-4
                    ${isRecording
                      ? 'bg-red-500 border-red-400 scale-110 shadow-2xl shadow-red-500/40'
                      : 'gold-gradient border-brand-gold/50 shadow-xl shadow-yellow-500/10 hover:scale-105 active:scale-95 text-black'}
                  `}
                >
                  {isRecording ? <StopCircle size={40} className="text-white" /> : <Mic size={40} />}
                  <span className={`text-[10px] font-black uppercase tracking-widest ${isRecording ? 'text-white' : 'text-black opacity-60'}`}>
                    {isRecording ? 'Stop' : 'Tap to Speak'}
                  </span>
                </button>
              </div>
              <p className="text-xs text-slate-500 font-bold uppercase tracking-widest text-center">
                Describe your business idea in your own words
              </p>
            </div>

            {/* Audio preview (recorded or uploaded) */}
            {audioPreview && (
              <div className="glass rounded-2xl p-4 border border-brand-gold/20 space-y-3">
                <p className="text-[10px] font-black text-brand-gold uppercase tracking-widest">
                  {audioFile?.name ?? 'Recorded audio'}
                </p>
                <audio controls className="w-full" src={audioPreview}>
                  <track kind="captions" />
                </audio>
                <button onClick={() => { setAudioFile(null); setAudioPreview(''); }}
                  className="text-xs text-red-400/70 hover:text-red-400 transition-colors">
                  × Remove
                </button>
              </div>
            )}

            {/* Upload or type */}
            <div className="space-y-4">
              <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest text-center">
                — or —
              </p>

              {/* File upload */}
              <label className="glass rounded-2xl border border-dashed border-white/10 p-5 flex flex-col items-center gap-3 cursor-pointer hover:border-brand-gold/30 transition-colors">
                <Upload size={24} className="text-slate-500" />
                <span className="text-xs font-bold text-slate-400">Upload audio file</span>
                <span className="text-[10px] text-slate-600">MP3, WAV, M4A, WebM · max 20 MB</span>
                <input type="file" className="hidden"
                  accept="audio/mp3,audio/mpeg,audio/wav,audio/m4a,audio/webm,audio/ogg"
                  onChange={handleFilePick} />
              </label>

              {/* Text fallback */}
              <div className="space-y-2">
                <p className="text-[10px] font-black text-slate-500 uppercase tracking-widest">
                  Or type your idea:
                </p>
                <textarea
                  rows={4}
                  value={textInput}
                  onChange={e => setTextInput(e.target.value)}
                  placeholder="e.g. I want to start a mobile laundry service in Lagos that uses an app for scheduling and provides 24-hour delivery…"
                  className="w-full bg-white/5 border border-white/10 rounded-2xl p-4 text-sm text-white placeholder:text-slate-600 focus:outline-none focus:border-brand-gold/30 resize-none"
                />
              </div>
            </div>

            {error && <p className="text-red-400 text-sm font-bold text-center">{error}</p>}

            <button
              onClick={handleGenerate}
              disabled={!audioFile && !textInput.trim()}
              className={`w-full py-4 rounded-2xl font-black text-sm uppercase tracking-[0.2em] flex items-center justify-center gap-3 transition-all
                ${(audioFile || textInput.trim())
                  ? 'gold-gradient text-black shadow-xl hover:scale-105 active:scale-95'
                  : 'bg-white/5 text-slate-600 cursor-not-allowed'}
              `}
            >
              <Send size={18} /> Generate Business Plan
            </button>
          </div>
        )}
      </main>
    </div>
  );
}
