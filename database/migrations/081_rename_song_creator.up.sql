-- Migration 081: Rename "Song Creator" → "Song/Music Composer"
-- Keeps slug 'song-creator' unchanged (used by backend dispatch)

UPDATE studio_tools
SET    name = 'Song/Music Composer',
       updated_at = NOW()
WHERE  slug = 'song-creator';
