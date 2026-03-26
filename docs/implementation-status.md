# Upgrade Guard — Implementation Status

**Design doc:** [upgrade-guard-design.md](upgrade-guard-design.md)

---

## Architecture Overview

The implementation follows the design doc's architecture. The core container
runs as a CLI (`upgrade-guard analyze`) and can be invoked from any CI system
via thin adapter scripts.

```
CLI (cobra) → PR Parser → State Loader → LLM Analyst (Claude + web search) → Output Formatter
```

---

## Component Status

### Core (internal/)

| Component | Package | Status | Notes |
|-----------|---------|--------|-------|
| Config loading | `internal/config` | ✅ Done | YAML config + defaults + env var override |
| Diff parser | `internal/parser` | ✅ Done | Unified diff parsing |
| Renovate metadata parser | `internal/parser` | ✅ Done | Table + prose formats |
| Component detection | `internal/parser` | ✅ Done | Helm charts, container images, Kustomize refs |
| Repo state inference | `internal/state` | ✅ Done | Scans Chart.yaml, values.yaml, kustomization.yaml, .tf files |
| Config file state | `internal/state` | ✅ Done | Merges upgrade-guard.yaml on top of inferred state |
| Cluster state loader | `internal/state` | ✅ Done | kubectl + helm queries for Full guard level |
| Prompt builder | `internal/analyst` | ✅ Done | Level-aware prompts (basic/standard/full) |
| Claude API + web search | `internal/analyst` | ✅ Done | Uses anthropic-sdk-go with WebSearchTool20250305 |
| GitHub PR comment | `internal/output` | ✅ Done | Creates or updates existing comment (marker-based) |
| Azure DevOps comment | `internal/output` | ✅ Done | REST API, creates or updates existing thread |
| JSON output | `internal/output` | ✅ Done | |
| Stdout output | `internal/output` | ✅ Done | |

### CLI (cmd/)

| Feature | Status | Notes |
|---------|--------|-------|
| `analyze` command | ✅ Done | --repo, --pr, --diff, --pr-body, --config, --output, --repo-root |
| Stdin pipe support | ✅ Done | `git diff ... \| upgrade-guard analyze` |
| GitHub API integration | ✅ Done | Fetches PR diff + body, posts comments |
| Azure DevOps flags | ✅ Done | --ado-org, --ado-project, --ado-repo, --ado-pr |
| Version/commit injection | ✅ Done | `-ldflags -X main.version=... -X main.commit=...` |

### CI Adapters

| Adapter | Status | Notes |
|---------|--------|-------|
| GitHub Action | ✅ Done | `adapters/github-action/action.yml` |
| Azure DevOps | ✅ Done | `adapters/azure-devops/pipeline-template.yml` |
| CI workflow (release) | ✅ Done | `.github/workflows/build-and-release.yaml` |

### Tooling

| Tool | Status | Notes |
|------|--------|-------|
| Dockerfile | ✅ Done | Alpine base, GoReleaser-built binary, non-root user |
| .goreleaser.yaml | ✅ Done | Multi-arch binary, GPG signing, grouped changelog |
| gitversion.yml | ✅ Done | Conventional commits → SemVer |
| .golangci.yml | ✅ Done | govet, errcheck, staticcheck, unused, gocritic + formatters |
| Makefile | ✅ Done | build, lint, test, docker-build targets |
| .gitignore | ✅ Done | |

### Tests

| Package | Status | Notes |
|---------|--------|-------|
| `internal/parser` | ✅ Done | Diff parsing, Renovate metadata, component detection |
| `internal/config` | ✅ Done | Config loading, defaults, model selection |
| `internal/state` | ✅ Done | Config file merge |
| `internal/analyst` | ✅ Done | Prompt building |
| `internal/output` | ✅ Done | Stdout, JSON formatting |
| Integration / E2E | ❌ Not started | Needs real PR diffs as fixtures |

---

## Design Deviations

| Topic | Design says | Implementation | Reason |
|-------|-------------|----------------|--------|
| Dockerfile | Distroless image | Alpine with GoReleaser binary copy | Matches template-go-app pattern; Alpine needed for git + ca-certificates in CI |
| Web search tool | Generic "web search" | `WebSearchTool20250305Param` | SDK-specific type name; newer `20260209` variant available if needed |
| Azure DevOps SDK | `github.com/microsoft/azure-devops-go-api` | Direct REST API calls | ADO Go SDK is poorly maintained; REST API v7.1 is simpler and reliable |
| Cluster state | `k8s.io/client-go` | `kubectl` + `helm` CLI via `exec` | Avoids heavy k8s client-go dependency; kubectl/helm already present in CI |

---

## Roadmap

### v0.1 (MVP) — Done
- [x] Parse Renovate PRs on GitHub
- [x] Detect Helm chart and container image version bumps
- [x] Infer state from repo files + upgrade-guard.yaml
- [x] Claude API with web search for risk analysis
- [x] Post markdown comment on GitHub PRs
- [x] Docker container image
- [x] Unit tests
- [x] CI/CD pipeline

### v0.2 — Done
- [x] Azure DevOps adapter (output + pipeline template)
- [x] Cluster query state loader (Full guard level)

### v0.3 — Planned
- [ ] Kustomize ref detection improvements
- [ ] Terraform module analysis
- [ ] Rate limiting / debounce for batch Renovate PRs
- [ ] Caching / deduplication of analysis

### v0.4+ — Future
- [ ] Custom plugin/hook system
- [ ] GitLab CI adapter
- [ ] UI dashboard
