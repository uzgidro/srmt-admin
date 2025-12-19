-- Create index on level column for faster lookups
CREATE INDEX IF NOT EXISTS idx_level_volume_level ON level_volume(level);

CREATE INDEX IF NOT EXISTS idx_level_volume_volume ON level_volume(volume);
