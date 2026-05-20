# Contributing to claude-statusline

Thanks for wanting to contribute. This doc covers what you need to get going.

## Development Setup

```bash
git clone https://github.com/ndcorder/claude-statusline.git
cd claude-statusline
make build    # produces ./claude-statusline
make test     # runs tests with -race
make bench    # benchmarks (3 iterations, -benchmem)
make install  # builds + copies to ~/.claude/claude-statusline
```

Requires Go 1.22+. No external dependencies — stdlib only, no `go.sum`.

## Testing

CI enforces **100% statement coverage**. If you add code, you must add tests that cover every branch.

```bash
# Run tests
make test

# Check coverage locally
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

Test files:

| File | What it covers |
|---|---|
| `coverage_test.go` | Unit tests for every branch: format helpers, config loading, git lifecycle, session edge cases, all `run()` paths |
| `bench_test.go` | Multi-turn session simulator, benchmarks (single render, full session, git cache, end-to-end binary, per-profile), simulation validation |

`TestMain` in bench_test.go sets up isolated temp paths for `sessionPath`, `historyPath`, and `configPath` so tests never touch real state.

## Code Style

- **gofmt** — non-negotiable. CI will catch it if you forget.
- **No external dependencies.** This project is stdlib-only. If you need something from a third-party package, implement it yourself or make a strong case in the PR.
- **No `go.sum`** — zero dependencies means no lockfile.
- Match existing patterns. If the codebase uses `strings.Builder`, don't introduce `bytes.Buffer` for the same purpose.

## Commit Messages

Conventional commits: `type(scope): description`

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `perf`

Examples:

```
feat(cache): add per-model sparkline coloring
fix(git): handle bare repos without upstream
test(session): cover history file truncation edge case
docs: update config options in README
refactor(format): simplify fmtTokens thresholds
```

Keep the subject line under 72 characters. Body is optional but appreciated for non-obvious changes.

## Pull Requests

- **CI must pass.** Tests, 100% coverage, cross-platform build matrix.
- **Signed commits required.** Configure `git config commit.gpgsign true` if you haven't already.
- Target the `master` branch.
- One logical change per PR. Don't bundle unrelated fixes.

What reviewers look for:

- Does it maintain 100% coverage?
- Does it respect the stdlib-only policy?
- Is the config field gated properly (feature does nothing when disabled)?
- Are edge cases handled (nil pointers, zero values, empty strings)?
- Does it match existing code style and patterns?

## Architecture

The tool reads JSON from stdin (piped by Claude Code's `statusLine.command` setting), computes metrics, and prints two ANSI-colored lines to stdout.

```
stdin (JSON) → parse (input.go) → compute metrics (main.go, session.go, git.go)
                                 → format output (format.go)
                                 → ANSI stdout (main.go)
```

Source files:

| File | Responsibility |
|---|---|
| `main.go` | Entry point, `run(r, w)` renders both output lines. Every segment is gated by its config field. |
| `input.go` | Struct definitions for the JSON schema Claude Code pipes in. |
| `config.go` | `Config` struct with nested sub-configs, `defaultConfig()`, `loadConfig()` from `~/.claude/statusline.json`. Partial JSON overrides work — unspecified fields keep defaults. |
| `session.go` | Tracks cumulative cache stats and turn count across invocations via `/tmp/claude-statusline-session-go`. Detects new sessions when `totalIn` decreases. Maintains history at `/tmp/claude-statusline-history` (capped at 50 entries). |
| `git.go` | Shells out to `git` for branch, ahead/behind, changed files, diff stats. Caches results in `.git/.statusline-cache-go` with a 5-second TTL. |
| `format.go` | ANSI color constants, token formatting (`fmtTokens`/`fmtTokensUnit`), duration formatting (`fmtDuration`), `colorPct` for threshold-based coloring. |

## Adding a New Segment

A "segment" is a discrete piece of information rendered in the status line (e.g., cost, tokens, git branch). Here's how to add one:

### 1. Add a config field

In `config.go`, add a field to `Config` (or a new nested struct if it has sub-options):

```go
type Config struct {
    // ... existing fields ...
    MyFeature  MyFeatureConfig `json:"my_feature"`
}

type MyFeatureConfig struct {
    Enabled bool `json:"enabled"`
    // add sub-options here
}
```

Set the default in `defaultConfig()`:

```go
func defaultConfig() Config {
    return Config{
        // ... existing defaults ...
        MyFeature: MyFeatureConfig{
            Enabled: true,
        },
    }
}
```

### 2. Add rendering logic

In `main.go`, inside `run()`, add your segment in the appropriate line (L1 or L2). Gate it behind `cfg`:

```go
// Line 1 segments use addSep(&l1) and write to l1
// Line 2 segments append to the parts slice

if cfg.MyFeature.Enabled && someDataExists {
    parts = append(parts, Dim+"label"+Reset+" "+Green+value+Reset)
}
```

Line 1 is the top line (identity, location, git, context, cost, duration). Line 2 is the bottom line (tokens, cache, rate limits, stats). Put your segment where it logically belongs.

### 3. Add tests

In `coverage_test.go`, add tests that cover:

- Segment renders correctly when enabled and data is present
- Segment is absent when disabled via config
- Segment handles zero/nil/empty input gracefully
- Any edge cases specific to your feature

Run `go test -coverprofile=coverage.out ./...` and verify 100% before opening the PR.

### 4. Update docs

If your segment adds user-facing config options, update the README's configuration section.

## Adding a Config Option

If you're adding an option to an existing segment (not a new segment):

1. **Add the field** to the appropriate struct in `config.go` with a `json:"snake_case"` tag.
2. **Set a default** in `defaultConfig()`. Remember: `json.Unmarshal` into a pre-filled default means users only need to specify the fields they want to override.
3. **Use it** in `main.go` where the segment renders.
4. **Test it** — at minimum, test the default behavior and the overridden behavior.
5. **Validate if needed** — if the option has constraints (e.g., `width > 0`), add a guard in the rendering logic (see `barWidth` in the context bar segment for an example).

Config lives at `~/.claude/statusline.json`. Users can dump the full default config with:

```bash
./claude-statusline --init-config
```
