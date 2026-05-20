# claude-statusline

[![GitHub release](https://img.shields.io/github/v/release/ndcorder/claude-statusline)](https://github.com/ndcorder/claude-statusline/releases)
[![CI](https://github.com/ndcorder/claude-statusline/actions/workflows/ci.yml/badge.svg)](https://github.com/ndcorder/claude-statusline/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A feature-rich status line for [Claude Code](https://claude.ai/code) with token stats, cache metrics, git info, and cost tracking.

**[Website](https://ndcorder.github.io/claude-statusline/)** · **[Changelog](CHANGELOG.md)** · **[Contributing](CONTRIBUTING.md)**

## Screenshot

```
cyan@host ~/project | Opus 4.6 (1M context) | main ↑2 +15/-3 (2f) | ████░░░░░░ 62% | $1.24@$3.40/hr | 12m34s
  tok ↓45.2k ↑8.1k | cache 74%hit r:28.5k/w:3.2k | saved +$0.38 | ▂▃▅▆▇█▇▆ | ~12turns | rate 5h18% 4h12m 7d3% | Δ+142/-37 | api 487s(65%) 16t/s | vs8avg $+0.12
```

**Line 1:** user@host, working directory, model, git branch/status, context usage bar, session cost & velocity, elapsed time

**Line 2:** token I/O, cumulative cache hit rate & savings, cache hit sparkline, estimated turns remaining, rate limit usage, line deltas, API throughput, cost vs historical average

## Install

### Homebrew

```bash
brew install ndcorder/tap/claude-statusline
```

### Pre-built binaries

Download from [Releases](https://github.com/ndcorder/claude-statusline/releases) and copy to `~/.claude/`:

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/ndcorder/claude-statusline/releases/latest/download/claude-statusline_$(curl -s https://api.github.com/repos/ndcorder/claude-statusline/releases/latest | grep tag_name | cut -d'"' -f4 | tr -d v)_darwin_arm64.tar.gz | tar xz -C ~/.claude/ claude-statusline
```

### With `go install`

```bash
go install github.com/ndcorder/claude-statusline@latest
cp $(go env GOPATH)/bin/claude-statusline ~/.claude/
```

### From source

```bash
git clone https://github.com/ndcorder/claude-statusline.git
cd claude-statusline
make install
```

## Configure Claude Code

Add to your Claude Code `settings.json`:

```json
{
  "statusLine": {
    "enabled": true,
    "command": "~/.claude/claude-statusline"
  }
}
```

## Configuration

Create `~/.claude/statusline.json` to customize. All fields are optional — unspecified fields keep their defaults (everything enabled).

### Example: minimal status line

```json
{
  "user_host": false,
  "git": { "enabled": false },
  "cache": { "sparkline": false, "savings": false },
  "api_stats": { "enabled": false },
  "session_compare": false
}
```

### Full reference

| Field | Type | Default | Description |
|---|---|---|---|
| `user_host` | bool | `true` | Show `user@hostname` |
| `cwd` | bool | `true` | Show working directory |
| `model` | bool | `true` | Show model name |
| `git.enabled` | bool | `true` | Show git info (disabling skips git calls entirely) |
| `git.ahead_behind` | bool | `true` | Show ↑ahead ↓behind counts |
| `git.changes` | bool | `true` | Show +added/-removed (files) |
| `git.cache_ttl` | int | `5` | Git cache TTL in seconds (0 = always refresh) |
| `context_bar.enabled` | bool | `true` | Show context usage bar |
| `context_bar.width` | int | `10` | Bar width in characters |
| `cost.enabled` | bool | `true` | Show session cost |
| `cost.precision` | int | `2` | Decimal places for cost display |
| `cost.velocity` | bool | `true` | Show spend rate |
| `cost.velocity_unit` | string | `"hour"` | `"hour"` or `"minute"` |
| `duration.enabled` | bool | `true` | Show elapsed time |
| `duration.format` | string | `"auto"` | `"auto"`, `"seconds"`, `"minutes"`, `"hours"` |
| `tokens.enabled` | bool | `true` | Show token I/O counts |
| `tokens.format` | string | `"auto"` | `"auto"`, `"raw"`, `"k"`, `"M"` |
| `cache.enabled` | bool | `true` | Show cache statistics |
| `cache.cumulative` | bool | `true` | Session-wide stats vs per-request |
| `cache.savings` | bool | `true` | Show estimated dollar savings |
| `cache.sparkline` | bool | `true` | Show cache hit sparkline |
| `cache.sparkline_width` | int | `8` | Number of entries in sparkline |
| `context_runway` | bool | `true` | Show estimated turns remaining |
| `rate_limits.enabled` | bool | `true` | Show rate limit usage |
| `rate_limits.show_reset` | bool | `true` | Show reset countdown |
| `rate_limits.show_7day` | bool | `true` | Show 7-day usage |
| `line_deltas` | bool | `true` | Show lines added/removed |
| `api_stats.enabled` | bool | `true` | Show API time and throughput |
| `api_stats.throughput` | bool | `true` | Show tokens/sec |
| `session_compare` | bool | `true` | Show cost vs historical average |
| `pricing.opus.cache_read_rate` | float | `13.50` | Opus cache read rate per 1M tokens |
| `pricing.opus.cache_create_rate` | float | `3.75` | Opus cache create rate per 1M tokens |
| `pricing.sonnet.cache_read_rate` | float | `2.70` | Sonnet cache read rate per 1M tokens |
| `pricing.sonnet.cache_create_rate` | float | `0.75` | Sonnet cache create rate per 1M tokens |
| `pricing.haiku.cache_read_rate` | float | `0.72` | Haiku cache read rate per 1M tokens |
| `pricing.haiku.cache_create_rate` | float | `0.20` | Haiku cache create rate per 1M tokens |
| `session.path` | string | `os.TempDir()/...` | Session state file path |
| `session.history_path` | string | `os.TempDir()/...` | History file path |
| `session.max_history` | int | `50` | Maximum history entries |

Invalid config values produce warnings on stderr and fall back to defaults.

### Generate default config

```bash
claude-statusline --init-config > ~/.claude/statusline.json
```

Then edit the file to disable or customize specific segments.

## How It Works

Claude Code pipes a JSON payload to stdin on each prompt. The binary parses it, computes derived metrics, and prints two ANSI-colored lines to stdout.

Session state (cumulative cache stats, turn count, sparkline history) is tracked in a temp file (configurable via `session.path`). A new session is detected when `total_input_tokens` decreases — indicating context compaction or a new conversation.

Git info is collected by shelling out to `git` and cached inside `.git/.statusline-cache-go` with a configurable TTL (default 5 seconds) to avoid repeated subprocess calls.

## Performance

Benchmarked with [hyperfine](https://github.com/sharkdp/hyperfine) against the equivalent bash+jq implementation (Apple M4 Max, 100 runs, 5 warmup):

| Scenario | Bash+jq | Go | Speedup |
|---|---|---|---|
| No git | 146.9ms ± 4.9 | 11.9ms ± 0.7 | **12.3x ± 0.8** |
| Git (warm cache) | 112.8ms ± 6.5 | 12.1ms ± 0.8 | **9.3x ± 0.8** |
| Git (cold cache) | 145.4ms ± 4.2 | 43.1ms ± 2.3 | **3.4x ± 0.2** |

The bash version spawns `jq` 13+ times, `awk`, `sed`, and multiple `git` subprocesses per render. The Go binary does a single JSON unmarshal, direct git calls with a 5-second cache, and zero external dependencies.

Run `make bench` for in-process benchmarks with simulated multi-turn Claude Code sessions.

## Development

```bash
make build          # build ./claude-statusline
make test           # run tests
make bench          # run benchmarks
make install        # build + copy to ~/.claude/
make clean          # remove binary
```

```bash
claude-statusline --version      # show version
claude-statusline --init-config  # dump default config as JSON
```

Zero external dependencies — stdlib only. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Security & Trust

The code you review is the code you run. Three ways to verify:

```bash
# Option 1: Build from source (strongest — Go verifies source via sum.golang.org)
go install github.com/ndcorder/claude-statusline@latest

# Option 2: Verify a pre-built binary (SLSA provenance via Sigstore)
gh attestation verify claude-statusline_*_darwin_arm64.tar.gz --owner ndcorder

# Option 3: Reproduce the release binary locally and compare checksums
make verify
```

Every release is **verified reproducible in CI** — binaries are independently rebuilt from source and checksum-compared before a release is considered complete.

Zero external dependencies. Signed commits. Branch protection. 100% test coverage enforced.

See [SECURITY.md](SECURITY.md) for full details and vulnerability disclosure policy.

## License

MIT
