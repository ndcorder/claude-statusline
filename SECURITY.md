# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email security concerns to nicolas@go-redrock.com
3. Include steps to reproduce if possible

I'll respond within 48 hours and work with you on a fix.

## Verifying Release Binaries

All release binaries include [GitHub Artifact Attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations) (SLSA build provenance).

### Verify a downloaded binary

```bash
gh attestation verify claude-statusline_*_darwin_arm64.tar.gz --owner ndcorder
```

This cryptographically proves the binary was:
- Built from this repository's source code
- Built by the GitHub Actions release workflow
- Not tampered with after building

No keys to manage — verification uses GitHub's OIDC-backed Sigstore transparency log.

### Build from source

For maximum assurance, build from source:

```bash
git clone https://github.com/ndcorder/claude-statusline.git
cd claude-statusline
make build
```

## Supply Chain

- **Zero external dependencies** — stdlib only, no transitive dependency risk
- **Reproducible builds** — built with `-trimpath` for path-independent output
- **SHA-256 checksums** — included in every release
- **100% test coverage** — CI enforces full coverage on every commit
- **Signed commits** — all commits are GPG-signed
