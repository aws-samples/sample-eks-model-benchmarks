-- Model cache table for tracking HF-to-S3 cached models and custom S3 models.
CREATE TABLE IF NOT EXISTS model_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hf_id TEXT,
    hf_revision TEXT DEFAULT 'main',
    s3_uri TEXT NOT NULL,
    display_name TEXT NOT NULL,
    size_bytes BIGINT,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'caching', 'cached', 'failed', 'deleting')),
    error_message TEXT,
    job_name TEXT,
    cached_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_model_cache_hf_id ON model_cache (hf_id, hf_revision) WHERE hf_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_model_cache_s3_uri ON model_cache (s3_uri);
CREATE INDEX IF NOT EXISTS idx_model_cache_status ON model_cache (status);
