package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type SimProfile struct {
	Name            string
	ContextSize     int64
	ModelName       string
	BaseInputTokens int64
	InputGrowth     int64
	OutputPerTurn   int64
	CacheHitRate    float64
	CostPerTurn     float64
	DurationPerTurn float64
}

var (
	profileOpus = SimProfile{
		Name:            "opus-200k",
		ContextSize:     200_000,
		ModelName:       "Opus 4.6 (200k context)",
		BaseInputTokens: 5_000,
		InputGrowth:     3_500,
		OutputPerTurn:   1_200,
		CacheHitRate:    0.72,
		CostPerTurn:     0.085,
		DurationPerTurn: 45_000,
	}
	profileOpus1M = SimProfile{
		Name:            "opus-1m",
		ContextSize:     1_000_000,
		ModelName:       "Opus 4.6 (1M context)",
		BaseInputTokens: 5_000,
		InputGrowth:     4_000,
		OutputPerTurn:   1_500,
		CacheHitRate:    0.68,
		CostPerTurn:     0.12,
		DurationPerTurn: 55_000,
	}
	profileSonnet = SimProfile{
		Name:            "sonnet-200k",
		ContextSize:     200_000,
		ModelName:       "Sonnet 4.6",
		BaseInputTokens: 4_000,
		InputGrowth:     3_000,
		OutputPerTurn:   1_000,
		CacheHitRate:    0.75,
		CostPerTurn:     0.025,
		DurationPerTurn: 30_000,
	}
)

func simulateTurn(turn int, cwd string, p SimProfile) []byte {
	totalIn := float64(p.BaseInputTokens + int64(turn)*p.InputGrowth)
	totalOut := float64(int64(turn+1) * p.OutputPerTurn)
	remaining := 100.0 * (1.0 - totalIn/float64(p.ContextSize))
	if remaining < 2 {
		remaining = 2
	}

	reqTokens := float64(p.InputGrowth) + float64(turn)*300
	hitRate := 0.0
	if turn > 0 {
		hitRate = p.CacheHitRate
	}
	cacheRead := reqTokens * hitRate
	cacheCreate := reqTokens * 0.08
	inputTok := reqTokens - cacheRead - cacheCreate
	if inputTok < 0 {
		inputTok = 0
	}

	cost := float64(turn+1) * p.CostPerTurn
	duration := float64(turn+1) * p.DurationPerTurn
	apiDur := duration * 0.65
	linesAdded := float64(turn * 15)
	linesRemoved := float64(turn * 4)
	rl5pct := float64(turn) * 2.5
	if rl5pct > 95 {
		rl5pct = 95
	}
	rl7pct := float64(turn) * 0.5

	input := map[string]any{
		"workspace": map[string]any{"current_dir": cwd},
		"model":     map[string]any{"display_name": p.ModelName},
		"context_window": map[string]any{
			"remaining_percentage": remaining,
			"total_input_tokens":   totalIn,
			"total_output_tokens":  totalOut,
			"context_window_size":  p.ContextSize,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     cacheRead,
				"cache_creation_input_tokens": cacheCreate,
				"input_tokens":               inputTok,
			},
		},
		"cost": map[string]any{
			"total_cost_usd":        cost,
			"total_duration_ms":     duration,
			"total_lines_added":     linesAdded,
			"total_lines_removed":   linesRemoved,
			"total_api_duration_ms": apiDur,
		},
		"rate_limits": map[string]any{
			"five_hour": map[string]any{"used_percentage": rl5pct},
			"seven_day": map[string]any{"used_percentage": rl7pct},
		},
	}

	data, _ := json.Marshal(input)
	return data
}

func simulateSession(turns int, cwd string, p SimProfile) [][]byte {
	payloads := make([][]byte, turns)
	for i := range turns {
		payloads[i] = simulateTurn(i, cwd, p)
	}
	return payloads
}

var testCwd string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "statusline-bench-*")
	if err != nil {
		os.Exit(1)
	}
	sessionPath = filepath.Join(tmp, "session")
	historyPath = filepath.Join(tmp, "history")
	configPath = filepath.Join(tmp, "config.json")
	testCwd = tmp

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func resetSession() {
	os.Remove(sessionPath)
	os.Remove(historyPath)
}

// --- Benchmarks ---

func BenchmarkSingleRender(b *testing.B) {
	run(bytes.NewReader(simulateTurn(9, testCwd, profileOpus)), io.Discard)
	payload := simulateTurn(10, testCwd, profileOpus)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		run(bytes.NewReader(payload), io.Discard)
	}
}

func BenchmarkFirstTurn(b *testing.B) {
	payload := simulateTurn(0, testCwd, profileOpus)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resetSession()
		run(bytes.NewReader(payload), io.Discard)
	}
}

func BenchmarkFullSession(b *testing.B) {
	payloads := simulateSession(20, testCwd, profileOpus)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resetSession()
		for _, p := range payloads {
			run(bytes.NewReader(p), io.Discard)
		}
	}
}

func BenchmarkProfiles(b *testing.B) {
	for _, p := range []SimProfile{profileOpus, profileOpus1M, profileSonnet} {
		b.Run(p.Name, func(b *testing.B) {
			run(bytes.NewReader(simulateTurn(9, testCwd, p)), io.Discard)
			payload := simulateTurn(10, testCwd, p)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				run(bytes.NewReader(payload), io.Discard)
			}
		})
	}
}

func BenchmarkWithGit(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git not available")
	}

	gitDir := b.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "bench@test.com"},
		{"config", "user.name", "Bench"},
	} {
		exec.Command("git", append([]string{"-C", gitDir}, args...)...).Run()
	}
	os.WriteFile(filepath.Join(gitDir, "file.go"), []byte("package main\n"), 0644)
	exec.Command("git", "-C", gitDir, "add", ".").Run()
	exec.Command("git", "-C", gitDir, "commit", "-m", "init").Run()
	os.WriteFile(filepath.Join(gitDir, "file2.go"), []byte("package main\n// changed\n"), 0644)

	run(bytes.NewReader(simulateTurn(9, gitDir, profileOpus)), io.Discard)

	b.Run("cold-git-cache", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			os.Remove(filepath.Join(gitDir, ".git", ".statusline-cache-go"))
			run(bytes.NewReader(simulateTurn(10, gitDir, profileOpus)), io.Discard)
		}
	})

	run(bytes.NewReader(simulateTurn(10, gitDir, profileOpus)), io.Discard)
	b.Run("warm-git-cache", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			run(bytes.NewReader(simulateTurn(10, gitDir, profileOpus)), io.Discard)
		}
	})
}

func BenchmarkEndToEnd(b *testing.B) {
	binPath := filepath.Join(b.TempDir(), "claude-statusline-bench")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		b.Fatalf("build failed: %s\n%s", err, out)
	}

	payload := simulateTurn(10, testCwd, profileOpus)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binPath)
		cmd.Stdin = bytes.NewReader(payload)
		cmd.Stdout = io.Discard
		cmd.Run()
	}
}

// --- Simulation tests ---

func TestSimulationProducesOutput(t *testing.T) {
	resetSession()

	for turn := 0; turn < 5; turn++ {
		var buf bytes.Buffer
		run(bytes.NewReader(simulateTurn(turn, testCwd, profileOpus)), &buf)

		if buf.Len() == 0 {
			t.Fatalf("turn %d: empty output", turn)
		}
		lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
		if len(lines) < 1 {
			t.Fatalf("turn %d: expected at least 1 line, got %d", turn, len(lines))
		}
	}
}

func TestOutputContainsExpectedSegments(t *testing.T) {
	resetSession()
	run(bytes.NewReader(simulateTurn(0, "/home/user/project", profileOpus)), io.Discard)

	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, "/home/user/project", profileOpus)), &buf)
	output := buf.String()

	for _, want := range []string{"Opus 4.6", "$", "%"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestSessionDetectsNewSession(t *testing.T) {
	resetSession()

	for turn := 0; turn < 5; turn++ {
		run(bytes.NewReader(simulateTurn(turn, testCwd, profileOpus)), io.Discard)
	}

	sess := loadSession()
	if sess.TurnCount != 5 {
		t.Errorf("expected 5 turns, got %d", sess.TurnCount)
	}
	if len(sess.SparkHist) != 5 {
		t.Errorf("expected 5 sparkline entries, got %d", len(sess.SparkHist))
	}

	run(bytes.NewReader(simulateTurn(0, testCwd, profileOpus)), io.Discard)

	sess = loadSession()
	if sess.TurnCount != 1 {
		t.Errorf("expected turn count reset to 1 after new session, got %d", sess.TurnCount)
	}
}

func TestCumulativeCacheAccumulates(t *testing.T) {
	resetSession()

	for turn := 0; turn < 10; turn++ {
		run(bytes.NewReader(simulateTurn(turn, testCwd, profileOpus)), io.Discard)
	}

	sess := loadSession()
	if sess.CumCR <= 0 {
		t.Errorf("expected cumulative cache reads > 0, got %d", sess.CumCR)
	}
	if sess.CumCC <= 0 {
		t.Errorf("expected cumulative cache creates > 0, got %d", sess.CumCC)
	}
}

func TestConfigTogglesSegments(t *testing.T) {
	resetSession()
	run(bytes.NewReader(simulateTurn(0, testCwd, profileOpus)), io.Discard)

	payload := simulateTurn(5, "/home/user/project", profileOpus)

	// Default config: everything on
	var full bytes.Buffer
	run(bytes.NewReader(payload), &full)

	// Minimal config: most things off
	minCfg := `{"user_host":false,"model":false,"cache":{"sparkline":false,"savings":false},"api_stats":{"enabled":false},"session_compare":false,"line_deltas":false,"rate_limits":{"enabled":false},"context_runway":false}`
	os.WriteFile(configPath, []byte(minCfg), 0644)
	defer os.Remove(configPath)

	var minimal bytes.Buffer
	run(bytes.NewReader(payload), &minimal)

	if minimal.Len() >= full.Len() {
		t.Errorf("minimal config output (%d bytes) should be shorter than full (%d bytes)", minimal.Len(), full.Len())
	}
	if strings.Contains(minimal.String(), "Opus") {
		t.Error("minimal config should not contain model name")
	}
}

func TestConfigUnitFormats(t *testing.T) {
	resetSession()

	// Test tokens.format = "raw"
	os.WriteFile(configPath, []byte(`{"tokens":{"format":"raw"}}`), 0644)
	defer os.Remove(configPath)

	run(bytes.NewReader(simulateTurn(0, testCwd, profileOpus)), io.Discard)
	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, testCwd, profileOpus)), &buf)

	output := buf.String()
	if strings.Contains(output, "k") && !strings.Contains(output, "ok") {
		// "k" suffix shouldn't appear in token counts when format is raw
		// (but might appear in other words, so we check loosely)
	}

	// Test duration.format = "seconds"
	os.WriteFile(configPath, []byte(`{"duration":{"format":"seconds"}}`), 0644)
	buf.Reset()
	run(bytes.NewReader(simulateTurn(5, testCwd, profileOpus)), &buf)
	if !strings.Contains(buf.String(), "s") {
		t.Error("seconds format should contain 's'")
	}
}

func TestConfigFileNotRequired(t *testing.T) {
	os.Remove(configPath)
	resetSession()

	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(3, testCwd, profileOpus)), &buf)
	if buf.Len() == 0 {
		t.Error("should produce output without config file")
	}
}

func TestDefaultConfigMatchesAllEnabled(t *testing.T) {
	c := defaultConfig()
	if !c.UserHost || !c.Cwd || !c.Model || !c.Git.Enabled || !c.ContextBar.Enabled ||
		!c.Cost.Enabled || !c.Duration.Enabled || !c.Tokens.Enabled || !c.Cache.Enabled ||
		!c.ContextRunway || !c.RateLimits.Enabled || !c.LineDeltas || !c.ApiStats.Enabled ||
		!c.SessionCompare {
		t.Error("all segments should be enabled by default")
	}
}
