-- Migration 129: store a user's ElevenLabs Instant-Voice-Clone id.
-- Lets the Talking Avatar tool speak a typed script in the USER'S OWN voice:
-- they record a short sample once → we register it with ElevenLabs → the
-- returned voice_id is stored here and reused for all future generations.
-- Empty string = no clone yet (the tool falls back to preset voices).
ALTER TABLE users ADD COLUMN IF NOT EXISTS cloned_voice_id TEXT NOT NULL DEFAULT '';
