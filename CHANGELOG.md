# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `--version` flag with build-time version injection via ldflags
- Configurable cache pricing rates per model (`pricing` config section)
- Configurable session and history file paths (`session` config section)
- Configurable git cache TTL (`git.cache_ttl`)
- Configurable max history entries (`session.max_history`)
- Config validation with stderr warnings for invalid values
- Homebrew tap for easy installation (`brew install ndcorder/tap/claude-statusline`)
- Project website at ndcorder.github.io/claude-statusline
- CONTRIBUTING.md with development guide

### Changed

- Session and history paths default to `os.TempDir()` instead of hardcoded `/tmp` (cross-platform)
- Sparkline storage cap is now 4× the display width instead of hardcoded 32
- Malformed config JSON now produces a stderr warning instead of silent failure
- Invalid JSON on stdin now produces a stderr warning instead of silent failure
- Refactored `main()` to delegate to `cliMain()` for cross-platform testability

## [0.5.2] - 2026-05-20

### Fixed

- Add `-buildvcs=false` flag for reproducible builds

## [0.5.1] - 2026-05-20

### Fixed

- Match goreleaser microarchitecture flags in CI verify job

## [0.5.0] - 2026-05-20

### Added

- Reproducible builds with `CGO_ENABLED=0` across goreleaser, Makefile, and verify target
- CI verify-reproducibility job: rebuilds each binary from source and confirms SHA-256 checksums match
- `make verify` target for local reproducibility verification
- Tiered trust model in SECURITY.md: source install, attestation verification, and local reproducibility

### Changed

- Updated README security section with streamlined verification instructions
- Expanded SECURITY.md with reproducible build documentation

## [0.4.0] - 2026-05-20

### Added

- GitHub Artifact Attestations (Sigstore-backed) on every release binary
- SHA-256 checksums in every release
- `-trimpath` flag for reproducible builds
- SECURITY.md with verification instructions and vulnerability disclosure policy
- Branch protection: require CI, require signed commits, block force push

## [0.3.0] - 2026-05-20

### Added

- GitHub Actions CI: test with race detector, enforce 100% coverage, cross-compile build matrix
- goreleaser: pre-built binaries for darwin/linux/windows × amd64/arm64 on tag push
- `--init-config` flag to dump default config as JSON
- Pre-built binary installation instructions in README

## [0.2.0] - 2026-05-19

### Added

- JSON config file (`~/.claude/statusline.json`) for full statusline customization
- Every segment toggleable with sensible defaults (all enabled)
- Unit options: cost precision, velocity unit, duration format, token format, bar/sparkline width
- Disabling git skips subprocess calls entirely (zero overhead)
- 100% test coverage with comprehensive unit tests for all functions and branches
- Mockable `userHomeDir` for config path testing
- Git lifecycle tests: remotes, ahead/behind, detached HEAD, worktrees, cache invalidation
- Performance comparison vs bash implementation in README
- Hyperfine benchmark results in documentation

### Changed

- Sparkline storage increased to 32 entries for configurable width

### Removed

- Dead code: impossible `json.Marshal` error checks, unreachable `SplitN` guard, redundant sparkline bounds

## [0.1.0] - 2026-05-19

### Added

- Initial release of Go CLI status line for Claude Code
- Token stats with input/output/cache breakdown
- Cumulative cache metrics and cost tracking
- Git info: branch, ahead/behind, changed files, diff stats
- Context runway estimation
- Session-over-session cost comparison
- ANSI-colored two-line output
- Multi-turn Claude Code session simulation benchmarks

### Fixed

- Correct GitHub username in module path and docs

[Unreleased]: https://github.com/ndcorder/claude-statusline/compare/v0.5.2...HEAD
[0.5.2]: https://github.com/ndcorder/claude-statusline/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/ndcorder/claude-statusline/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/ndcorder/claude-statusline/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/ndcorder/claude-statusline/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/ndcorder/claude-statusline/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/ndcorder/claude-statusline/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ndcorder/claude-statusline/releases/tag/v0.1.0
