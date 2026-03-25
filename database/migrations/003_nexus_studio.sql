-- 003_nexus_studio.sql
-- Purpose: Schema for the points-funded creative studio.

-- 1. Studio Tools Catalogue
CREATE TABLE studio_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    category TEXT CHECK (category IN ('Chat', 'Create', 'Learn', 'Build')),
    point_cost BIGINT NOT NULL DEFAULT 0,
    provider TEXT NOT NULL, -- e.g. 'FAL_AI', 'GROQ', 'GOOGLE'
    provider_tool_id TEXT NOT NULL, -- e.g. 'flux-schnell', 'llama-3-70b'
    icon_name TEXT, -- Lucide icon key
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 2. AI Generation History & Gallery
CREATE TABLE ai_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    tool_id UUID NOT NULL REFERENCES studio_tools(id),
    prompt TEXT NOT NULL,
    status TEXT CHECK (status IN ('pending', 'processing', 'completed', 'failed')) DEFAULT 'pending',
    output_url TEXT, -- Pre-signed S3 URL
    error_message TEXT,
    points_deducted BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL, -- 30-day lifecycle
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_ai_generations_user ON ai_generations(user_id);
CREATE INDEX idx_ai_generations_status ON ai_generations(status);

-- Seed Initial Tools
INSERT INTO studio_tools (name, description, category, point_cost, provider, provider_tool_id, icon_name) VALUES
('Ask Nexus', 'Conversational AI assistant for brainstorming and help.', 'Chat', 0, 'GROQ', 'llama-4-scout', 'MessageSquare'),
('My AI Photo', 'Generate professional AI portraits from text.', 'Create', 10, 'HUGGING_FACE', 'flux-1-schnell', 'Camera'),
('Background Remover', 'Instantly remove backgrounds from your photos.', 'Create', 2, 'REM_BG', 'self-hosted', 'Scissors'),
('Study Guide', 'Generate a structured study guide on any topic.', 'Learn', 3, 'NOTEBOOK_LM', 'pdf-gen', 'BookOpen'),
('My Podcast', 'Turn any topic into a 5-minute AI podcast episode.', 'Learn', 4, 'NOTEBOOK_LM', 'audio-gen', 'Mic');
