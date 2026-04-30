package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/accelbench/accelbench/internal/database"
)

// LoadgenOutput represents the JSON output from the load generator.
type LoadgenOutput struct {
	Requests []RequestResult `json:"requests"`
	Summary  Summary         `json:"summary"`
}

// RequestResult holds per-request measurements from the load generator.
// Supports both legacy AccelBench loadgen and inference-perf output formats.
type RequestResult struct {
	TTFTMs          float64 `json:"ttft_ms"`
	E2ELatencyMs    float64 `json:"e2e_latency_ms"`
	ITLMs           float64 `json:"itl_ms"`            // legacy loadgen
	TPOTMs          float64 `json:"tpot_ms"`           // inference-perf (time per output token)
	OutputTokens    int     `json:"output_tokens"`
	InputTokens     int     `json:"input_tokens"`
	DurationSeconds float64 `json:"duration_seconds"`
	Success         bool    `json:"success"`

	// Extended latency breakdown (PRD-14)
	PrefillTimeMs float64 `json:"prefill_time_ms"`
	DecodeTimeMs  float64 `json:"decode_time_ms"`
	QueueTimeMs   float64 `json:"queue_time_ms"`
}

// Summary holds aggregate metrics from the load generator.
// Supports both legacy AccelBench loadgen and inference-perf output formats.
type Summary struct {
	// Legacy loadgen fields
	TotalDurationSeconds   float64 `json:"total_duration_seconds"`
	ThroughputAggregateTPS float64 `json:"throughput_aggregate_tps"`

	// inference-perf fields
	TotalDurationS float64 `json:"total_duration_s"`
	ThroughputTPS  float64 `json:"throughput_tps"`

	// Common fields
	TotalRequests      int     `json:"total_requests"`
	SuccessfulRequests int     `json:"successful_requests"`
	FailedRequests     int     `json:"failed_requests"`
	RequestsPerSecond  float64 `json:"requests_per_second"`

	// Extended metrics from inference-perf
	InputThroughputTPS float64  `json:"input_throughput_tps,omitempty"`
	OutputLengthMean   float64  `json:"output_length_mean,omitempty"`
	TPOTP50Ms          float64  `json:"tpot_p50_ms,omitempty"`
	TPOTP90Ms          float64  `json:"tpot_p90_ms,omitempty"`
	TPOTP99Ms          float64  `json:"tpot_p99_ms,omitempty"`

	// Optional accelerator metrics (from legacy loadgen)
	AcceleratorUtilizationPct *float64 `json:"accelerator_utilization_pct,omitempty"`
	AcceleratorMemoryPeakGiB  *float64 `json:"accelerator_memory_peak_gib,omitempty"`
}

// InferencePerfOutput represents the JSON output from inference-perf v0.2.0+
type InferencePerfOutput struct {
	LoadSummary struct {
		Count int `json:"count"`
	} `json:"load_summary"`
	Successes struct {
		Count   int `json:"count"`
		Latency struct {
			RequestLatency struct {
				Median float64 `json:"median"`
				P90    float64 `json:"p90"`
				P95    float64 `json:"p95"`
				P99    float64 `json:"p99"`
			} `json:"request_latency"`
			TimeToFirstToken struct {
				Median float64 `json:"median"`
				P90    float64 `json:"p90"`
				P95    float64 `json:"p95"`
				P99    float64 `json:"p99"`
			} `json:"time_to_first_token"`
			InterTokenLatency struct {
				Median float64 `json:"median"`
				P90    float64 `json:"p90"`
				P95    float64 `json:"p95"`
				P99    float64 `json:"p99"`
			} `json:"inter_token_latency"`
			TimePerOutputToken struct {
				Median float64 `json:"median"`
				P90    float64 `json:"p90"`
				P95    float64 `json:"p95"`
				P99    float64 `json:"p99"`
			} `json:"time_per_output_token"`
		} `json:"latency"`
		Throughput struct {
			InputTokensPerSec  float64 `json:"input_tokens_per_sec"`
			OutputTokensPerSec float64 `json:"output_tokens_per_sec"`
			TotalTokensPerSec  float64 `json:"total_tokens_per_sec"`
			RequestsPerSec     float64 `json:"requests_per_sec"`
		} `json:"throughput"`
		OutputLen struct {
			Mean float64 `json:"mean"`
		} `json:"output_len"`
	} `json:"successes"`
	Failures struct {
		Count int `json:"count"`
	} `json:"failures"`
}

// ParseLoadgenOutput parses the JSON output from a load generator pod.
// With S3 storage, the JSON is read directly from a file without truncation.
// Falls back to log parsing for backward compatibility.
func ParseLoadgenOutput(data []byte) (*LoadgenOutput, error) {
	var out LoadgenOutput

	// Strategy 1: Try direct JSON unmarshal (legacy AccelBench format)
	if err := json.Unmarshal(data, &out); err == nil && len(out.Requests) > 0 {
		return &out, nil
	}

	// Strategy 2: Try inference-perf v0.2.0 format
	var ipOut InferencePerfOutput
	if err := json.Unmarshal(data, &ipOut); err == nil && ipOut.Successes.Count > 0 {
		return convertInferencePerfOutput(&ipOut), nil
	}

	// Strategy 3: Look for marker-delimited JSON (log parsing fallback)
	beginMarker := []byte("ACCELBENCH_JSON_BEGIN")
	endMarker := []byte("ACCELBENCH_JSON_END")
	if beginIdx := bytes.Index(data, beginMarker); beginIdx >= 0 {
		rest := data[beginIdx+len(beginMarker):]
		if endIdx := bytes.Index(rest, endMarker); endIdx >= 0 {
			jsonData := bytes.TrimSpace(rest[:endIdx])
			if err := json.Unmarshal(jsonData, &out); err == nil && len(out.Requests) > 0 {
				return &out, nil
			}
		}
	}

	// Strategy 4: Scan line-by-line for a JSON payload
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		if err := json.Unmarshal(line, &out); err == nil && len(out.Requests) > 0 {
			return &out, nil
		}
	}

	return nil, fmt.Errorf("no valid JSON payload found in %d bytes of output", len(data))
}

// convertInferencePerfOutput converts inference-perf format to LoadgenOutput.
// Since inference-perf provides aggregate stats (not per-request), we create
// synthetic request entries to pass through the existing metrics pipeline.
func convertInferencePerfOutput(ip *InferencePerfOutput) *LoadgenOutput {
	// inference-perf reports latencies in seconds, convert to milliseconds
	ttftP50 := ip.Successes.Latency.TimeToFirstToken.Median * 1000
	ttftP90 := ip.Successes.Latency.TimeToFirstToken.P90 * 1000
	ttftP99 := ip.Successes.Latency.TimeToFirstToken.P99 * 1000

	e2eP50 := ip.Successes.Latency.RequestLatency.Median * 1000
	e2eP90 := ip.Successes.Latency.RequestLatency.P90 * 1000
	e2eP99 := ip.Successes.Latency.RequestLatency.P99 * 1000

	itlP50 := ip.Successes.Latency.InterTokenLatency.Median * 1000
	itlP90 := ip.Successes.Latency.InterTokenLatency.P90 * 1000
	itlP99 := ip.Successes.Latency.InterTokenLatency.P99 * 1000

	// TPOT (time per output token) in milliseconds
	tpotP50 := ip.Successes.Latency.TimePerOutputToken.Median * 1000
	tpotP90 := ip.Successes.Latency.TimePerOutputToken.P90 * 1000
	tpotP99 := ip.Successes.Latency.TimePerOutputToken.P99 * 1000

	// Create synthetic requests at p50, p90, p99 to preserve percentile info
	// The ComputeMetrics function will recompute percentiles from these
	requests := []RequestResult{
		{TTFTMs: ttftP50, E2ELatencyMs: e2eP50, ITLMs: itlP50, TPOTMs: tpotP50, Success: true},
		{TTFTMs: ttftP50, E2ELatencyMs: e2eP50, ITLMs: itlP50, TPOTMs: tpotP50, Success: true},
		{TTFTMs: ttftP90, E2ELatencyMs: e2eP90, ITLMs: itlP90, TPOTMs: tpotP90, Success: true},
		{TTFTMs: ttftP99, E2ELatencyMs: e2eP99, ITLMs: itlP99, TPOTMs: tpotP99, Success: true},
	}

	return &LoadgenOutput{
		Requests: requests,
		Summary: Summary{
			TotalRequests:          ip.LoadSummary.Count,
			SuccessfulRequests:     ip.Successes.Count,
			FailedRequests:         ip.Failures.Count,
			ThroughputAggregateTPS: ip.Successes.Throughput.OutputTokensPerSec,
			RequestsPerSecond:      ip.Successes.Throughput.RequestsPerSec,
			InputThroughputTPS:     ip.Successes.Throughput.InputTokensPerSec,
			OutputLengthMean:       ip.Successes.OutputLen.Mean,
			TPOTP50Ms:              tpotP50,
			TPOTP90Ms:              tpotP90,
			TPOTP99Ms:              tpotP99,
		},
	}
}

// ComputeMetrics takes parsed loadgen output and computes the full set of
// benchmark metrics including p50/p90/p95/p99 percentiles.
// Supports both legacy AccelBench loadgen and inference-perf output formats.
func ComputeMetrics(out *LoadgenOutput) *database.BenchmarkMetrics {
	successful := filterSuccessful(out.Requests)

	var ttfts, e2es, itls []float64
	var tpots, prefills, decodes, queues []float64
	var totalOutputTokens int
	for _, r := range successful {
		ttfts = append(ttfts, r.TTFTMs)
		e2es = append(e2es, r.E2ELatencyMs)
		// Use ITL if available, otherwise use TPOT (inference-perf)
		itl := r.ITLMs
		if itl == 0 && r.TPOTMs > 0 {
			itl = r.TPOTMs
		}
		itls = append(itls, itl)
		totalOutputTokens += r.OutputTokens

		// Extended latency breakdown (only include if > 0)
		if r.TPOTMs > 0 {
			tpots = append(tpots, r.TPOTMs)
		}
		if r.PrefillTimeMs > 0 {
			prefills = append(prefills, r.PrefillTimeMs)
		}
		if r.DecodeTimeMs > 0 {
			decodes = append(decodes, r.DecodeTimeMs)
		}
		if r.QueueTimeMs > 0 {
			queues = append(queues, r.QueueTimeMs)
		}
	}

	ttftP50, ttftP90, ttftP95, ttftP99 := percentiles(ttfts)
	e2eP50, e2eP90, e2eP95, e2eP99 := percentiles(e2es)
	itlP50, itlP90, itlP95, itlP99 := percentiles(itls)

	// Extended latency percentiles
	// For TPOT, prefer per-request data but fall back to summary values from inference-perf
	tpotP50, tpotP90, _, tpotP99 := percentiles(tpots)
	if tpotP50 == nil && out.Summary.TPOTP50Ms > 0 {
		v50 := out.Summary.TPOTP50Ms
		tpotP50 = &v50
	}
	if tpotP90 == nil && out.Summary.TPOTP90Ms > 0 {
		v90 := out.Summary.TPOTP90Ms
		tpotP90 = &v90
	}
	if tpotP99 == nil && out.Summary.TPOTP99Ms > 0 {
		v99 := out.Summary.TPOTP99Ms
		tpotP99 = &v99
	}
	prefillP50, _, _, _ := percentiles(prefills)
	decodeP50, _, _, _ := percentiles(decodes)
	queueP50, _, _, _ := percentiles(queues)

	// Per-request throughput: average output tokens / average duration.
	var throughputPerRequest *float64
	if len(successful) > 0 {
		avgTokens := float64(totalOutputTokens) / float64(len(successful))
		avgDur := mean(extractField(successful, func(r RequestResult) float64 { return r.DurationSeconds }))
		if avgDur > 0 {
			v := avgTokens / avgDur
			throughputPerRequest = &v
		}
	}

	// Handle both field naming conventions for aggregate TPS
	aggTPSVal := out.Summary.ThroughputAggregateTPS
	if aggTPSVal == 0 {
		aggTPSVal = out.Summary.ThroughputTPS // inference-perf field
	}
	aggTPS := &aggTPSVal

	rps := &out.Summary.RequestsPerSecond

	// Handle both field naming conventions for duration
	durVal := out.Summary.TotalDurationSeconds
	if durVal == 0 {
		durVal = out.Summary.TotalDurationS // inference-perf field
	}
	dur := &durVal

	successCount := out.Summary.SuccessfulRequests
	failCount := out.Summary.FailedRequests

	// Input throughput (prompt tokens/sec) from inference-perf
	var inputTPS *float64
	if out.Summary.InputThroughputTPS > 0 {
		inputTPS = &out.Summary.InputThroughputTPS
	}

	// Output length mean from inference-perf
	var outputLenMean *float64
	if out.Summary.OutputLengthMean > 0 {
		outputLenMean = &out.Summary.OutputLengthMean
	}

	// GenerationThroughputTPS is the same as ThroughputAggregateTPS (output tokens/sec).
	// We set both for consistency between GPU scraper metrics and loadgen metrics.
	var genTPS *float64
	if aggTPS != nil && *aggTPS > 0 {
		genTPS = aggTPS
	}

	return &database.BenchmarkMetrics{
		TTFTP50Ms:               ttftP50,
		TTFTP90Ms:               ttftP90,
		TTFTP95Ms:               ttftP95,
		TTFTP99Ms:               ttftP99,
		E2ELatencyP50Ms:         e2eP50,
		E2ELatencyP90Ms:         e2eP90,
		E2ELatencyP95Ms:         e2eP95,
		E2ELatencyP99Ms:         e2eP99,
		ITLP50Ms:                itlP50,
		ITLP90Ms:                itlP90,
		ITLP95Ms:                itlP95,
		ITLP99Ms:                itlP99,
		TPOTP50Ms:               tpotP50,
		TPOTP90Ms:               tpotP90,
		TPOTP99Ms:               tpotP99,
		PrefillTimeP50Ms:        prefillP50,
		DecodeTimeP50Ms:         decodeP50,
		QueueTimeP50Ms:          queueP50,
		ThroughputPerRequestTPS: throughputPerRequest,
		ThroughputAggregateTPS:  aggTPS,
		RequestsPerSecond:       rps,
		AcceleratorUtilizationPct: out.Summary.AcceleratorUtilizationPct,
		AcceleratorMemoryPeakGiB:  out.Summary.AcceleratorMemoryPeakGiB,
		SuccessfulRequests:        &successCount,
		FailedRequests:            &failCount,
		TotalDurationSeconds:      dur,
		PromptThroughputTPS:       inputTPS,
		GenerationThroughputTPS:   genTPS,
		OutputLengthMean:          outputLenMean,
	}
}

// percentiles computes p50, p90, p95, p99 from a slice of float64 values.
// Returns nil pointers if the slice is empty.
func percentiles(vals []float64) (p50, p90, p95, p99 *float64) {
	if len(vals) == 0 {
		return nil, nil, nil, nil
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	p50v := percentile(sorted, 50)
	p90v := percentile(sorted, 90)
	p95v := percentile(sorted, 95)
	p99v := percentile(sorted, 99)
	return &p50v, &p90v, &p95v, &p99v
}

// percentile computes the p-th percentile from a sorted slice using
// the nearest-rank method.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := (p / 100.0) * float64(len(sorted))
	idx := int(math.Ceil(rank)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func filterSuccessful(results []RequestResult) []RequestResult {
	var out []RequestResult
	for _, r := range results {
		if r.Success {
			out = append(out, r)
		}
	}
	return out
}

func extractField(results []RequestResult, f func(RequestResult) float64) []float64 {
	vals := make([]float64, len(results))
	for i, r := range results {
		vals[i] = f(r)
	}
	return vals
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
