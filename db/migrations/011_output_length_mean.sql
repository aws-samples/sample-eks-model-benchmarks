-- Add output_length_mean to benchmark_metrics for inference-perf output statistics
ALTER TABLE benchmark_metrics ADD COLUMN IF NOT EXISTS output_length_mean DOUBLE PRECISION;
