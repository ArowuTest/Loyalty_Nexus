'use client';

import { useState } from 'react';
import { Loader2, Sparkles, ArrowRight } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

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
    return typedFields.every((f) => !f.required || (values[f.key]?.trim().length ?? 0) >= 3);
  }

  function handleSubmit() {
    if (!isValid() || isLoading || !canAfford) return;
    const parts = typedFields
      .filter((f) => values[f.key]?.trim())
      .map((f) => `${f.label}: ${values[f.key].trim()}`);
    const payload: GeneratePayload = {
      prompt: parts.join('\n'),
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

      {/* ── Dynamic fields ── */}
      {typedFields.map((field, idx) => (
        <div key={field.key}>
          <label className="text-white/50 text-[11px] uppercase tracking-wider font-semibold mb-1.5 block">
            {field.label}
            {field.required && <span className="text-red-400 ml-1">*</span>}
          </label>

          {field.type === 'textarea' ? (
            <>
              <textarea
                value={values[field.key]}
                onChange={(e) => setValue(field.key, e.target.value)}
                placeholder={field.placeholder}
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
        disabled={!isValid() || isLoading || !canAfford}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid() && !isLoading && canAfford
            ? 'bg-gradient-to-r from-nexus-600 to-purple-600 text-white hover:opacity-90 active:scale-[0.98] shadow-lg shadow-nexus-900/30'
            : 'bg-white/5 text-white/20 cursor-not-allowed',
        )}
      >
        {isLoading
          ? <><Loader2 size={15} className="animate-spin" /> Generating…</>
          : <><Sparkles size={15} /> {btnLabel}</>
        }
      </button>
    </div>
  );
}
