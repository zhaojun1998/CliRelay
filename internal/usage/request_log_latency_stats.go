package usage

import "fmt"

// LatencyThroughputStats holds latency (TTFB) and throughput (tokens/sec)
// aggregates over a time window, shared by the dashboard and monitor pages.
//
// TTFB comes from first_token_ms (request start -> first response chunk). Rows
// with first_token_ms = 0 (failed or non-streaming requests that never recorded
// a first chunk) are excluded via CASE so the average is not dragged down by
// placeholder zeros.
//
// TokensPerSecond is the overall output throughput: total output tokens divided
// by total request latency (seconds) across the window. Min/Max are the slowest
// and fastest single-request output rates; the weighted-average throughput
// always falls within [Min, Max].
type LatencyThroughputStats struct {
	AvgTTFBMs          float64 `json:"avg_ttfb_ms"`
	MinTTFBMs          float64 `json:"min_ttfb_ms"`
	MaxTTFBMs          float64 `json:"max_ttfb_ms"`
	TokensPerSecond    float64 `json:"tokens_per_second"`
	MinTokensPerSecond float64 `json:"min_tokens_per_second"`
	MaxTokensPerSecond float64 `json:"max_tokens_per_second"`
	SampleCount        int64   `json:"sample_count"`
}

// QueryLatencyThroughput aggregates TTFB and tokens/sec over the window. An
// empty apiKey aggregates across all keys (dashboard); a non-empty apiKey
// filters to that key (monitor), reusing buildWhereClause so key-matching
// semantics stay consistent with the other usage queries.
func QueryLatencyThroughput(apiKey string, win TimeWindow) (LatencyThroughputStats, error) {
	db := getDB()
	if db == nil {
		return LatencyThroughputStats{}, nil
	}

	where, args := buildWhereClause(LogQueryParams{APIKey: apiKey, Window: &win})

	// One pass, conditional aggregates so each metric carries its own filter
	// without affecting the others. tokens/sec is computed in Go from the two
	// sums to avoid a divide-by-zero in SQL.
	q := "SELECT " +
		"COALESCE(AVG(CASE WHEN first_token_ms > 0 THEN first_token_ms END), 0), " +
		"COALESCE(MIN(CASE WHEN first_token_ms > 0 THEN first_token_ms END), 0), " +
		"COALESCE(MAX(CASE WHEN first_token_ms > 0 THEN first_token_ms END), 0), " +
		"COALESCE(SUM(CASE WHEN latency_ms > 0 THEN output_tokens ELSE 0 END), 0), " +
		"COALESCE(SUM(CASE WHEN latency_ms > 0 THEN latency_ms ELSE 0 END), 0), " +
		"COALESCE(MIN(CASE WHEN latency_ms > 0 AND output_tokens > 0 THEN output_tokens * 1000.0 / latency_ms END), 0), " +
		"COALESCE(MAX(CASE WHEN latency_ms > 0 AND output_tokens > 0 THEN output_tokens * 1000.0 / latency_ms END), 0), " +
		"COUNT(CASE WHEN first_token_ms > 0 THEN 1 END) " +
		"FROM request_logs" + where

	var stats LatencyThroughputStats
	var sumOutputTokens, sumLatencyMs int64
	err := db.QueryRow(q, args...).Scan(
		&stats.AvgTTFBMs,
		&stats.MinTTFBMs,
		&stats.MaxTTFBMs,
		&sumOutputTokens,
		&sumLatencyMs,
		&stats.MinTokensPerSecond,
		&stats.MaxTokensPerSecond,
		&stats.SampleCount,
	)
	if err != nil {
		return LatencyThroughputStats{}, fmt.Errorf("usage: latency/throughput query: %w", err)
	}

	if sumLatencyMs > 0 {
		stats.TokensPerSecond = float64(sumOutputTokens) * 1000.0 / float64(sumLatencyMs)
	}

	return stats, nil
}
