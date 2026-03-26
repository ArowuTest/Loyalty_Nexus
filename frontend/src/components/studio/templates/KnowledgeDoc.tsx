'use client';

import { useState } from 'react';
import { Loader2, Sparkles } from 'lucide-react';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';

// Default fallback for tools with no ui_config.fields
const DEFAULT_FIELDS = [
  { key: 'prompt', label: 'Describe what you want', type: 'textarea' as const, required: true,
    placeholder: 'Provide details about what you\'d like to generate…', rows: 5, default: '' },
];

export default function KnowledgeDoc({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg    = tool.ui_config ?? {};
  const fields = cfg.fields?.length ? cfg.fields : DEFAULT_FIELDS;

  const [values, setValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    fields.forEach((f) => { init[f.key] = f.default ?? ''; });
    return init;
  });

  const canAfford = tool.is_free || userPoints >= tool.point_cost;

  function isValid(): boolean {
    return fields.every((f) => !f.required || values[f.key]?.trim().length >= 3);
  }

  function handleSubmit() {
    if (!isValid() || isLoading || !canAfford) return;
    // Build a structured prompt from all fields
    const parts = fields
      .filter((f) => values[f.key]?.trim())
      .map((f) => `${f.label}: ${values[f.key].trim()}`);
    const payload: GeneratePayload = {
      prompt: parts.join('\n'),
      extra_params: { ...values },
    };
    onSubmit(payload);
  }

  function setValue(key: string, val: string) {
    setValues((prev) => ({ ...prev, [key]: val }));
  }

  return (
    <div className="space-y-5">

      {/* ── Dynamic fields ── */}
      {fields.map((field, idx) => (
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
              <p className="text-white/25 text-[11px] mt-1">{values[field.key]?.length ?? 0}/1000 characters</p>
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
            cfg.output_format === 'document' ? 'bg-blue-500/20 text-blue-300' :
            cfg.output_format === 'audio'    ? 'bg-green-500/20 text-green-300' :
                                               'bg-white/10 text-white/50',
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
          : <><Sparkles size={15} /> Generate →</>
        }
      </button>
    </div>
  );
}
