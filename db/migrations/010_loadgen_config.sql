-- Add loadgen_config column to store the inference-perf configuration YAML
-- This preserves the exact benchmark parameters used for each run

ALTER TABLE benchmark_runs ADD COLUMN IF NOT EXISTS loadgen_config TEXT;

COMMENT ON COLUMN benchmark_runs.loadgen_config IS 'inference-perf configuration YAML used for this benchmark run';
