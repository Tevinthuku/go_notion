ALTER TABLE pages ADD COLUMN IF NOT EXISTS is_top_level boolean NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_pages_is_top_level ON pages (is_top_level);

-- the existing pages before this migration are all top level
UPDATE pages SET is_top_level = true;
