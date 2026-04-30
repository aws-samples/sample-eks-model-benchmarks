// Package oom provides OOM (Out of Memory) detection for Kubernetes pods.
package oom

import (
	"time"
)

// Event represents a detected OOM occurrence.
type Event struct {
	ID             string    `json:"id" db:"id"`
	RunID          string    `json:"run_id" db:"run_id"`
	ModelHfID      string    `json:"model_hf_id" db:"model_hf_id"`
	InstanceType   string    `json:"instance_type" db:"instance_type"`
	PodName        string    `json:"pod_name" db:"pod_name"`
	ContainerName  string    `json:"container_name" db:"container_name"`
	DetectionMethod string   `json:"detection_method" db:"detection_method"`
	ExitCode       int       `json:"exit_code" db:"exit_code"`
	Message        string    `json:"message" db:"message"`
	OccurredAt     time.Time `json:"occurred_at" db:"occurred_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`

	// Configuration at time of OOM (for analysis)
	TensorParallelDegree int    `json:"tensor_parallel_degree" db:"tensor_parallel_degree"`
	Concurrency          int    `json:"concurrency" db:"concurrency"`
	MaxModelLen          int    `json:"max_model_len" db:"max_model_len"`
	Quantization         string `json:"quantization,omitempty" db:"quantization"`
}

// DetectionMethod indicates how the OOM was detected.
const (
	// DetectionTerminationReason - Pod status shows OOMKilled reason
	DetectionTerminationReason = "termination_reason"

	// DetectionExitCode137 - Container exited with code 137 (SIGKILL from OOM killer)
	DetectionExitCode137 = "exit_code_137"

	// DetectionKubeEvent - Kubernetes event with OOMKilling reason
	DetectionKubeEvent = "kube_event"

	// DetectionRestartIncrease - Container restart count increased during benchmark
	DetectionRestartIncrease = "restart_increase"
)

// History holds OOM events for a model+instance combination.
type History struct {
	ModelHfID    string  `json:"model_hf_id"`
	InstanceType string  `json:"instance_type"`
	Events       []Event `json:"events"`
	TotalCount   int     `json:"total_count"`
}

// Suggestion provides recommendations based on OOM history.
type Suggestion struct {
	ReduceConcurrency bool   `json:"reduce_concurrency,omitempty"`
	IncreaseOverhead  bool   `json:"increase_overhead,omitempty"`
	SuggestedValue    int    `json:"suggested_value,omitempty"`
	Message           string `json:"message"`
}
