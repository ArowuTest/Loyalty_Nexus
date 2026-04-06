-- Migration 101: Seed Gemma 4 / Nexus AI Tools
-- Adds four new AI tools powered by Gemma 4 multimodal capabilities.
-- Fully idempotent — safe to run multiple times.
-- ============================================================

-- ─── 1. Ensure the canonical slug list allows the new tools ──────────────────
-- The 067 migration deleted non-canonical slugs. We must ensure these new
-- slugs are not accidentally deleted by any future re-run of 067.
-- (No action needed here — 067 only deletes slugs NOT in its explicit list,
--  and new migrations run after 067, so these rows are safe.)

-- ─── 2. Upsert the four new Gemma 4 tools ────────────────────────────────────
INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     is_active, is_free, icon, sort_order, entry_point_cost, ui_template, ui_config,
     created_at, updated_at)
VALUES

-- ── Nexus Code Pro ────────────────────────────────────────────────────────────
-- Upgrade of code-helper with optional image upload for visual debugging.
-- Uses VisionAsk template so users can attach screenshots, error tracebacks,
-- or architecture diagrams alongside their code question.
(
  gen_random_uuid(),
  'Nexus Code Pro',
  'code-pro',
  'Advanced AI code assistant with visual debugging. Attach a screenshot of your error, UI bug, or architecture diagram for context-aware solutions.',
  'Build',
  40,
  'pollinations',
  'qwen-coder',
  true,
  false,
  '🧑‍💻',
  27,
  10,
  'VisionAsk',
  '{
    "upload_label": "Attach screenshot (optional)",
    "upload_accept": ["image/png", "image/jpeg", "image/webp", "image/gif"],
    "max_file_mb": 20,
    "prompt_optional": false,
    "example_questions": [
      "Why is this error happening and how do I fix it?",
      "Explain what this code does and suggest improvements",
      "Convert this UI design into React + Tailwind code",
      "Debug this traceback and show the corrected code",
      "What architecture pattern is shown in this diagram?",
      "Write unit tests for this function",
      "Optimise this SQL query for performance",
      "Explain this code to a junior developer"
    ],
    "prompt_placeholder": "Describe your code problem — or attach a screenshot of the error, UI bug, or architecture diagram…"
  }'::jsonb,
  NOW(),
  NOW()
),

-- ── Nexus Document Analyzer ───────────────────────────────────────────────────
-- Structured extraction from PDFs, invoices, charts, and scanned documents.
-- Uses KnowledgeDoc template with document upload support.
(
  gen_random_uuid(),
  'Nexus Document Analyzer',
  'doc-analyzer',
  'Upload a PDF, invoice, contract, or chart and get structured insights, summaries, and extracted data in seconds.',
  'Build',
  35,
  'gemini',
  'gemini-2.0-flash',
  true,
  false,
  '📋',
  28,
  10,
  'KnowledgeDoc',
  '{
    "output_format": "document",
    "fields": [
      {
        "key": "prompt",
        "label": "What do you want to know?",
        "type": "textarea",
        "required": false,
        "placeholder": "e.g. Summarise the key findings — or Extract all invoice line items — or What are the payment terms?",
        "rows": 4,
        "default": ""
      }
    ],
    "prompt_placeholder": "Ask a question about your document — or leave blank for a full analysis…",
    "prompt_hint": "Upload a PDF, invoice, contract, research paper, or scanned document. Nexus AI will extract, summarise, and answer your questions."
  }'::jsonb,
  NOW(),
  NOW()
),

-- ── Nexus Localization Engine ─────────────────────────────────────────────────
-- OCR + African dialect translation from screenshots.
-- Uses VisionAsk template with language selector.
(
  gen_random_uuid(),
  'Nexus Localization Engine',
  'localize-ui',
  'Upload a screenshot of your app or marketing material. Nexus AI reads all visible text and translates it into Yoruba, Hausa, Igbo, or Nigerian Pidgin.',
  'Language & Translation',
  50,
  'pollinations',
  'gemini-vision',
  true,
  false,
  '🌍',
  29,
  15,
  'VisionAsk',
  '{
    "upload_label": "Upload UI screenshot or marketing image",
    "upload_accept": ["image/png", "image/jpeg", "image/webp", "image/gif"],
    "max_file_mb": 20,
    "prompt_optional": true,
    "example_questions": [
      "Translate all text to Yoruba",
      "Translate all text to Hausa",
      "Translate all text to Igbo",
      "Translate all text to Nigerian Pidgin",
      "Translate the button labels and error messages only",
      "Translate the navigation menu items",
      "Translate all CTAs and headlines",
      "Provide a full localisation with cultural notes"
    ],
    "prompt_placeholder": "Describe what to translate — e.g. Translate all text to Yoruba — or leave blank for full auto-translation…"
  }'::jsonb,
  NOW(),
  NOW()
),

-- ── Nexus Agentic Workflow Builder ────────────────────────────────────────────
-- Multi-step agentic reasoning loop for complex tasks.
-- Uses KnowledgeDoc template with optional document upload.
(
  gen_random_uuid(),
  'Nexus Agent',
  'nexus-agent',
  'Describe a complex multi-step task in plain English. Nexus AI breaks it into steps, executes each one, and delivers a complete, polished result.',
  'Build',
  60,
  'gemini',
  'gemini-2.0-flash',
  true,
  false,
  '🤖',
  30,
  15,
  'KnowledgeDoc',
  '{
    "output_format": "document",
    "fields": [
      {
        "key": "prompt",
        "label": "Describe your multi-step task",
        "type": "textarea",
        "required": true,
        "placeholder": "e.g. Analyse this CSV data, identify the top 3 trends, and write an executive summary with recommendations…",
        "rows": 6,
        "default": ""
      }
    ],
    "prompt_placeholder": "Describe your complex task — Nexus Agent will plan, execute, and deliver a complete result…",
    "prompt_hint": "Best for: data analysis + report writing, research + synthesis, content planning + execution, document review + recommendations."
  }'::jsonb,
  NOW(),
  NOW()
)

ON CONFLICT (slug) DO UPDATE SET
    name             = EXCLUDED.name,
    description      = EXCLUDED.description,
    category         = EXCLUDED.category,
    point_cost       = EXCLUDED.point_cost,
    provider         = EXCLUDED.provider,
    provider_tool    = EXCLUDED.provider_tool,
    is_active        = EXCLUDED.is_active,
    is_free          = EXCLUDED.is_free,
    icon             = EXCLUDED.icon,
    sort_order       = EXCLUDED.sort_order,
    entry_point_cost = EXCLUDED.entry_point_cost,
    ui_template      = EXCLUDED.ui_template,
    ui_config        = EXCLUDED.ui_config,
    updated_at       = NOW();

-- ─── 3. Verification query (informational only) ───────────────────────────────
-- SELECT slug, name, category, point_cost, ui_template
-- FROM studio_tools
-- WHERE slug IN ('code-pro', 'doc-analyzer', 'localize-ui', 'nexus-agent')
-- ORDER BY sort_order;
