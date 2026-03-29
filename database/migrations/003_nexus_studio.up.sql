-- 003_nexus_studio.sql
-- Purpose: Schema for the points-funded creative studio.

-- 1. Studio Tools Catalogue
CREATE TABLE IF NOT EXISTS studio_tools (
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
CREATE TABLE IF NOT EXISTS ai_generations (
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

CREATE INDEX IF NOT EXISTS idx_ai_generations_user ON ai_generations(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_generations_status ON ai_generations(status);

-- Seed Initial Tools (Full Catalogue - Appendix B)
INSERT INTO studio_tools (name, description, category, point_cost, provider, provider_tool_id, icon_name) VALUES
('Ask Nexus', 'Conversational AI assistant for brainstorming and help.', 'Chat', 0, 'GROQ', 'llama-4-scout', 'MessageSquare'),
('My AI Photo', 'Generate professional AI portraits from text.', 'Create', 10, 'HUGGING_FACE', 'flux-1-schnell', 'Camera'),
('Background Remover', 'Instantly remove backgrounds from your photos.', 'Create', 2, 'REM_BG', 'self-hosted', 'Scissors'),
('Animate My Photo', 'Turn your AI photo into a 5-second video.', 'Create', 65, 'FAL_AI', 'ltx-video', 'Video'),
('My Marketing Jingle', 'Generate 30s original music for your brand.', 'Create', 100, 'MUBERT', 'audio-gen', 'Music'),
('My Video Story', 'Combined AI Photo and Jingle into a branded video.', 'Create', 470, 'PIPELINE', 'composite-video', 'Clapperboard'),
('Study Guide', 'Generate a structured study guide on any topic.', 'Learn', 3, 'NOTEBOOK_LM', 'pdf-gen', 'BookOpen'),
('Quiz Me', 'Generate 10 multiple-choice questions on any topic.', 'Learn', 2, 'NOTEBOOK_LM', 'quiz-gen', 'HelpCircle'),
('Mind Map', 'Create a visual mind map from any concept.', 'Learn', 2, 'NOTEBOOK_LM', 'mindmap-gen', 'Network'),
('Deep Research Brief', 'Comprehensive research coverage on any topic.', 'Learn', 3, 'NOTEBOOK_LM', 'research-gen', 'Search'),
('My Podcast', 'Turn any topic into a 5-minute conversation.', 'Learn', 4, 'NOTEBOOK_LM', 'audio-gen', 'Mic'),
('Slide Deck', 'Professional PowerPoint presentation on any topic.', 'Build', 4, 'NOTEBOOK_LM', 'pptx-gen', 'Presentation'),
('Infographic', 'Visual summary of key facts and topics.', 'Build', 4, 'NOTEBOOK_LM', 'infographic-gen', 'PieChart'),
('Business Plan Summary', 'One-page professional business plan summary.', 'Build', 5, 'NOTEBOOK_LM', 'business-plan', 'FileText'),
('Voice to Plan', 'Record your idea to get a structured business plan.', 'Build', 6, 'ASSEMBLY_AI', 'voice-plan', 'Mic2'),
('Local Translation', 'Translate any text to Hausa, Yoruba, Igbo or Pidgin.', 'Build', 2, 'GOOGLE', 'translate', 'Languages'),
('Text to Speech', 'Natural audio reading with a Nigerian accent.', 'Build', 5, 'GOOGLE', 'tts-nigeria', 'Volume2')
ON CONFLICT (name) DO NOTHING;
