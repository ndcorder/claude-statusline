# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go CLI tool that renders Claude Code's status line. It reads JSON from stdin (piped by Claude Code's `statusLine.command` setting), computes metrics, and prints two ANSI-colored lines to stdout.

## Build & Install

```bash
make build          # builds ./claude-statusline
make install        # builds + copies to ~/.claude/claude-statusline
make clean          # removes binary
```

No external dependencies — stdlib only (`go.mod` has zero requires).

## Architecture

Single-package `main` with six files, each owning one concern:

- **main.go** — Entry point. Reads JSON from stdin, assembles two output lines. Line 1: user@host, cwd, model, git info, context bar, cost, duration. Line 2: tokens, cache stats, savings, sparkline, runway, rate limits, line deltas, API throughput, session comparison.
- **input.go** — Struct definitions for the JSON schema Claude Code pipes in (context window, cost, rate limits).
- **session.go** — Tracks cumulative cache stats and turn count across invocations via `/tmp/claude-statusline-session-go`. Detects new sessions by comparing `totalIn` with previous value. Maintains a history file at `/tmp/claude-statusline-history` for cross-session cost comparison.
- **git.go** — Shells out to `git` for branch, ahead/behind, changed files, and diff stats. Caches results in `.statusline-cache-go` inside the git dir with a 5-second TTL.
- **config.go** — Feature flags (`Config` struct) controlling which status line segments appear. All default to `true`.
- **format.go** — ANSI color constants, token formatting helpers, and color-by-percentage logic.

## Key Design Decisions

- **No `go.sum`** — zero dependencies means no lockfile needed.
- **Session detection** — A new session is detected when `totalIn` decreases (context was compacted or a new conversation started). Previous session stats are appended to history.
- **Cache pricing** — `CacheSavings` uses hardcoded per-model rates (Opus/Sonnet/Haiku) to estimate dollar savings from cache hits vs. creation overhead.
- **Git cache** — Written inside `.git/` as `.statusline-cache-go` to avoid polluting the working tree.
- **Sparkline** — Last 8 cache hit percentages visualized as Unicode block characters.
- **Context runway** — Estimates remaining turns from `(remaining tokens) / (avg tokens per turn)`.
