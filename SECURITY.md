# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email security concerns to nicolas@go-redrock.com
3. Include steps to reproduce if possible

I'll respond within 48 hours and work with you on a fix.

## Trusting the Code You Run

Three ways to verify, from easiest to most thorough:

### 1. Install from source (strongest)

```bash
go install github.com/ndcorder/claude-statusline@latest
```

Go compiles from source on your machine. The source is fetched through the [Go module proxy](https://proxy.golang.org) and verified against the [Go checksum database](https://sum.golang.org) — a public transparency log that guarantees the code you compile is exactly what's in this repository. No trust in pre-built binaries required.

### 2. Verify a pre-built binary (quick)

```bash
gh attestation verify claude-statusline_*_darwin_arm64.tar.gz --owner ndcorder
```

[GitHub Artifact Attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations) prove the binary was built from this repository's source, by the GitHub Actions release workflow, signed via [Sigstore](https://www.sigstore.dev/) on a public transparency log. No keys to manage.

### 3. Reproduce the build yourself (maximum assurance)

Check out the tagged release and build with the same flags CI uses:

```bash
git clone https://github.com/ndcorder/claude-statusline.git
cd claude-statusline
git checkout v1.0.0  # or any release tag
make build
```

Or compare your local build against the release automatically:

```bash
make verify
```

This downloads the release binary for your platform, rebuilds from source, and compares SHA-256 hashes. A match proves the release binary is byte-identical to what the source code produces.

**Every release is verified reproducible by CI** — a separate workflow job rebuilds each binary from source and confirms checksums match before the release is considered complete.

## Supply Chain

| Measure | Detail |
|---|---|
| Zero dependencies | stdlib only — no transitive supply chain risk |
| Reproducible builds | `CGO_ENABLED=0 -trimpath -ldflags='-s -w'` for deterministic output |
| CI-verified reproducibility | Every release is independently rebuilt and checksum-compared in CI |
| SLSA build provenance | All release artifacts attested via GitHub Artifact Attestations |
| SHA-256 checksums | Included and attested in every release |
| Signed commits | All commits are GPG-signed |
| Branch protection | CI required, signed commits required, force push blocked |
| 100% test coverage | Enforced by CI on every commit |
