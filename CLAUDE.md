# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go CLI tool that renders Claude Code's status line. It reads JSON from stdin (piped by Claude Code's `statusLine.command` setting), computes metrics, and prints two ANSI-colored lines to stdout. Supports `--init-config` to dump default config as JSON.

## Build, Test & Install

```bash
make build          # builds ./claude-statusline
make test           # runs tests with -v
make bench          # runs benchmarks (3 iterations, -benchmem)
make install        # builds + copies to ~/.claude/claude-statusline
make clean          # removes binary
```

No external dependencies — stdlib only (`go.mod` has zero requires).

## Testing

100% statement coverage, enforced by CI. Tests live in two files:

- **bench_test.go** — Multi-turn Claude Code session simulator, benchmarks (single render, full session, git cold/warm cache, end-to-end binary, per-profile), and simulation validation tests.
- **coverage_test.go** — Unit tests targeting every branch: format helpers, config loading (including mocked `userHomeDir`), git lifecycle (remotes, detached HEAD, worktrees, cache hit/miss/invalid), session edge cases, and all `run()` paths.

`TestMain` in bench_test.go sets up isolated temp paths for `sessionPath`, `historyPath`, and `configPath` so tests never touch real state.

## Architecture

Single-package `main` with six source files:

- **main.go** — Entry point and `run(r, w)` which owns all rendering. Line 1: user@host, cwd, model, git info, context bar, cost, duration. Line 2: tokens, cache stats, savings, sparkline, runway, rate limits, line deltas, API throughput, session comparison. Every segment is gated by `cfg`.
- **input.go** — Struct definitions for the JSON schema Claude Code pipes in.
- **config.go** — `Config` struct with nested sub-configs, `defaultConfig()`, `loadConfig()` reading from `~/.claude/statusline.json`. Partial JSON overrides work — unspecified fields keep defaults.
- **session.go** — Tracks cumulative cache stats and turn count across invocations via `/tmp/claude-statusline-session-go`. Detects new sessions when `totalIn` decreases. Maintains history at `/tmp/claude-statusline-history` (capped at 50 entries).
- **git.go** — Shells out to `git` for branch, ahead/behind, changed files, diff stats. Caches results in `.git/.statusline-cache-go` with a 5-second TTL.
- **format.go** — ANSI color constants, `fmtTokens`/`fmtTokensUnit` (auto/raw/k/M), `fmtDuration` (auto/seconds/minutes/hours), `colorPct`.

## Key Design Decisions

- **No `go.sum`** — zero dependencies means no lockfile needed.
- **Session detection** — `totalIn` decrease triggers new session. Previous session stats are appended to history for cross-session cost comparison.
- **Cache pricing** — Hardcoded per-model rates (Opus/Sonnet/Haiku) estimate dollar savings from cache hits vs. creation overhead.
- **Git cache** — Written inside `.git/` as `.statusline-cache-go` to avoid polluting the working tree. Disabling `git.enabled` skips all git subprocess calls (~34ms savings).
- **Sparkline storage** — Session stores up to 32 entries; display truncates to `sparkline_width` (default 8).
- **Config loading** — `json.Unmarshal` into pre-filled `defaultConfig()` means partial configs work. File absence is silent (returns defaults).
- **Reproducible builds** — `-trimpath` in Makefile and goreleaser.

## CI & Releases

- **CI** (`.github/workflows/ci.yml`) — Runs on push/PR to master. Tests with `-race`, enforces 100% coverage, cross-compile build matrix.
- **Release** (`.github/workflows/release.yml`) — Triggered by `v*` tags. Goreleaser builds binaries for darwin/linux/windows × amd64/arm64. GitHub Artifact Attestations (SLSA provenance) are generated for all release assets.
- **Branch protection** — CI required, signed commits required, force push blocked. Admin bypass enabled for direct pushes.
