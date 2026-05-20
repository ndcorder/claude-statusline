package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- format.go ---

func TestFmtTokensAllRanges(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{500, "500"},
		{1500, "1.5k"},
		{15000, "15k"},
		{1500000, "1.5M"},
	}
	for _, tt := range tests {
		if got := fmtTokens(tt.in); got != tt.want {
			t.Errorf("fmtTokens(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFmtTokensUnitAllFormats(t *testing.T) {
	tests := []struct {
		n      int64
		fmt    string
		want   string
	}{
		{5000, "auto", "5.0k"},
		{5000, "raw", "5000"},
		{5000, "k", "5.0k"},
		{500, "k", "500"},
		{5000, "M", "0.01M"},
	}
	for _, tt := range tests {
		if got := fmtTokensUnit(tt.n, tt.fmt); got != tt.want {
			t.Errorf("fmtTokensUnit(%d, %q) = %q, want %q", tt.n, tt.fmt, got, tt.want)
		}
	}
}

func TestFmtDurationAllFormats(t *testing.T) {
	tests := []struct {
		ms     float64
		fmt    string
		want   string
	}{
		{45000, "auto", "45s"},
		{90000, "auto", "1m30s"},
		{3700000, "auto", "1h1m"},
		{90000, "seconds", "90s"},
		{90000, "minutes", "1.5m"},
		{3600000, "hours", "1.00h"},
		{90000, "bogus", "1m30s"},
	}
	for _, tt := range tests {
		if got := fmtDuration(tt.ms, tt.fmt); got != tt.want {
			t.Errorf("fmtDuration(%.0f, %q) = %q, want %q", tt.ms, tt.fmt, got, tt.want)
		}
	}
}

func TestDerefNil(t *testing.T) {
	if deref(nil) != 0 {
		t.Error("deref(nil) should be 0")
	}
}

// --- config.go ---

func TestLoadConfigFromDefaultPath(t *testing.T) {
	old := configPath
	configPath = ""
	defer func() { configPath = old }()

	c := loadConfig()
	if !c.UserHost {
		t.Error("should return defaults")
	}
}

func TestLoadConfigHomeDirError(t *testing.T) {
	oldPath := configPath
	oldHome := userHomeDir
	configPath = ""
	userHomeDir = func() (string, error) { return "", fmt.Errorf("no home") }
	defer func() { configPath = oldPath; userHomeDir = oldHome }()

	c := loadConfig()
	if !c.UserHost {
		t.Error("should return defaults when home dir unavailable")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	os.WriteFile(configPath, []byte("{not json"), 0644)
	defer os.Remove(configPath)

	c := loadConfig()
	if !c.UserHost {
		t.Error("should return defaults for invalid JSON")
	}
}

// --- git.go ---

func TestGetGitInfoNoRepo(t *testing.T) {
	if getGitInfo(t.TempDir()) != nil {
		t.Error("expected nil for non-git directory")
	}
}

func TestGetGitInfoComplete(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	gitRun := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Helper()
			t.Fatalf("git %v: %s\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	gitRun("init")
	gitRun("config", "user.email", "test@test.com")
	gitRun("config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644)
	gitRun("add", ".")
	gitRun("commit", "-m", "init")

	// Clean repo
	gi := getGitInfo(dir)
	if gi == nil {
		t.Fatal("expected git info")
	}
	if gi.Branch == "" {
		t.Error("expected branch")
	}
	if gi.Changed != 0 {
		t.Error("clean repo should have 0 changes")
	}

	// Dirty repo
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello world\nnew line\n"), 0644)
	os.Remove(filepath.Join(dir, ".git", ".statusline-cache-go"))

	gi = getGitInfo(dir)
	if gi.Changed != 1 {
		t.Errorf("expected 1 changed file, got %d", gi.Changed)
	}
	if gi.Added <= 0 {
		t.Error("expected added lines > 0")
	}

	// Detached HEAD
	hash := gitRun("rev-parse", "HEAD")
	gitRun("checkout", hash)
	os.Remove(filepath.Join(dir, ".git", ".statusline-cache-go"))

	gi = getGitInfo(dir)
	if gi == nil || gi.Branch == "" {
		t.Error("detached HEAD should show short hash")
	}

	// Worktree (absolute gitDir path)
	gitRun("checkout", "-b", "main")
	wtDir := filepath.Join(dir, "..", "wt-test")
	gitRun("worktree", "add", wtDir, "-b", "wt-branch")
	defer func() {
		exec.Command("git", "-C", dir, "worktree", "remove", "--force", wtDir).Run()
	}()

	gi = getGitInfo(wtDir)
	if gi == nil {
		t.Error("expected git info from worktree")
	}
}

func TestGetGitInfoWithRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Create a bare "remote" repo
	remoteDir := t.TempDir()
	exec.Command("git", "init", "--bare", remoteDir).Run()

	// Create a local clone
	localDir := filepath.Join(t.TempDir(), "local")
	exec.Command("git", "clone", remoteDir, localDir).Run()
	exec.Command("git", "-C", localDir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", localDir, "config", "user.name", "T").Run()

	// Create commits and push
	os.WriteFile(filepath.Join(localDir, "f.txt"), []byte("a"), 0644)
	exec.Command("git", "-C", localDir, "add", ".").Run()
	exec.Command("git", "-C", localDir, "commit", "-m", "first").Run()
	exec.Command("git", "-C", localDir, "push", "origin", "HEAD").Run()

	// Add local commits ahead of remote
	os.WriteFile(filepath.Join(localDir, "f.txt"), []byte("b"), 0644)
	exec.Command("git", "-C", localDir, "add", ".").Run()
	exec.Command("git", "-C", localDir, "commit", "-m", "ahead").Run()

	os.Remove(filepath.Join(localDir, ".git", ".statusline-cache-go"))
	gi := getGitInfo(localDir)
	if gi == nil {
		t.Fatal("expected git info")
	}
	if gi.Ahead != 1 {
		t.Errorf("expected 1 ahead, got %d", gi.Ahead)
	}
}

func TestGetGitInfoCacheHit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "T").Run()
	os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "i").Run()

	gi1 := getGitInfo(dir)
	gi2 := getGitInfo(dir)
	if gi1 == nil || gi2 == nil {
		t.Fatal("both calls should return git info")
	}
	if gi1.Branch != gi2.Branch {
		t.Error("cached result should match")
	}
}

func TestGetGitInfoCacheInvalid(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "T").Run()
	os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "i").Run()

	// Write invalid cache
	os.WriteFile(filepath.Join(dir, ".git", ".statusline-cache-go"), []byte("not json"), 0644)

	gi := getGitInfo(dir)
	if gi == nil {
		t.Error("should recover from invalid cache")
	}
}

// --- session.go ---

func TestLoadSessionInvalidJSON(t *testing.T) {
	os.WriteFile(sessionPath, []byte("not json"), 0644)
	defer os.Remove(sessionPath)

	sess := loadSession()
	if sess.TurnCount != 0 {
		t.Error("invalid JSON should return empty session")
	}
}

func TestSessionUpdateSameTotalIn(t *testing.T) {
	s := Session{PrevTotalIn: 5000, TurnCount: 3}
	s.update(5000, 100, 50, 200)
	if s.TurnCount != 3 {
		t.Error("same totalIn should not change turn count")
	}
}

func TestAppendHistoryOpenError(t *testing.T) {
	old := historyPath
	historyPath = "/nonexistent/dir/history"
	defer func() { historyPath = old }()

	appendHistory(1.0, 1000, 5000, 1000, 5)
}

func TestAppendHistoryTruncation(t *testing.T) {
	resetSession()
	max := cfg.Session.MaxHistory
	for i := 0; i < max+10; i++ {
		appendHistory(float64(i)*0.01, 1000, 5000, 1000, 5)
	}

	data, _ := os.ReadFile(historyPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > max {
		t.Errorf("should truncate to %d, got %d", max, len(lines))
	}
}

func TestAppendHistoryReadError(t *testing.T) {
	old := historyPath
	dir := t.TempDir()
	historyPath = filepath.Join(dir, "history")
	defer func() { historyPath = old }()

	appendHistory(1.0, 1000, 5000, 1000, 5)
	os.Chmod(historyPath, 0200)
	defer os.Chmod(historyPath, 0644)
	appendHistory(2.0, 2000, 10000, 2000, 10)
}

func TestLoadHistoryAvgEdgeCases(t *testing.T) {
	resetSession()

	if loadHistoryAvg() != nil {
		t.Error("no file should return nil")
	}

	os.WriteFile(historyPath, []byte(""), 0644)
	if loadHistoryAvg() != nil {
		t.Error("empty file should return nil")
	}

	os.WriteFile(historyPath, []byte("not-a-number\n"), 0644)
	if avg := loadHistoryAvg(); avg != nil && avg.Count > 0 {
		t.Error("unparseable entries should yield count 0")
	}

	os.WriteFile(historyPath, []byte("0.5000|45000|10000|2000|5\n1.0000|90000|20000|4000|10\n"), 0644)
	avg := loadHistoryAvg()
	if avg == nil || avg.Count != 2 {
		t.Fatal("expected 2 entries")
	}
	if avg.AvgCost != 0.75 {
		t.Errorf("expected 0.75, got %f", avg.AvgCost)
	}
}

// --- main.go ---

type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) { return 0, fmt.Errorf("fail") }

func TestMainVersion(t *testing.T) {
	var buf bytes.Buffer
	cliMain([]string{"claude-statusline", "--version"}, nil, &buf)
	if !strings.Contains(buf.String(), "claude-statusline") {
		t.Errorf("--version should print program name, got %q", buf.String())
	}
}

func TestMainInitConfig(t *testing.T) {
	var buf bytes.Buffer
	cliMain([]string{"claude-statusline", "--init-config"}, nil, &buf)

	var c Config
	if err := json.Unmarshal(buf.Bytes(), &c); err != nil {
		t.Fatalf("--init-config output is not valid JSON: %v", err)
	}
	if !c.UserHost || !c.Cache.Enabled {
		t.Error("should output default config with all features enabled")
	}
}

func TestCliMainDelegatesToRun(t *testing.T) {
	resetSession()
	var buf bytes.Buffer
	cliMain([]string{"claude-statusline"}, bytes.NewReader(simulateTurn(3, testCwd, profileOpus)), &buf)
	if buf.Len() == 0 {
		t.Error("cliMain with no flags should delegate to run()")
	}
}

func TestMainFunc(t *testing.T) {
	resetSession()
	var buf bytes.Buffer
	cliMain(
		[]string{"claude-statusline"},
		bytes.NewReader(simulateTurn(3, testCwd, profileOpus)),
		&buf,
	)
	out := buf.Bytes()

	if len(out) == 0 {
		t.Error("cliMain should produce output")
	}
}

func TestRunReadError(t *testing.T) {
	run(&errorReader{}, io.Discard)
}

func TestRunEmptyInput(t *testing.T) {
	var buf bytes.Buffer
	run(bytes.NewReader([]byte{}), &buf)
}

func TestRunInvalidJSONInput(t *testing.T) {
	resetSession()
	var buf bytes.Buffer
	run(bytes.NewReader([]byte("{not valid json}")), &buf)
	// Should not panic; warning goes to os.Stderr
}

func TestRunSessionPathDefaults(t *testing.T) {
	// Save and clear session/history paths to exercise the
	// "if sessionPath == ''" and "if historyPath == ''" guards in run()
	oldSP := sessionPath
	oldHP := historyPath
	sessionPath = ""
	historyPath = ""
	defer func() { sessionPath = oldSP; historyPath = oldHP }()

	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(3, testCwd, profileOpus)), &buf)
	if buf.Len() == 0 {
		t.Error("should produce output when session paths come from config defaults")
	}

	// Restore for other tests
	sessionPath = oldSP
	historyPath = oldHP
}

func TestRunCwdFallbackToCwd(t *testing.T) {
	resetSession()
	input := map[string]any{
		"cwd":   "/fallback/cwd",
		"model": map[string]any{"display_name": "Test"},
		"context_window": map[string]any{
			"remaining_percentage": 80.0,
			"total_input_tokens":   5000.0,
			"total_output_tokens":  1000.0,
			"context_window_size":  200000.0,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     100.0,
				"cache_creation_input_tokens": 50.0,
				"input_tokens":               200.0,
			},
		},
	}
	data, _ := json.Marshal(input)
	var buf bytes.Buffer
	run(bytes.NewReader(data), &buf)
	if !strings.Contains(buf.String(), "/fallback/cwd") {
		t.Error("should use cwd field as fallback")
	}
}

func TestRunCwdFallbackToGetwd(t *testing.T) {
	resetSession()
	input := map[string]any{
		"model": map[string]any{"display_name": "Test"},
		"context_window": map[string]any{
			"remaining_percentage": 80.0,
			"total_input_tokens":   5000.0,
			"total_output_tokens":  1000.0,
			"context_window_size":  200000.0,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     100.0,
				"cache_creation_input_tokens": 50.0,
				"input_tokens":               200.0,
			},
		},
	}
	data, _ := json.Marshal(input)
	var buf bytes.Buffer
	run(bytes.NewReader(data), &buf)
	if buf.Len() == 0 {
		t.Error("should produce output with os.Getwd fallback")
	}
}

func TestRunCostVelocityMinute(t *testing.T) {
	resetSession()
	os.WriteFile(configPath, []byte(`{"cost":{"velocity_unit":"minute"}}`), 0644)
	defer os.Remove(configPath)

	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, testCwd, profileOpus)), &buf)
	if !strings.Contains(buf.String(), "/min") {
		t.Error("should show $/min")
	}
}

func TestRunNonCumulativeCache(t *testing.T) {
	resetSession()
	os.WriteFile(configPath, []byte(`{"cache":{"cumulative":false}}`), 0644)
	defer os.Remove(configPath)

	run(bytes.NewReader(simulateTurn(0, testCwd, profileOpus)), io.Discard)
	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, testCwd, profileOpus)), &buf)
	if !strings.Contains(buf.String(), "cache") {
		t.Error("should show per-request cache stats")
	}
}

func TestRunContextBarColors(t *testing.T) {
	resetSession()

	makeInput := func(remaining float64) []byte {
		in := map[string]any{
			"workspace": map[string]any{"current_dir": testCwd},
			"model":     map[string]any{"display_name": "Test"},
			"context_window": map[string]any{
				"remaining_percentage": remaining,
				"total_input_tokens":   5000.0,
				"total_output_tokens":  1000.0,
				"context_window_size":  200000.0,
				"current_usage": map[string]any{
					"cache_read_input_tokens":     100.0,
					"cache_creation_input_tokens": 50.0,
					"input_tokens":               200.0,
				},
			},
			"cost": map[string]any{
				"total_cost_usd":    0.5,
				"total_duration_ms": 30000.0,
			},
		}
		d, _ := json.Marshal(in)
		return d
	}

	for _, tc := range []struct {
		remaining float64
		color     string
	}{
		{5, Red},
		{20, Yellow},
		{60, Green},
	} {
		var buf bytes.Buffer
		run(bytes.NewReader(makeInput(tc.remaining)), &buf)
		if !strings.Contains(buf.String(), tc.color) {
			t.Errorf("remaining %.0f%% should use color %q", tc.remaining, tc.color)
		}
	}
}

func TestRunHaikuPricing(t *testing.T) {
	resetSession()
	in := map[string]any{
		"workspace": map[string]any{"current_dir": testCwd},
		"model":     map[string]any{"display_name": "Haiku 4.5"},
		"context_window": map[string]any{
			"remaining_percentage": 80.0,
			"total_input_tokens":   10000.0,
			"total_output_tokens":  2000.0,
			"context_window_size":  200000.0,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     5000.0,
				"cache_creation_input_tokens": 500.0,
				"input_tokens":               200.0,
			},
		},
		"cost": map[string]any{
			"total_cost_usd":        0.1,
			"total_duration_ms":     90000.0,
			"total_api_duration_ms": 60000.0,
		},
	}
	data, _ := json.Marshal(in)
	run(bytes.NewReader(data), io.Discard)

	var buf bytes.Buffer
	run(bytes.NewReader(data), &buf)
	if !strings.Contains(buf.String(), "saved") {
		t.Error("haiku model should show savings")
	}
}

func TestRunNegativeSavings(t *testing.T) {
	resetSession()
	// Huge cache creation, tiny cache read → negative savings
	in := map[string]any{
		"workspace": map[string]any{"current_dir": testCwd},
		"model":     map[string]any{"display_name": "Sonnet 4.6"},
		"context_window": map[string]any{
			"remaining_percentage": 80.0,
			"total_input_tokens":   10000.0,
			"total_output_tokens":  2000.0,
			"context_window_size":  200000.0,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     10.0,
				"cache_creation_input_tokens": 50000.0,
				"input_tokens":               200.0,
			},
		},
	}
	data, _ := json.Marshal(in)
	run(bytes.NewReader(data), io.Discard)

	var buf bytes.Buffer
	run(bytes.NewReader(data), &buf)
	if !strings.Contains(buf.String(), Red) {
		t.Error("negative savings should show red")
	}
}

func TestRunSparklineOverflow(t *testing.T) {
	resetSession()
	os.WriteFile(configPath, []byte(`{"cache":{"sparkline_width":3}}`), 0644)
	defer os.Remove(configPath)

	// Run enough turns to fill sparkline beyond width
	for i := 0; i < 8; i++ {
		run(bytes.NewReader(simulateTurn(i, testCwd, profileOpus)), io.Discard)
	}
	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(8, testCwd, profileOpus)), &buf)
	// Output should still work with truncated sparkline
	if buf.Len() == 0 {
		t.Error("should produce output with sparkline overflow")
	}
}

func TestRunRateLimitsAllResetFormats(t *testing.T) {
	resetSession()
	now := float64(time.Now().Unix())

	for _, tc := range []struct {
		name   string
		offset float64
		want   string
	}{
		{"seconds", 30, "s"},
		{"minutes", 300, "m"},
		{"hours", 5000, "h"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := map[string]any{
				"workspace": map[string]any{"current_dir": testCwd},
				"model":     map[string]any{"display_name": "Test"},
				"context_window": map[string]any{
					"remaining_percentage": 80.0,
					"total_input_tokens":   5000.0,
					"total_output_tokens":  1000.0,
					"context_window_size":  200000.0,
					"current_usage": map[string]any{
						"cache_read_input_tokens":     100.0,
						"cache_creation_input_tokens": 50.0,
						"input_tokens":               200.0,
					},
				},
				"rate_limits": map[string]any{
					"five_hour": map[string]any{
						"used_percentage": 45.0,
						"resets_at":       now + tc.offset,
					},
					"seven_day": map[string]any{
						"used_percentage": 12.0,
					},
				},
			}
			data, _ := json.Marshal(in)
			var buf bytes.Buffer
			run(bytes.NewReader(data), &buf)
			if !strings.Contains(buf.String(), tc.want) {
				t.Errorf("reset format should contain %q", tc.want)
			}
		})
	}
}

func TestRunSessionCompare(t *testing.T) {
	resetSession()
	os.WriteFile(historyPath, []byte("0.5000|45000|10000|2000|5\n"), 0644)

	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, testCwd, profileOpus)), &buf)
	if !strings.Contains(buf.String(), "avg") {
		t.Error("should show session comparison")
	}
}

func TestRunContextRunwayColors(t *testing.T) {
	resetSession()

	// First prime a few turns to build up avgGrowth
	for i := 0; i < 3; i++ {
		run(bytes.NewReader(simulateTurn(i, testCwd, profileOpus)), io.Discard)
	}

	// Low remaining → few estimated turns → red
	in := map[string]any{
		"workspace": map[string]any{"current_dir": testCwd},
		"model":     map[string]any{"display_name": "Opus 4.6"},
		"context_window": map[string]any{
			"remaining_percentage": 3.0,
			"total_input_tokens":   194000.0,
			"total_output_tokens":  5000.0,
			"context_window_size":  200000.0,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     100.0,
				"cache_creation_input_tokens": 50.0,
				"input_tokens":               200.0,
			},
		},
	}
	data, _ := json.Marshal(in)
	var buf bytes.Buffer
	run(bytes.NewReader(data), &buf)
	if !strings.Contains(buf.String(), "turns") {
		t.Error("should show runway estimate")
	}
}

func TestRunGitBehind(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	resetSession()

	// Create remote with 2 commits, clone, then reset local back 1
	remoteDir := t.TempDir()
	exec.Command("git", "init", "--bare", remoteDir).Run()
	dir := filepath.Join(t.TempDir(), "local")
	exec.Command("git", "clone", remoteDir, dir).Run()
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "T").Run()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "first").Run()
	exec.Command("git", "-C", dir, "push", "origin", "HEAD").Run()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("b"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "second").Run()
	exec.Command("git", "-C", dir, "push", "origin", "HEAD").Run()

	// Reset local back 1 commit so it's behind
	exec.Command("git", "-C", dir, "reset", "--hard", "HEAD~1").Run()

	os.Remove(filepath.Join(dir, ".git", ".statusline-cache-go"))
	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(3, dir, profileOpus)), &buf)
	if !strings.Contains(buf.String(), "↓") {
		t.Error("should show behind indicator")
	}
}

func TestRunSavingsDecimalPadding(t *testing.T) {
	resetSession()
	// Craft values so net savings = 0.100..., which after TrimRight("0") becomes
	// "0.1" with 1 decimal → triggers the <2 decimals padding branch.
	// Sonnet: saveRate=2.70, overheadRate=0.75
	// CumCR=37100*2.70=100170, CumCC=100*0.75=75 → net=100095 → $0.100 → "0.1" → pad
	makeInput := func(totalIn float64) []byte {
		in := map[string]any{
			"workspace": map[string]any{"current_dir": testCwd},
			"model":     map[string]any{"display_name": "Sonnet 4.6"},
			"context_window": map[string]any{
				"remaining_percentage": 80.0,
				"total_input_tokens":   totalIn,
				"total_output_tokens":  2000.0,
				"context_window_size":  200000.0,
				"current_usage": map[string]any{
					"cache_read_input_tokens":     18550.0,
					"cache_creation_input_tokens": 50.0,
					"input_tokens":               200.0,
				},
			},
		}
		d, _ := json.Marshal(in)
		return d
	}
	run(bytes.NewReader(makeInput(10000)), io.Discard)

	var buf bytes.Buffer
	run(bytes.NewReader(makeInput(15000)), &buf)
	if !strings.Contains(buf.String(), "saved") {
		t.Error("should show savings with decimal padding")
	}
}

func TestRunEdgeConfigValues(t *testing.T) {
	resetSession()
	// barWidth <= 0 and precision < 0
	os.WriteFile(configPath, []byte(`{"context_bar":{"width":0},"cost":{"precision":-1},"cache":{"sparkline_width":0}}`), 0644)
	defer os.Remove(configPath)

	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, testCwd, profileOpus)), &buf)
	if buf.Len() == 0 {
		t.Error("should produce output with edge config values")
	}
}

// --- config.go validate() ---

func TestValidateInvalidVelocityUnit(t *testing.T) {
	c := defaultConfig()
	c.Cost.VelocityUnit = "bogus"
	var buf bytes.Buffer
	c.validate(&buf)
	if c.Cost.VelocityUnit != "hour" {
		t.Errorf("expected reset to hour, got %q", c.Cost.VelocityUnit)
	}
	if !strings.Contains(buf.String(), "invalid velocity_unit") {
		t.Error("should warn about invalid velocity_unit")
	}
}

func TestValidateInvalidDurationFormat(t *testing.T) {
	c := defaultConfig()
	c.Duration.Format = "bogus"
	var buf bytes.Buffer
	c.validate(&buf)
	if c.Duration.Format != "auto" {
		t.Errorf("expected reset to auto, got %q", c.Duration.Format)
	}
	if !strings.Contains(buf.String(), "invalid duration.format") {
		t.Error("should warn about invalid duration format")
	}
}

func TestValidateInvalidTokensFormat(t *testing.T) {
	c := defaultConfig()
	c.Tokens.Format = "bogus"
	var buf bytes.Buffer
	c.validate(&buf)
	if c.Tokens.Format != "auto" {
		t.Errorf("expected reset to auto, got %q", c.Tokens.Format)
	}
	if !strings.Contains(buf.String(), "invalid tokens.format") {
		t.Error("should warn about invalid tokens format")
	}
}

func TestValidateNegativeGitCacheTTL(t *testing.T) {
	c := defaultConfig()
	c.Git.CacheTTL = -1
	var buf bytes.Buffer
	c.validate(&buf)
	if c.Git.CacheTTL != 5 {
		t.Errorf("expected reset to 5, got %d", c.Git.CacheTTL)
	}
	if !strings.Contains(buf.String(), "invalid git.cache_ttl") {
		t.Error("should warn about invalid git.cache_ttl")
	}
}

func TestValidateInvalidSessionMaxHistory(t *testing.T) {
	c := defaultConfig()
	c.Session.MaxHistory = 0
	var buf bytes.Buffer
	c.validate(&buf)
	if c.Session.MaxHistory != 50 {
		t.Errorf("expected reset to 50, got %d", c.Session.MaxHistory)
	}
	if !strings.Contains(buf.String(), "invalid session.max_history") {
		t.Error("should warn about invalid session.max_history")
	}
}

func TestValidateValidValues(t *testing.T) {
	c := defaultConfig()
	var buf bytes.Buffer
	c.validate(&buf)
	if buf.Len() != 0 {
		t.Errorf("default config should produce no warnings, got %q", buf.String())
	}
}

func TestValidateAllValidEnumValues(t *testing.T) {
	// velocity_unit: "minute" branch
	c := defaultConfig()
	c.Cost.VelocityUnit = "minute"
	var buf bytes.Buffer
	c.validate(&buf)
	if c.Cost.VelocityUnit != "minute" {
		t.Error("minute should be accepted")
	}

	// duration.format: each valid value
	for _, f := range []string{"seconds", "minutes", "hours"} {
		c = defaultConfig()
		c.Duration.Format = f
		buf.Reset()
		c.validate(&buf)
		if c.Duration.Format != f {
			t.Errorf("%q should be accepted", f)
		}
	}

	// tokens.format: each valid value
	for _, f := range []string{"raw", "k", "M"} {
		c = defaultConfig()
		c.Tokens.Format = f
		buf.Reset()
		c.validate(&buf)
		if c.Tokens.Format != f {
			t.Errorf("%q should be accepted", f)
		}
	}

	if buf.Len() != 0 {
		t.Errorf("valid values should produce no warnings, got %q", buf.String())
	}
}

func TestSessionSparkHistTruncation(t *testing.T) {
	resetSession()
	s := loadSession()
	s.PrevTotalIn = 0
	for i := 0; i < 35; i++ {
		s.update(int64((i+1)*1000), 500, 100, 200)
	}
	if len(s.SparkHist) > 32 {
		t.Errorf("spark hist should truncate to 32, got %d", len(s.SparkHist))
	}
}

func TestRunContextRunwayYellow(t *testing.T) {
	resetSession()
	// Prime turns
	for i := 0; i < 5; i++ {
		run(bytes.NewReader(simulateTurn(i, testCwd, profileOpus)), io.Discard)
	}

	// ~6 turns remaining → yellow
	in := map[string]any{
		"workspace": map[string]any{"current_dir": testCwd},
		"model":     map[string]any{"display_name": "Opus 4.6"},
		"context_window": map[string]any{
			"remaining_percentage": 12.0,
			"total_input_tokens":   176000.0,
			"total_output_tokens":  5000.0,
			"context_window_size":  200000.0,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     100.0,
				"cache_creation_input_tokens": 50.0,
				"input_tokens":               200.0,
			},
		},
	}
	data, _ := json.Marshal(in)
	var buf bytes.Buffer
	run(bytes.NewReader(data), &buf)
	if !strings.Contains(buf.String(), Yellow) {
		t.Error("~6 remaining turns should show yellow")
	}
}

func TestRunAllDisabled(t *testing.T) {
	resetSession()
	allOff := `{"user_host":false,"cwd":false,"model":false,"git":{"enabled":false},"context_bar":{"enabled":false},"cost":{"enabled":false},"duration":{"enabled":false},"tokens":{"enabled":false},"cache":{"enabled":false},"context_runway":false,"rate_limits":{"enabled":false},"line_deltas":false,"api_stats":{"enabled":false},"session_compare":false}`
	os.WriteFile(configPath, []byte(allOff), 0644)
	defer os.Remove(configPath)

	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, testCwd, profileOpus)), &buf)
	if buf.Len() != 0 {
		t.Errorf("all disabled should produce empty output, got %q", buf.String())
	}
}

func TestRunWithGitInfo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	resetSession()

	// Create a repo with a remote to exercise ahead/behind in run()
	remoteDir := t.TempDir()
	exec.Command("git", "init", "--bare", remoteDir).Run()
	dir := filepath.Join(t.TempDir(), "local")
	exec.Command("git", "clone", remoteDir, dir).Run()
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "T").Run()
	os.WriteFile(filepath.Join(dir, "f.go"), []byte("package main\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()
	exec.Command("git", "-C", dir, "push", "origin", "HEAD").Run()

	// Make local ahead by 1
	os.WriteFile(filepath.Join(dir, "g.go"), []byte("package main\n// new\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "ahead").Run()

	// Also add an uncommitted file for changes display
	os.WriteFile(filepath.Join(dir, "h.go"), []byte("package main\n// uncommitted\n"), 0644)

	os.Remove(filepath.Join(dir, ".git", ".statusline-cache-go"))
	var buf bytes.Buffer
	run(bytes.NewReader(simulateTurn(5, dir, profileOpus)), &buf)
	output := buf.String()

	if !strings.Contains(output, Blue) {
		t.Error("should contain branch name in blue")
	}
	if !strings.Contains(output, "↑") {
		t.Error("should show ahead indicator")
	}
}

func TestRunNoRemainingPercentage(t *testing.T) {
	resetSession()
	in := map[string]any{
		"workspace": map[string]any{"current_dir": testCwd},
		"model":     map[string]any{"display_name": "Test"},
		"context_window": map[string]any{
			"total_input_tokens":  5000.0,
			"total_output_tokens": 1000.0,
			"current_usage": map[string]any{
				"cache_read_input_tokens":     100.0,
				"cache_creation_input_tokens": 50.0,
				"input_tokens":               200.0,
			},
		},
	}
	data, _ := json.Marshal(in)
	var buf bytes.Buffer
	run(bytes.NewReader(data), &buf)
	if strings.Contains(buf.String(), "█") {
		t.Error("should not show context bar without remaining_percentage")
	}
}
