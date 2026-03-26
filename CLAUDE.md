# Upgrade Guard

Kubernetes infrastructure dependency upgrade risk assessment tool. Uses Claude API
with web search to analyze PRs from Renovate/Dependabot and post risk assessments.

## Project Structure

```
cmd/upgrade-guard/main.go        — CLI entrypoint (cobra)
internal/config/                  — Config loading (upgrade-guard.yaml)
internal/parser/                  — PR diff parsing, Renovate metadata, component detection
internal/state/                   — Platform state inference (repo files + config)
internal/analyst/                 — LLM analysis (Claude API + web search, prompt building)
internal/output/                  — Output formatters (GitHub PR comment, JSON, stdout)
adapters/github-action/           — GitHub Action adapter
docs/                             — Design doc + implementation status
```

## Build & Test

```bash
make build                        # Build binary
make test                         # Run tests
make lint                         # Run golangci-lint
make docker-build                 # Build Docker image locally
```

## Key Design Decisions

- **Go** — single static binary, fast cold starts, distroless container
- **Anthropic SDK only** — no multi-provider abstraction; SDK has native web search
- **Guard levels** — basic (diff only), standard (repo-aware), full (cluster-connected)
- **Model auto-selection** — Sonnet for basic/standard, Opus for full level
- **Server-side web search** — `WebSearchTool20250305Param`, API handles search execution

## Conventions

- Follow template-go-app patterns (gitversion, goreleaser, golangci-lint, Alpine Dockerfile)
- Conventional commits for versioning
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Config: defaults → YAML file → env vars (highest priority)

## Testing

Run `make test` or `go test ./...`. Tests are co-located with source in `*_test.go` files.
The `internal/analyst/anthropic.go` integration with Claude API is not unit-tested (requires API key);
test prompt building and output formatting instead.

## Docs

- [Design document](docs/upgrade-guard-design.md) — architecture, guard levels, prompt design
- [Implementation status](docs/implementation-status.md) — what's done, what's planned, deviations
