package database

import (
	"context"
	"fmt"
	"time"
)

// CreateOOMEvent inserts a new OOM event record.
func (r *Repository) CreateOOMEvent(ctx context.Context, event *OOMEvent) error {
	query := `
		INSERT INTO oom_events (
			run_id, model_hf_id, instance_type, pod_name, container_name,
			detection_method, exit_code, message, occurred_at,
			tensor_parallel_degree, concurrency, max_model_len, quantization
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`
	return r.pool.QueryRow(ctx, query,
		event.RunID, event.ModelHfID, event.InstanceType, event.PodName, event.ContainerName,
		event.DetectionMethod, event.ExitCode, event.Message, event.OccurredAt,
		event.TensorParallelDegree, event.Concurrency, event.MaxModelLen, event.Quantization,
	).Scan(&event.ID, &event.CreatedAt)
}

// GetOOMHistory returns OOM events for a model+instance combination.
func (r *Repository) GetOOMHistory(ctx context.Context, modelHfID, instanceType string, limit int) (*OOMHistory, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, run_id, model_hf_id, instance_type, pod_name, container_name,
			   detection_method, exit_code, message, occurred_at, created_at,
			   tensor_parallel_degree, concurrency, max_model_len, quantization
		FROM oom_events
		WHERE model_hf_id = $1 AND instance_type = $2
		ORDER BY occurred_at DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, modelHfID, instanceType, limit)
	if err != nil {
		return nil, fmt.Errorf("query oom events: %w", err)
	}
	defer rows.Close()

	history := &OOMHistory{
		ModelHfID:    modelHfID,
		InstanceType: instanceType,
	}

	for rows.Next() {
		var ev OOMEvent
		var runID, containerName, quantization *string
		var exitCode, tp, concurrency, maxModelLen *int
		var occurredAt, createdAt time.Time

		err := rows.Scan(
			&ev.ID, &runID, &ev.ModelHfID, &ev.InstanceType, &ev.PodName, &containerName,
			&ev.DetectionMethod, &exitCode, &ev.Message, &occurredAt, &createdAt,
			&tp, &concurrency, &maxModelLen, &quantization,
		)
		if err != nil {
			return nil, fmt.Errorf("scan oom event: %w", err)
		}

		if runID != nil {
			ev.RunID = *runID
		}
		if containerName != nil {
			ev.ContainerName = *containerName
		}
		if exitCode != nil {
			ev.ExitCode = *exitCode
		}
		if tp != nil {
			ev.TensorParallelDegree = *tp
		}
		if concurrency != nil {
			ev.Concurrency = *concurrency
		}
		if maxModelLen != nil {
			ev.MaxModelLen = *maxModelLen
		}
		if quantization != nil {
			ev.Quantization = *quantization
		}
		ev.OccurredAt = occurredAt
		ev.CreatedAt = createdAt

		history.Events = append(history.Events, ev)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM oom_events WHERE model_hf_id = $1 AND instance_type = $2`
	if err := r.pool.QueryRow(ctx, countQuery, modelHfID, instanceType).Scan(&history.TotalCount); err != nil {
		return nil, fmt.Errorf("count oom events: %w", err)
	}

	return history, rows.Err()
}
