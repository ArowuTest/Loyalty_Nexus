-- Revert: rename back to "Song Creator"
UPDATE studio_tools
SET    name = 'Song Creator',
       updated_at = NOW()
WHERE  slug = 'song-creator';
