-- Add loadgen_started_at to track when the load generator job actually starts
-- This allows accurate calculation of benchmark duration (excluding model deployment time)

ALTER TABLE benchmark_runs ADD COLUMN IF NOT EXISTS loadgen_started_at TIMESTAMPTZ;
