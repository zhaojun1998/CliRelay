package usage

import (
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func approxEqualFloat(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-9
}

func TestQueryLatencyThroughputAggregates(t *testing.T) {
	initTestUsageDB(t, config.RequestLogStorageConfig{StoreContent: false})

	now := time.Now().UTC()
	// A: latency 1000ms, ttfb 200ms, output 100 -> 100 tok/s
	InsertLog("", "", "m", "codex", "codex", "auth-1", false, now, 1000, 200, TokenStats{OutputTokens: 100, TotalTokens: 120}, "", "")
	// B: latency 2000ms, ttfb 400ms, output 100 -> 50 tok/s
	InsertLog("", "", "m", "codex", "codex", "auth-1", false, now, 2000, 400, TokenStats{OutputTokens: 100, TotalTokens: 130}, "", "")
	// C: latency 500ms, ttfb 100ms, output 100 -> 200 tok/s
	InsertLog("", "", "m", "codex", "codex", "auth-1", false, now, 500, 100, TokenStats{OutputTokens: 100, TotalTokens: 110}, "", "")
	// D: failed, no timing/tokens -> excluded from every metric (ttfb=0, latency=0)
	InsertLog("", "", "m", "codex", "codex", "auth-1", true, now, 0, 0, TokenStats{}, "", "")

	stats, err := QueryLatencyThroughput("", WindowFromDays(7))
	if err != nil {
		t.Fatalf("QueryLatencyThroughput() error = %v", err)
	}

	if want := (200.0 + 400 + 100) / 3; !approxEqualFloat(stats.AvgTTFBMs, want) {
		t.Fatalf("AvgTTFBMs = %v, want %v", stats.AvgTTFBMs, want)
	}
	if stats.MinTTFBMs != 100 {
		t.Fatalf("MinTTFBMs = %v, want 100", stats.MinTTFBMs)
	}
	if stats.MaxTTFBMs != 400 {
		t.Fatalf("MaxTTFBMs = %v, want 400", stats.MaxTTFBMs)
	}
	if stats.SampleCount != 3 {
		t.Fatalf("SampleCount = %d, want 3 (ttfb=0 rows excluded)", stats.SampleCount)
	}
	// overall throughput: sum(output)=300 over sum(latency)=3.5s
	if want := 300.0 * 1000.0 / 3500.0; !approxEqualFloat(stats.TokensPerSecond, want) {
		t.Fatalf("TokensPerSecond = %v, want %v", stats.TokensPerSecond, want)
	}
	// single-request rate extremes: B=50, C=200
	if stats.MinTokensPerSecond != 50 {
		t.Fatalf("MinTokensPerSecond = %v, want 50", stats.MinTokensPerSecond)
	}
	if stats.MaxTokensPerSecond != 200 {
		t.Fatalf("MaxTokensPerSecond = %v, want 200", stats.MaxTokensPerSecond)
	}
}

func TestQueryLatencyThroughputEmptyWindowReturnsZero(t *testing.T) {
	initTestUsageDB(t, config.RequestLogStorageConfig{StoreContent: false})

	now := time.Now().UTC()
	InsertLog("", "", "m", "codex", "codex", "auth-1", false, now, 1000, 200, TokenStats{OutputTokens: 100}, "", "")

	// Window fully in the past -> no rows -> all-zero stats, no NaN/divide-by-zero.
	past := TimeWindow{Start: now.AddDate(0, 0, -10), End: now.AddDate(0, 0, -9)}
	stats, err := QueryLatencyThroughput("", past)
	if err != nil {
		t.Fatalf("QueryLatencyThroughput() error = %v", err)
	}
	if stats.SampleCount != 0 || stats.AvgTTFBMs != 0 || stats.TokensPerSecond != 0 ||
		stats.MinTTFBMs != 0 || stats.MaxTTFBMs != 0 ||
		stats.MinTokensPerSecond != 0 || stats.MaxTokensPerSecond != 0 {
		t.Fatalf("empty window stats = %+v, want all zero", stats)
	}
}
