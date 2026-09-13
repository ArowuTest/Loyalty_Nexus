-- Migration 131: AI Routing V2
-- Per-tool/per-stage routing, surge limits, attempt telemetry and model intelligence.

ALTER TABLE studio_tools
  ADD COLUMN IF NOT EXISTS execution_profile TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS is_internal BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS ai_tool_stages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tool_id UUID NOT NULL REFERENCES studio_tools(id) ON DELETE CASCADE,
  stage_key TEXT NOT NULL,
  capability TEXT NOT NULL,
  routing_policy TEXT NOT NULL DEFAULT 'FREE_FIRST',
  queue_class TEXT NOT NULL DEFAULT 'INTERACTIVE',
  max_queue_seconds INT NOT NULL DEFAULT 30,
  paid_hourly_budget_micros BIGINT NOT NULL DEFAULT 0,
  paid_daily_budget_micros BIGINT NOT NULL DEFAULT 0,
  is_required BOOLEAN NOT NULL DEFAULT TRUE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(tool_id, stage_key),
  CHECK (routing_policy IN ('FREE_FIRST','BALANCED','QUALITY_FIRST','FREE_ONLY','PREMIUM_ONLY')),
  CHECK (queue_class IN ('REALTIME','INTERACTIVE','ASYNC','HEAVY_ASYNC','BACKGROUND')),
  CHECK (max_queue_seconds >= 0),
  CHECK (paid_hourly_budget_micros >= 0),
  CHECK (paid_daily_budget_micros >= 0)
);

CREATE TABLE IF NOT EXISTS ai_tool_provider_bindings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  stage_id UUID NOT NULL REFERENCES ai_tool_stages(id) ON DELETE CASCADE,
  provider_id UUID NOT NULL REFERENCES ai_provider_configs(id) ON DELETE CASCADE,
  priority INT NOT NULL DEFAULT 100,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  cost_tier TEXT NOT NULL DEFAULT 'FREE',
  max_concurrent INT NOT NULL DEFAULT 0,
  requests_per_minute INT NOT NULL DEFAULT 0,
  timeout_ms INT NOT NULL DEFAULT 120000,
  max_retries INT NOT NULL DEFAULT 0,
  allow_paid_fallback BOOLEAN NOT NULL DEFAULT TRUE,
  circuit_failure_threshold INT NOT NULL DEFAULT 5,
  circuit_open_seconds INT NOT NULL DEFAULT 60,
  request_config JSONB NOT NULL DEFAULT '{}'::jsonb,
  config_version BIGINT NOT NULL DEFAULT 1,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(stage_id, provider_id),
  CHECK (cost_tier IN ('FREE','LOW_COST','PREMIUM')),
  CHECK (priority > 0),
  CHECK (max_concurrent >= 0),
  CHECK (requests_per_minute >= 0),
  CHECK (timeout_ms BETWEEN 1000 AND 600000),
  CHECK (max_retries BETWEEN 0 AND 3),
  CHECK (circuit_failure_threshold BETWEEN 1 AND 100),
  CHECK (circuit_open_seconds BETWEEN 1 AND 86400)
);

CREATE INDEX IF NOT EXISTS idx_ai_tool_bindings_stage_priority
  ON ai_tool_provider_bindings(stage_id, priority) WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS ai_generation_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  generation_id UUID REFERENCES ai_generations(id) ON DELETE CASCADE,
  tool_id UUID REFERENCES studio_tools(id) ON DELETE SET NULL,
  stage_id UUID REFERENCES ai_tool_stages(id) ON DELETE SET NULL,
  binding_id UUID REFERENCES ai_tool_provider_bindings(id) ON DELETE SET NULL,
  provider_id UUID REFERENCES ai_provider_configs(id) ON DELETE SET NULL,
  stage_key TEXT NOT NULL DEFAULT 'main',
  attempt_no INT NOT NULL DEFAULT 1,
  provider_slug TEXT NOT NULL DEFAULT '',
  model_id TEXT NOT NULL DEFAULT '',
  outcome TEXT NOT NULL,
  error_class TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  http_status INT NOT NULL DEFAULT 0,
  duration_ms INT NOT NULL DEFAULT 0,
  cost_micros INT NOT NULL DEFAULT 0,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  CHECK (outcome IN ('STARTED','SUCCEEDED','FAILED','SKIPPED_CAPACITY','SKIPPED_CIRCUIT','SKIPPED_BUDGET'))
);

CREATE INDEX IF NOT EXISTS idx_ai_generation_attempts_gen
  ON ai_generation_attempts(generation_id, stage_key, attempt_no);
CREATE INDEX IF NOT EXISTS idx_ai_generation_attempts_provider_time
  ON ai_generation_attempts(provider_id, started_at DESC);

CREATE TABLE IF NOT EXISTS ai_model_catalog (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source TEXT NOT NULL,
  model_id TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  display_name TEXT NOT NULL DEFAULT '',
  is_free BOOLEAN NOT NULL DEFAULT FALSE,
  is_available BOOLEAN NOT NULL DEFAULT TRUE,
  context_window BIGINT NOT NULL DEFAULT 0,
  capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
  pricing JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(source, model_id)
);
CREATE TABLE IF NOT EXISTS ai_model_scores (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  model_catalog_id UUID NOT NULL REFERENCES ai_model_catalog(id) ON DELETE CASCADE,
  capability TEXT NOT NULL,
  score_source TEXT NOT NULL,
  score NUMERIC(8,4) NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(model_catalog_id, capability, score_source)
);

-- Backfill execution profiles for known current tools. Runtime migration remains
-- gated until all specialised paths are converted to Router V2.
UPDATE studio_tools SET execution_profile = CASE
  WHEN slug = 'web-search-ai' THEN 'web-search'
  WHEN slug IN ('translate','local-translation') THEN 'translate'
  WHEN slug IN ('narrate','narrate-pro','text-to-speech') THEN 'tts'
  WHEN slug IN ('transcribe','transcribe-african') THEN 'transcribe'
  WHEN slug IN ('bg-remover','background-remover') THEN 'background-remove'
  WHEN slug IN ('ai-photo','ai-photo-pro','ai-photo-max','ai-photo-dream','my-ai-photo') THEN 'image-generate'
  WHEN slug IN ('photo-editor') THEN 'image-edit'
  WHEN slug IN ('image-compose') THEN 'image-compose'
  WHEN slug IN ('animate-photo','video-premium','video-cinematic','video-veo','my-video-story','animate-my-photo','video-story') THEN 'video-generate'
  WHEN slug IN ('video-edit') THEN 'video-edit'
  WHEN slug IN ('video-extend') THEN 'video-extend'
  WHEN slug IN ('bg-music','jingle','my-marketing-jingle','song-creator','instrumental') THEN 'music-generate'
  WHEN slug IN ('image-analyser','ask-my-photo','code-pro','doc-analyzer','localize-ui') THEN 'vision'
  WHEN slug = 'talking-avatar' THEN 'avatar'
  WHEN slug = 'video-slideshow' THEN 'render'
  WHEN slug IN ('podcast','my-podcast') THEN 'podcast'
  WHEN slug = 'video-jingle' THEN 'video-jingle'
  ELSE 'text-generate'
END
WHERE execution_profile = '';
-- Internal Model Scout is governed like every other AI workload but never appears in the customer catalogue.
INSERT INTO studio_tools
  (id, name, slug, description, category, point_cost, provider, provider_tool, provider_tool_id,
   is_active, is_free, is_internal, execution_profile, ui_template, ui_config, sort_order, created_at, updated_at)
VALUES
  (gen_random_uuid(), 'AI Model Scout', '__ai-model-scout',
   'Internal background model-discovery and recommendation agent', 'Build', 0, '', '', '',
   TRUE, TRUE, TRUE, 'text-generate', 'knowledge-doc', '{}'::jsonb, 9999, NOW(), NOW())
ON CONFLICT (slug) DO UPDATE SET
  is_internal = TRUE, is_active = TRUE, is_free = TRUE, execution_profile = 'text-generate', updated_at = NOW();

INSERT INTO studio_tools
  (id, name, slug, description, category, point_cost, provider, provider_tool, provider_tool_id,
   is_active, is_free, is_internal, execution_profile, ui_template, ui_config, sort_order, created_at, updated_at)
VALUES
  (gen_random_uuid(), 'AI Memory Summarizer', '__ai-memory-summarizer',
   'Internal conversation-memory compression workload', 'Build', 0, '', '', '',
   TRUE, TRUE, TRUE, 'text-generate', 'knowledge-doc', '{}'::jsonb, 9998, NOW(), NOW())
ON CONFLICT (slug) DO UPDATE SET
  is_internal = TRUE, is_active = TRUE, is_free = TRUE, execution_profile = 'text-generate', updated_at = NOW();

-- Standard single-stage tools.
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, max_queue_seconds)
SELECT id, 'main',
  CASE execution_profile
    WHEN 'web-search' THEN 'text.web_search'
    WHEN 'translate' THEN 'language.translate'
    WHEN 'tts' THEN 'audio.tts'
    WHEN 'transcribe' THEN 'audio.transcribe'
    WHEN 'background-remove' THEN 'image.background_remove'
    WHEN 'image-generate' THEN 'image.generate'
    WHEN 'image-edit' THEN 'image.edit'
    WHEN 'image-compose' THEN 'image.compose'
    WHEN 'video-generate' THEN 'video.generate'
    WHEN 'video-edit' THEN 'video.edit'
    WHEN 'video-extend' THEN 'video.extend'
    WHEN 'music-generate' THEN 'audio.music'
    WHEN 'vision' THEN 'text.vision'
    WHEN 'avatar' THEN 'avatar.text'
    WHEN 'render' THEN 'render.video'
    ELSE 'text.generate'
  END,
  CASE WHEN execution_profile IN ('image-generate','image-edit','image-compose','video-generate','video-edit','video-extend','avatar','render')
       THEN 'QUALITY_FIRST' ELSE 'FREE_FIRST' END,
  CASE WHEN execution_profile IN ('video-generate','video-edit','video-extend','render') THEN 'HEAVY_ASYNC'
       WHEN execution_profile IN ('image-generate','image-edit','image-compose','avatar','music-generate') THEN 'ASYNC'
       ELSE 'INTERACTIVE' END,
  CASE WHEN execution_profile IN ('video-generate','video-edit','video-extend','render') THEN 300
       WHEN execution_profile IN ('image-generate','image-edit','image-compose','avatar','music-generate') THEN 120
       ELSE 30 END
FROM studio_tools
WHERE execution_profile NOT IN ('podcast','video-jingle')
ON CONFLICT (tool_id, stage_key) DO NOTHING;
-- Optional cheap/free prompt-enhancement stages for generative media.
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, max_queue_seconds, is_required, sort_order)
SELECT id, 'prompt', 'text.generate', 'FREE_ONLY', 'INTERACTIVE', 5, FALSE, 0
FROM studio_tools
WHERE execution_profile IN ('image-generate','image-compose','video-generate','video-edit','video-extend')
ON CONFLICT (tool_id, stage_key) DO NOTHING;

-- Nexus Agent search is an independently configurable provider stage.
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, max_queue_seconds, is_required, sort_order)
SELECT id, 'search', 'web.search', 'QUALITY_FIRST', 'INTERACTIVE', 10, FALSE, 1
FROM studio_tools WHERE slug = 'nexus-agent'
ON CONFLICT (tool_id, stage_key) DO NOTHING;

-- Talking-avatar fallback stages. Clone speech is intentionally isolated from generic TTS.
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, max_queue_seconds, is_required, sort_order)
SELECT id, 'speech', 'audio.tts', 'QUALITY_FIRST', 'ASYNC', 30, FALSE, 1
FROM studio_tools WHERE execution_profile = 'avatar'
ON CONFLICT (tool_id, stage_key) DO NOTHING;

INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, max_queue_seconds, is_required, sort_order)
SELECT id, 'clone-speech', 'audio.tts.clone', 'PREMIUM_ONLY', 'ASYNC', 30, FALSE, 1
FROM studio_tools WHERE execution_profile = 'avatar'
ON CONFLICT (tool_id, stage_key) DO NOTHING;

-- Composite stages.
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, sort_order)
SELECT id, 'script', 'text.generate', 'FREE_FIRST', 'INTERACTIVE', 1 FROM studio_tools WHERE execution_profile='podcast'
ON CONFLICT (tool_id, stage_key) DO NOTHING;
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, sort_order)
SELECT id, 'speech', 'audio.tts', 'QUALITY_FIRST', 'ASYNC', 2 FROM studio_tools WHERE execution_profile='podcast'
ON CONFLICT (tool_id, stage_key) DO NOTHING;
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, sort_order)
SELECT id, 'music', 'audio.music', 'QUALITY_FIRST', 'ASYNC', 1 FROM studio_tools WHERE execution_profile='video-jingle'
ON CONFLICT (tool_id, stage_key) DO NOTHING;
INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, sort_order)
SELECT id, 'video', 'video.generate', 'QUALITY_FIRST', 'HEAVY_ASYNC', 2 FROM studio_tools WHERE execution_profile='video-jingle'
ON CONFLICT (tool_id, stage_key) DO NOTHING;

-- Provider drivers required by Router V2 but absent from the legacy registry.
INSERT INTO ai_provider_configs
  (name, slug, category, template, env_key, model_id, extra_config, priority, is_primary, is_active, cost_micros, pulse_pts, notes)
VALUES
  ('OpenRouter Catalog Source', 'openrouter-catalog', 'text', 'openai-compatible', 'OPENROUTER_API_KEY', 'openrouter/free',
   '{"base_url":"https://openrouter.ai/api","catalog_source":"openrouter","catalog_only":true,"default_max_concurrent":4,"default_rpm":10,"default_timeout_ms":120000}'::jsonb,
   999, FALSE, TRUE, 0, 0, 'Credential/catalog source only; excluded from automatic user-traffic bindings'),
  ('Tavily Search', 'tavily-search', 'search', 'tavily-search', 'TAVILY_API_KEY', '',
   '{}'::jsonb, 1, TRUE, TRUE, 0, 0, 'Nexus Agent web search provider; credential is Admin-swappable'),
  ('OpenAI Transcribe', 'openai-transcribe', 'transcribe', 'openai-transcribe', 'OPENAI_API_KEY', 'gpt-4o-transcribe',
   '{}'::jsonb, 1, TRUE, TRUE, 20, 0, 'Admin-configurable OpenAI transcription model'),
  ('Suno Music', 'suno-music', 'music', 'suno-music', 'SUNO_API_KEY', '',
   '{}'::jsonb, 1, TRUE, TRUE, 50000, 0, 'Premium full-song and instrumental generation'),
  ('HuggingFace MusicGen', 'hf-musicgen', 'music', 'hf-musicgen', 'HF_TOKEN', 'facebook/musicgen-small',
   '{}'::jsonb, 90, FALSE, TRUE, 0, 0, 'Free music fallback'),
  ('xAI Grok Imagine Image', 'grok-imagine-image', 'image', 'grok-image', 'XAI_API_KEY', 'grok-imagine-image',
   '{"resolution":"2k"}'::jsonb, 1, TRUE, TRUE, 70000, 0, 'Premium xAI image generation and multi-reference composition'),
  ('Pollinations GPT Image', 'pollinations-gptimage', 'image', 'pollinations-gpt-image', 'POLLINATIONS_SECRET_KEY', 'gptimage',
   '{}'::jsonb, 4, FALSE, TRUE, 20000, 0, 'Premium Pollinations image model; model_id is Admin-swappable'),
  ('Pollinations GPT Image Large', 'pollinations-gptimage-large', 'image', 'pollinations-gpt-image', 'POLLINATIONS_SECRET_KEY', 'gptimage-large',
   '{}'::jsonb, 5, FALSE, TRUE, 30000, 0, 'High-quality Pollinations image model'),
  ('Pollinations Seedream', 'pollinations-seedream5', 'image', 'pollinations-gpt-image', 'POLLINATIONS_SECRET_KEY', 'seedream5',
   '{}'::jsonb, 6, FALSE, TRUE, 10000, 0, 'Dream-style image generation'),
  ('Pollinations Image Edit', 'pollinations-image-edit', 'image', 'pollinations-image-edit', 'POLLINATIONS_SECRET_KEY', 'p-image-edit',
   '{}'::jsonb, 1, TRUE, TRUE, 10000, 0, 'Image-to-image edit driver'),
  ('FAL Flux Ultra Reference', 'fal-flux-ultra', 'image', 'fal-image-ultra', 'FAL_API_KEY', 'fal-ai/flux-pro/v1.1-ultra',
   '{}'::jsonb, 2, FALSE, TRUE, 40000, 0, 'Reference-guided image generation'),
  ('FAL Kontext Edit', 'fal-kontext-edit', 'image', 'fal-image-edit', 'FAL_API_KEY', 'fal-ai/flux-pro/kontext',
   '{}'::jsonb, 2, FALSE, TRUE, 20000, 0, 'Premium image editing'),
  ('xAI Grok Imagine Video', 'grok-imagine-video', 'video', 'grok-video', 'XAI_API_KEY', 'grok-imagine-video',
   '{}'::jsonb, 1, TRUE, TRUE, 300000, 0, 'xAI text/image/reference/video edit/extend driver'),
  ('FAL Kling Multi Image', 'fal-kling-multi', 'video', 'fal-video-multi', 'FAL_API_KEY', 'fal-ai/kling-video/v2.6/pro/multi-image-to-video',
   '{}'::jsonb, 2, FALSE, TRUE, 56000, 0, 'FAL multi-image video fallback')
ON CONFLICT (slug) DO UPDATE SET
  template = EXCLUDED.template,
  env_key = EXCLUDED.env_key,
  model_id = EXCLUDED.model_id,
  extra_config = ai_provider_configs.extra_config || EXCLUDED.extra_config,
  cost_micros = EXCLUDED.cost_micros,
  notes = EXCLUDED.notes,
  updated_at = NOW();

-- Backfill bindings from the existing category chains. This preserves current
-- configured order as migration data, but priority becomes stage-specific from now on.
INSERT INTO ai_tool_provider_bindings(stage_id, provider_id, priority, cost_tier)
SELECT s.id, p.id, p.priority,
  CASE WHEN p.cost_micros = 0 THEN 'FREE'
       WHEN p.cost_micros < 10000 THEN 'LOW_COST'
       ELSE 'PREMIUM' END
FROM ai_tool_stages s
JOIN ai_provider_configs p ON p.category = CASE
  WHEN s.capability IN ('text.generate','text.chat','text.web_search') THEN 'text'
  WHEN s.capability IN ('image.generate','image.edit','image.compose') THEN 'image'
  WHEN s.capability IN ('video.generate','video.edit','video.extend') THEN 'video'
  WHEN s.capability = 'audio.tts' THEN 'tts'
  WHEN s.capability = 'audio.transcribe' THEN 'transcribe'
  WHEN s.capability = 'language.translate' THEN 'translate'
  WHEN s.capability = 'audio.music' THEN 'music'
  WHEN s.capability = 'image.background_remove' THEN 'bg-remove'
  WHEN s.capability = 'text.vision' THEN 'vision'
  WHEN s.capability LIKE 'avatar.%' THEN 'avatar'
  WHEN s.capability = 'render.video' THEN 'render'
  WHEN s.capability = 'web.search' THEN 'search'
  ELSE '__none__' END
WHERE p.is_active = TRUE
  AND COALESCE(p.extra_config->>'catalog_only','false') <> 'true'
ON CONFLICT (stage_id, provider_id) DO NOTHING;

-- Clone-speech is deliberately restricted to clone-capable ElevenLabs TTS drivers.
INSERT INTO ai_tool_provider_bindings(stage_id, provider_id, priority, cost_tier, max_concurrent, requests_per_minute)
SELECT s.id, p.id, p.priority,
  CASE WHEN p.cost_micros = 0 THEN 'FREE'
       WHEN p.cost_micros < 10000 THEN 'LOW_COST'
       ELSE 'PREMIUM' END,
  12, 120
FROM ai_tool_stages s
JOIN studio_tools t ON t.id = s.tool_id
JOIN ai_provider_configs p ON p.template = 'elevenlabs-tts' AND p.is_active = TRUE
WHERE t.execution_profile = 'avatar' AND s.stage_key = 'clone-speech'
ON CONFLICT (stage_id, provider_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS ai_routing_change_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_type TEXT NOT NULL,
  entity_id UUID,
  action TEXT NOT NULL,
  before_state JSONB NOT NULL DEFAULT '{}'::jsonb,
  after_state JSONB NOT NULL DEFAULT '{}'::jsonb,
  changed_by TEXT NOT NULL DEFAULT '',
  changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_routing_change_log_time ON ai_routing_change_log(changed_at DESC);

CREATE TABLE IF NOT EXISTS ai_model_recommendations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tool_slug TEXT NOT NULL,
  stage_key TEXT NOT NULL DEFAULT 'main',
  model_catalog_id UUID NOT NULL REFERENCES ai_model_catalog(id) ON DELETE CASCADE,
  recommendation TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  score NUMERIC(8,4) NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'PENDING',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ,
  CHECK (status IN ('PENDING','APPROVED','REJECTED','SUPERSEDED'))
);
CREATE INDEX IF NOT EXISTS idx_ai_model_recommendations_pending
  ON ai_model_recommendations(status, created_at DESC);

UPDATE ai_tool_stages s
SET routing_policy = 'FREE_ONLY', queue_class = 'BACKGROUND', max_queue_seconds = 0
FROM studio_tools t
WHERE s.tool_id = t.id AND t.slug = '__ai-model-scout' AND s.stage_key = 'main';

UPDATE ai_tool_stages s
SET routing_policy = 'FREE_FIRST', queue_class = 'BACKGROUND', max_queue_seconds = 15
FROM studio_tools t
WHERE s.tool_id = t.id AND t.slug = '__ai-memory-summarizer' AND s.stage_key = 'main';

UPDATE ai_tool_provider_bindings b
SET request_config = jsonb_set(COALESCE(b.request_config, '{}'::jsonb), '{web_search}', 'true'::jsonb)
FROM ai_tool_stages s
JOIN studio_tools t ON t.id = s.tool_id
WHERE b.stage_id = s.id AND t.slug = 'web-search-ai';

-- Sensible initial music policies; Admin remains authoritative after migration.
UPDATE ai_tool_stages s
SET routing_policy = CASE WHEN t.slug = 'bg-music' THEN 'FREE_FIRST' ELSE 'QUALITY_FIRST' END,
    queue_class = 'ASYNC'
FROM studio_tools t
WHERE s.tool_id = t.id AND t.slug IN ('song-creator','instrumental','jingle','my-marketing-jingle','bg-music');

UPDATE ai_tool_provider_bindings b
SET priority = CASE
  WHEN p.slug = 'suno-music' THEN 1
  WHEN p.slug = 'elevenlabs-music' THEN 2
  WHEN p.slug = 'pollinations-elevenmusic' THEN 3
  WHEN p.slug = 'mubert' THEN 4
  WHEN p.slug = 'hf-musicgen' THEN 5
  ELSE b.priority
END
FROM ai_tool_stages s, studio_tools t, ai_provider_configs p
WHERE b.stage_id = s.id
  AND t.id = s.tool_id
  AND p.id = b.provider_id
  AND t.slug IN ('song-creator','instrumental','jingle','my-marketing-jingle');

UPDATE ai_tool_provider_bindings b
SET priority = CASE
  WHEN p.slug = 'hf-musicgen' THEN 1
  WHEN p.slug = 'pollinations-elevenmusic' THEN 2
  WHEN p.slug = 'elevenlabs-music' THEN 3
  WHEN p.slug = 'mubert' THEN 4
  WHEN p.slug = 'suno-music' THEN 5
  ELSE b.priority
END
FROM ai_tool_stages s, studio_tools t, ai_provider_configs p
WHERE b.stage_id = s.id
  AND t.id = s.tool_id
  AND p.id = b.provider_id
  AND t.slug = 'bg-music';

INSERT INTO ai_tool_stages(tool_id, stage_key, capability, routing_policy, queue_class, max_queue_seconds, is_required, sort_order)
SELECT id, 'prompt', 'text.generate', 'FREE_ONLY', 'INTERACTIVE', 5, FALSE, 0
FROM studio_tools WHERE execution_profile = 'video-jingle'
ON CONFLICT (tool_id, stage_key) DO NOTHING;

-- The video-jingle prompt stage is created after the generic binding backfill,
-- so bind it explicitly to the active text provider pool.
INSERT INTO ai_tool_provider_bindings(stage_id, provider_id, priority, cost_tier)
SELECT s.id, p.id, p.priority,
  CASE WHEN p.cost_micros = 0 THEN 'FREE'
       WHEN p.cost_micros < 10000 THEN 'LOW_COST'
       ELSE 'PREMIUM' END
FROM ai_tool_stages s
JOIN studio_tools t ON t.id = s.tool_id
JOIN ai_provider_configs p ON p.category = 'text'
WHERE t.execution_profile = 'video-jingle'
  AND s.stage_key = 'prompt'
  AND p.is_active = TRUE
  AND COALESCE(p.extra_config->>'catalog_only','false') <> 'true'
ON CONFLICT (stage_id, provider_id) DO NOTHING;

-- Stage activation follows the tool catalogue. Inactive/hidden tools must not
-- consume routing capacity or appear as uncovered active stages.
UPDATE ai_tool_stages s
SET is_active = t.is_active,
    updated_at = NOW()
FROM studio_tools t
WHERE s.tool_id = t.id
  AND s.is_active IS DISTINCT FROM t.is_active;

-- Seed surge protection for every migrated binding. 0 means unlimited in the
-- runtime capacity controller, so inherited legacy bindings must receive
-- queue-class defaults just like bindings created later through Admin.
UPDATE ai_tool_provider_bindings b
SET max_concurrent = CASE
      WHEN b.max_concurrent > 0 THEN b.max_concurrent
      WHEN s.queue_class = 'HEAVY_ASYNC' THEN 4
      WHEN s.queue_class = 'ASYNC' THEN 12
      WHEN s.queue_class = 'BACKGROUND' THEN 4
      WHEN s.queue_class IN ('REALTIME','INTERACTIVE') THEN 32
      ELSE 12
    END,
    requests_per_minute = CASE
      WHEN b.requests_per_minute > 0 THEN b.requests_per_minute
      WHEN s.queue_class = 'HEAVY_ASYNC' THEN 30
      WHEN s.queue_class = 'ASYNC' THEN 120
      WHEN s.queue_class = 'BACKGROUND' THEN 60
      WHEN s.queue_class IN ('REALTIME','INTERACTIVE') THEN 300
      ELSE 120
    END,
    updated_at = NOW()
FROM ai_tool_stages s
WHERE b.stage_id = s.id;

-- A REQUIRED active tool stage with no active provider candidate would be a
-- silently dead tool. Warn loudly but do NOT abort the migration: optional
-- stages (clone-speech, prompt, search) legitimately have no bindings, and an
-- admin who has disabled a whole provider category (e.g. all ElevenLabs TTS,
-- or translate) on a LIVE database must not brick the deploy — the router
-- already degrades gracefully via ErrNoConfiguredAIRoute. The original guard
-- ignored is_required and RAISEd EXCEPTION, which aborted the whole (atomic)
-- migration on realistic production data.
DO $$
DECLARE
  bad_count INT;
BEGIN
  SELECT count(*) INTO bad_count
  FROM ai_tool_stages s
  JOIN studio_tools t ON t.id = s.tool_id
  WHERE s.is_active = TRUE
    AND s.is_required = TRUE
    AND t.is_active = TRUE
    AND NOT EXISTS (
      SELECT 1
      FROM ai_tool_provider_bindings b
      WHERE b.stage_id = s.id AND b.is_active = TRUE
    );
  IF bad_count > 0 THEN
    RAISE WARNING 'Migration 131: % required active AI stage(s) have no active provider binding — configure them in Admin before those tools are used', bad_count;
  END IF;
END $$;
