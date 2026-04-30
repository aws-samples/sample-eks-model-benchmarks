-- OOM events table for tracking out-of-memory occurrences
CREATE TABLE IF NOT EXISTS oom_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID REFERENCES benchmark_runs(id) ON DELETE CASCADE,
    model_hf_id VARCHAR(255) NOT NULL,
    instance_type VARCHAR(50) NOT NULL,
    pod_name VARCHAR(255) NOT NULL,
    container_name VARCHAR(255),
    detection_method VARCHAR(50) NOT NULL,
    exit_code INT,
    message TEXT,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Configuration at time of OOM
    tensor_parallel_degree INT,
    concurrency INT,
    max_model_len INT,
    quantization VARCHAR(20)
);

-- Index for looking up OOM history by model+instance
CREATE INDEX idx_oom_events_model_instance ON oom_events(model_hf_id, instance_type);

-- Index for looking up OOMs by run
CREATE INDEX idx_oom_events_run_id ON oom_events(run_id);
