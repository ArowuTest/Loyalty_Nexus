'use client';

import { useState, useRef } from 'react';
import { Loader2, Sparkles, ArrowRight, Paperclip, X, FileText, AlertCircle } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

// Default fallback for tools with no ui_config.fields
const DEFAULT_FIELDS = [
  {
    key: 'prompt', label: 'Describe what you want', type: 'textarea' as const,
    required: true, placeholder: 'Provide details about what you\'d like to generate…',
    rows: 5, default: '',
  },
];

// ─── Translate-specific language list ────────────────────────────────────────
const TRANSLATE_LANGUAGES = [
  { code: 'en',    label: 'English' },
  { code: 'fr',    label: 'French' },
  { code: 'es',    label: 'Spanish' },
  { code: 'pt',    label: 'Portuguese' },
  { code: 'de',    label: 'German' },
  { code: 'ar',    label: 'Arabic' },
  { code: 'zh',    label: 'Chinese' },
  { code: 'sw',    label: 'Swahili' },
  { code: 'yo',    label: 'Yoruba' },
  { code: 'ha',    label: 'Hausa' },
  { code: 'ig',    label: 'Igbo' },
  { code: 'pcm',   label: 'Nigerian Pidgin' },
  { code: 'af',    label: 'Afrikaans' },
];

// ─── Slugs that support document upload (FEAT-01) ─────────────────────────────
const DOCUMENT_UPLOAD_SLUGS = new Set([
  'study-guide', 'quiz', 'mindmap', 'research-brief',
  'bizplan', 'slide-deck', 'infographic', 'podcast',
]);

// ─── Translate layout ─────────────────────────────────────────────────────────
function TranslateLayout({
  tool, onSubmit, isLoading, canAfford,
}: { tool: TemplateProps['tool']; onSubmit: TemplateProps['onSubmit']; isLoading: boolean; canAfford: boolean }) {
  const cfg = tool.ui_config ?? {};
  const languages = cfg.translate_languages ?? TRANSLATE_LANGUAGES;

  const [text,       setText]       = useState('');
  const [sourceLang, setSourceLang] = useState('auto');
  const [targetLang, setTargetLang] = useState('en');

  const isValid = text.trim().length >= 3 && targetLang !== '';

  function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;
    const srcLabel = sourceLang === 'auto' ? 'Auto-detect' : (languages as { code: string; label: string }[]).find((l) => l.code === sourceLang)?.label ?? sourceLang;
    const tgtLabel = (languages as { code: string; label: string }[]).find((l) => l.code === targetLang)?.label ?? targetLang;
    const payload: GeneratePayload = {
      prompt: `Translate the following text from ${srcLabel} to ${tgtLabel}:\n\n${text.trim()}`,
      language: targetLang,
      extra_params: {
        source_language: sourceLang,
        target_language: targetLang,
        original_text: text.trim(),
      },
    };
    onSubmit(payload);
  }

  return (
    <div className="space-y-5">
      {/* Language pair */}
      <div className="flex items-center gap-2">
        <div className="flex-1">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">From</label>
          <select
            value={sourceLang}
            onChange={(e) => setSourceLang(e.target.value)}
            className="nexus-input w-full text-sm"
          >
            <option value="auto">Auto-detect</option>
            {(languages as { code: string; label: string }[]).map((l) => (
              <option key={l.code} value={l.code}>{l.label}</option>
            ))}
          </select>
        </div>
        <div className="flex-shrink-0 mt-5">
          <ArrowRight size={16} className="text-white/25" />
        </div>
        <div className="flex-1">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">To</label>
          <select
            value={targetLang}
            onChange={(e) => setTargetLang(e.target.value)}
            className="nexus-input w-full text-sm"
          >
            {(languages as { code: string; label: string }[]).map((l) => (
              <option key={l.code} value={l.code}>{l.label}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Quick language targets */}
      <div>
        <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-2 block">Quick select target</label>
        <div className="flex flex-wrap gap-1.5">
          {(['English', 'French', 'Yoruba', 'Hausa', 'Igbo', 'Spanish', 'Portuguese', 'Arabic', 'Swahili'] as const).map((label) => {
            const lang = (languages as { code: string; label: string }[]).find((l) => l.label === label);
            if (!lang) return null;
            return (
              <button
                key={lang.code}
                onClick={() => setTargetLang(lang.code)}
                className={cn(
                  'text-xs px-3 py-1.5 rounded-full border font-medium transition-all',
                  targetLang === lang.code
                    ? 'bg-nexus-600 text-white border-nexus-500'
                    : 'text-white/55 border-white/15 hover:border-white/30 hover:text-white/80',
                )}
              >
                {label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Text input */}
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">Text to translate</label>
          <span className="text-white/25 text-[11px]">{text.length}/5000</span>
        </div>
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={cfg.prompt_placeholder ?? 'Paste or type the text you want to translate…'}
          rows={5}
          autoFocus
          className="nexus-input resize-none w-full text-sm leading-relaxed"
        />
      </div>

      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && !isLoading && canAfford
            ? 'bg-gradient-to-r from-nexus-600 to-purple-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Translating…</>
          : <><ArrowRight size={15} /> Translate →</>
        }
      </button>
    </div>
  );
}

// ─── Main KnowledgeDoc component ─────────────────────────────────────────────
export default function KnowledgeDoc({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg    = tool.ui_config ?? {};
  const fields = cfg.fields?.length ? cfg.fields : DEFAULT_FIELDS;

  const canAfford = tool.is_free || userPoints >= tool.point_cost;
  const supportsDocUpload = DOCUMENT_UPLOAD_SLUGS.has(tool.slug);

  // ── Document upload state (FEAT-01) ──────────────────────────────────────
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [docFile,       setDocFile]       = useState<File | null>(null);
  const [docURL,        setDocURL]        = useState<string | null>(null);
  const [docUploading,  setDocUploading]  = useState(false);
  const [docUploadErr,  setDocUploadErr]  = useState<string | null>(null);

  // If this is the translate tool, render dedicated layout
  if (tool.slug === 'translate') {
    return <TranslateLayout tool={tool} onSubmit={onSubmit} isLoading={isLoading} canAfford={canAfford} />;
  }

  // ── Generic field-driven layout ──────────────────────────────────────────

  type FieldDef = {
    key: string;
    label: string;
    type: 'textarea' | 'select' | 'text';
    required?: boolean;
    placeholder?: string;
    rows?: number;
    options?: string[];
    default?: string;
  };

  const typedFields = fields as FieldDef[];

  const [values, setValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    typedFields.forEach((f) => { init[f.key] = f.default ?? ''; });
    return init;
  });

  function isValid(): boolean {
    // If a document is uploaded, the prompt field is optional (document provides context)
    if (docURL) return true;
    return typedFields.every((f) => !f.required || (values[f.key]?.trim().length ?? 0) >= 3);
  }

  async function handleDocumentSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    const allowed = ['application/pdf', 'text/plain', 'text/markdown'];
    const ext = file.name.toLowerCase();
    const isAllowed = allowed.includes(file.type) || ext.endsWith('.pdf') || ext.endsWith('.txt') || ext.endsWith('.md');
    if (!isAllowed) {
      setDocUploadErr('Only PDF, TXT, and Markdown files are supported.');
      return;
    }
    if (file.size > 50 * 1024 * 1024) {
      setDocUploadErr('File must be under 50 MB.');
      return;
    }

    setDocFile(file);
    setDocUploadErr(null);
    setDocUploading(true);
    try {
      const result = await api.uploadAsset(file);
      setDocURL(result.url);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Upload failed';
      setDocUploadErr(msg);
      setDocFile(null);
    } finally {
      setDocUploading(false);
    }
  }

  function removeDocument() {
    setDocFile(null);
    setDocURL(null);
    setDocUploadErr(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  }

  function handleSubmit() {
    if (!isValid() || isLoading || !canAfford || docUploading) return;
    const parts = typedFields
      .filter((f) => values[f.key]?.trim())
      .map((f) => `${f.label}: ${values[f.key].trim()}`);
    const promptText = parts.join('\n') || (docURL ? 'Analyse the uploaded document and generate the output.' : '');
    const payload: GeneratePayload = {
      prompt: promptText,
      ...(docURL ? { document_url: docURL } : {}),
      extra_params: {
        ...values,
        output_format: cfg.output_format,
      },
    };
    onSubmit(payload);
  }

  function setValue(key: string, val: string) {
    setValues((prev) => ({ ...prev, [key]: val }));
  }

  // Button label based on output type
  const btnLabel =
    cfg.output_format === 'document' ? 'Generate Document →'
    : cfg.output_format === 'audio'  ? 'Generate Audio →'
    : 'Generate →';

  return (
    <div className="space-y-5">

      {/* ── Document upload zone (FEAT-01) — only for knowledge tools ── */}
      {supportsDocUpload && (
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold">
              Upload Document <span className="text-white/30 font-normal normal-case">(optional)</span>
            </label>
            <span className="text-white/25 text-[10px]">PDF, TXT, MD · max 50 MB</span>
          </div>

          {docFile ? (
            /* Uploaded file pill */
            <div className={cn(
              'flex items-center gap-3 px-4 py-3 rounded-xl border',
              docUploading
                ? 'border-nexus-500/40 bg-nexus-900/30'
                : docUploadErr
                  ? 'border-red-500/40 bg-red-900/20'
                  : 'border-green-500/40 bg-green-900/20',
            )}>
              {docUploading ? (
                <Loader2 size={16} className="animate-spin text-nexus-400 flex-shrink-0" />
              ) : docUploadErr ? (
                <AlertCircle size={16} className="text-red-400 flex-shrink-0" />
              ) : (
                <FileText size={16} className="text-green-400 flex-shrink-0" />
              )}
              <div className="flex-1 min-w-0">
                <p className="text-white/80 text-sm font-medium truncate">{docFile.name}</p>
                <p className="text-white/35 text-[11px]">
                  {docUploading ? 'Uploading…' : docUploadErr ? docUploadErr : `${(docFile.size / 1024).toFixed(0)} KB · Ready`}
                </p>
              </div>
              {!docUploading && (
                <button
                  onClick={removeDocument}
                  className="text-white/30 hover:text-white/70 transition-colors flex-shrink-0"
                  title="Remove document"
                >
                  <X size={15} />
                </button>
              )}
            </div>
          ) : (
            /* Upload drop zone */
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="w-full flex items-center gap-3 px-4 py-3.5 rounded-xl border border-dashed border-white/15 hover:border-nexus-500/50 hover:bg-nexus-900/20 transition-all text-left group"
            >
              <Paperclip size={16} className="text-white/30 group-hover:text-nexus-400 transition-colors flex-shrink-0" />
              <div>
                <p className="text-white/45 text-sm group-hover:text-white/65 transition-colors">
                  Attach a document for AI to analyse
                </p>
                <p className="text-white/25 text-[11px]">PDF, plain text, or Markdown</p>
              </div>
            </button>
          )}

          <input
            ref={fileInputRef}
            type="file"
            accept=".pdf,.txt,.md,application/pdf,text/plain,text/markdown"
            onChange={handleDocumentSelect}
            className="hidden"
          />

          {docURL && !docUploadErr && (
            <p className="text-green-400/70 text-[11px] mt-1.5 flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-green-400 inline-block" />
              Document uploaded — AI will analyse it as the primary source
            </p>
          )}
        </div>
      )}

      {/* ── Dynamic fields ── */}
      {typedFields.map((field, idx) => (
        <div key={field.key}>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
            {field.label}
            {field.required && !docURL && <span className="text-red-400 ml-1">*</span>}
            {field.required && docURL && <span className="text-white/25 text-[10px] ml-1 font-normal">(optional with document)</span>}
          </label>

          {field.type === 'textarea' ? (
            <>
              <textarea
                value={values[field.key]}
                onChange={(e) => setValue(field.key, e.target.value)}
                placeholder={docURL ? (field.placeholder ?? 'Add additional instructions (optional)…') : field.placeholder}
                rows={field.rows ?? 4}
                autoFocus={idx === 0}
                className="nexus-input resize-none w-full text-sm leading-relaxed"
              />
              <p className="text-white/25 text-[11px] mt-1">
                {values[field.key]?.length ?? 0}/1000 characters
              </p>
            </>
          ) : field.type === 'select' ? (
            <select
              value={values[field.key]}
              onChange={(e) => setValue(field.key, e.target.value)}
              className="nexus-input w-full text-sm"
            >
              {!field.required && <option value="">Choose an option…</option>}
              {(field.options ?? []).map((opt) => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
          ) : (
            <input
              type="text"
              value={values[field.key]}
              onChange={(e) => setValue(field.key, e.target.value)}
              placeholder={field.placeholder}
              autoFocus={idx === 0}
              className="nexus-input w-full text-sm"
            />
          )}
        </div>
      ))}

      {/* ── Output format badge ── */}
      {cfg.output_format && (
        <div className="flex items-center gap-2">
          <span className="text-white/35 text-xs">Output:</span>
          <span className={cn(
            'text-xs px-2.5 py-1 rounded-full font-semibold',
            cfg.output_format === 'document' ? 'bg-blue-500/20 text-blue-300 border border-blue-500/30'
            : cfg.output_format === 'audio'  ? 'bg-green-500/20 text-green-300 border border-green-500/30'
            :                                  'bg-white/10 text-white/50 border border-white/15',
          )}>
            {cfg.output_format === 'document' ? '📄 Document'
              : cfg.output_format === 'audio' ? '🎙 Audio'
              : '📝 Text'}
          </span>
        </div>
      )}

      {/* ── Output hint ── */}
      {cfg.output_hint && (
        <p className="text-white/30 text-xs leading-relaxed">{cfg.output_hint}</p>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid() || isLoading || !canAfford || docUploading}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid() && !isLoading && canAfford && !docUploading
            ? 'bg-gradient-to-r from-nexus-600 to-purple-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-nexus-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Generating…</>
          : docUploading
            ? <><Loader2 size={15} className="animate-spin" /> Uploading document…</>
            : <><Sparkles size={15} /> {btnLabel}</>
        }
      </button>
    </div>
  );
}
