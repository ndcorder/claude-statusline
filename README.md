# claude-statusline

A feature-rich status line for [Claude Code](https://claude.ai/code) with token stats, cache metrics, git info, and cost tracking.

## Screenshot

```
cyan@host ~/project | Opus 4.6 (1M context) | main ↑2 +15/-3 (2f) | ████░░░░░░ 62% | $1.24@$3.40/hr | 12m34s
  tok ↓45.2k ↑8.1k | cache 74%hit r:28.5k/w:3.2k | saved +$0.38 | ▂▃▅▆▇█▇▆ | ~12turns | rate 5h18% 4h12m 7d3% | Δ+142/-37 | api 487s(65%) 16t/s | vs8avg $+0.12
```

**Line 1:** user@host, working directory, model, git branch/status, context usage bar, session cost & velocity, elapsed time

**Line 2:** token I/O, cumulative cache hit rate & savings, cache hit sparkline, estimated turns remaining, rate limit usage, line deltas, API throughput, cost vs historical average

## Install

### From source

```bash
git clone https://github.com/kexxt/claude-statusline.git
cd claude-statusline
make install
```

This builds the binary and copies it to `~/.claude/claude-statusline`.

### With `go install`

```bash
go install github.com/kexxt/claude-statusline@latest
```

Then copy the binary to `~/.claude/`:

```bash
cp $(go env GOPATH)/bin/claude-statusline ~/.claude/
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

## Features

All segments are configurable via the `Config` struct in `config.go`:

| Feature | Default | Description |
|---|---|---|
| CostVelocity | on | $/hr rate when session > 1 min |
| TokenThroughput | on | Output tokens/sec during API calls |
| CumulativeCache | on | Session-wide cache stats (vs per-request) |
| ContextRunway | on | Estimated turns until context exhaustion |
| CacheSavings | on | Dollar savings from cache hits |
| Sparkline | on | Last 8 cache hit rates as Unicode blocks |
| SessionCompare | on | Cost difference vs historical average |

## How It Works

Claude Code pipes a JSON payload to stdin on each prompt. The binary parses it, computes derived metrics, and prints two ANSI-colored lines to stdout.

Session state (cumulative cache stats, turn count, sparkline history) is tracked in `/tmp/claude-statusline-session-go`. A new session is detected when `total_input_tokens` decreases — indicating context compaction or a new conversation.

Git info is collected by shelling out to `git` and cached for 5 seconds inside `.git/.statusline-cache-go` to avoid repeated subprocess calls.

## Development

```bash
make build          # build ./claude-statusline
make test           # run tests
make bench          # run benchmarks
make install        # build + copy to ~/.claude/
make clean          # remove binary
```

Zero external dependencies — stdlib only.

## License

MIT
