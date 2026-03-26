1# Upgrade Guard — Architecture Design Document

**Status:** Draft v0.1
**Date:** 2026-03-26
**Author:** Michiel / VH & Co

---

## 1. Problem Statement

Renovate (and similar dependency update bots) creates PRs that bump versions of infrastructure components — Helm charts, container images, Kustomize references. These PRs show changelogs and release notes, but they have zero awareness of:

- **Cross-component dependencies** — e.g. cluster-autoscaler has a hard version skew policy with the Kubernetes control plane version, but the K8s version isn't in the app manifests Renovate touches.
- **Upgrade ordering constraints** — some components must be upgraded before or after others.
- **Breaking changes in context** — a deprecation in cert-manager v1.14 might not matter unless you're using a specific feature that your config happens to rely on.
- **Platform-level side effects** — an ArgoCD upgrade might change CRD schemas, which affects every Application resource in the cluster.

Teams cannot reasonably know all of these interdependencies across the CNCF ecosystem. The result is merged PRs that silently break things.

---

## 2. Solution Overview

**Upgrade Guard** is a containerized tool that:

1. Gets triggered by a dependency update PR (from Renovate, Dependabot, or manual)
2. Extracts what changed (component, from-version, to-version)
3. Determines the current platform state (K8s version, other component versions)
4. Uses an LLM with web search to research changelogs, compatibility matrices, and known issues for the specific version transition
5. Posts a structured risk assessment as a PR comment

The core is a **portable container image** that can be invoked from any CI system. Thin adapter scripts handle CI-specific integration (GitHub Actions, Azure DevOps Pipelines, GitLab CI, etc.).

---

## 3. Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     CI / Pipeline Layer                  │
│  (thin adapters — trigger + pass context + post comment) │
├──────────────┬──────────────────────────────┬───────────┤
│ GitHub Action │  Azure DevOps Task/Pipeline  │ GitLab CI │
└──────┬───────┴──────────────┬───────────────┴───────┬───┘
       │                      │                       │
       ▼                      ▼                       ▼
┌─────────────────────────────────────────────────────────┐
│              Upgrade Guard Container (Core)              │
│                                                         │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │
│  │  PR Parser   │  │ State Loader │  │  LLM Analyst   │ │
│  │             │  │              │  │                │ │
│  │ - diff      │  │ - file scan  │  │ - web search   │ │
│  │ - renovate  │  │ - config     │  │ - changelog    │ │
│  │   metadata  │  │   file       │  │   analysis     │ │
│  │ - detect    │  │ - cluster    │  │ - compat check │ │
│  │   component │  │   query      │  │ - risk score   │ │
│  └──────┬──────┘  └──────┬───────┘  └───────┬────────┘ │
│         │                │                   │          │
│         ▼                ▼                   ▼          │
│  ┌─────────────────────────────────────────────────┐    │
│  │              Context Assembler                   │    │
│  │  Merges: diff + state + component metadata       │    │
│  │  Outputs: structured prompt for LLM              │    │
│  └──────────────────────┬──────────────────────────┘    │
│                         │                               │
│                         ▼                               │
│  ┌─────────────────────────────────────────────────┐    │
│  │              Output Formatter                    │    │
│  │  - Markdown PR comment                           │    │
│  │  - JSON (for programmatic consumption)           │    │
│  │  - Stdout (for local/debug use)                  │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

---

## 4. Core Components

### 4.1 PR Parser

Responsible for understanding what the PR changes.

**Inputs:**
- PR diff (unified diff format)
- Renovate PR body / metadata (if available — contains from/to versions, changelog links)

**Outputs:**
- List of changed components, each with:
  - `name` — e.g. `cluster-autoscaler`, `cert-manager`, `argocd`
  - `type` — `helm-chart` | `container-image` | `kustomize-ref` | `terraform-module`
  - `from_version` — previous version
  - `to_version` — new version
  - `source` — registry/repo URL
  - `files_changed` — which files in the repo were modified

**Detection strategies (in order):**
1. Parse Renovate JSON metadata from PR body (most reliable)
2. Parse diff for known patterns: `image:`, `tag:`, Helm `Chart.yaml` version, kustomize image transformers, Terraform module `source`/`version`
3. Fall back to filename + content heuristics

### 4.2 Guard Levels

The depth of analysis scales based on how much context is available. Each level builds on the previous one. Users choose their level via config, or the tool auto-detects based on what's available.

#### 🟢 Basic (zero config — works out of the box)

**Input:** PR diff only. No platform state.

**What it does:**
- Parses the PR to identify component + version change
- LLM researches changelogs, breaking changes, and known issues via web search
- Produces a **general compatibility matrix** — e.g. "cluster-autoscaler 1.30.x requires K8s ≥1.30, 1.29.x supports K8s 1.25–1.29" — covering all supported K8s versions
- Flags deprecations, migration steps, and CRD changes

**What it can't do:**
- Tell you if YOUR specific setup is affected (it doesn't know your K8s version)
- Cross-reference with other components in your stack

**Value:** Still catches 80% of issues. A team reading "this requires K8s ≥1.30" will immediately know if that's a problem.

#### 🟡 Standard (repo-aware)

**Input:** PR diff + platform state from repo files and/or `upgrade-guard.yaml`.

**How state is gathered (cascading, merged):**
1. **Infer from repo files** — scan for version pins:
   - `Chart.yaml` / `values.yaml` → Helm chart versions, image tags
   - Kustomize overlays → image tags, resource references
   - Terraform files → provider versions, module versions (including K8s version from `aws_eks_cluster`, `azurerm_kubernetes_cluster`, `google_container_cluster`)
   - ArgoCD `Application` manifests → target revisions
2. **Explicit config file** — `upgrade-guard.yaml` at repo root:

```yaml
# upgrade-guard.yaml
level: standard               # basic | standard | full

platform:
  kubernetes_version: "1.29"   # override or supplement inference
  provider: eks                # eks | aks | gke | k3s | kind | other

components:                    # pin versions the tool can't infer
  cluster-autoscaler: "1.29.0"
  cert-manager: "1.14.5"
  argocd: "2.10.6"
  external-secrets: "0.9.13"

context: |                     # free-text context for the LLM
  We use Cilium as CNI (not kube-proxy).
  ArgoCD manages all workloads via app-of-apps.
  External Secrets Operator syncs from AWS Secrets Manager.
```

**What it adds over Basic:**
- "cluster-autoscaler 1.30.x requires K8s ≥1.30. **Your cluster is on 1.29. This will break.**"
- Cross-component checks — e.g. "you're upgrading cert-manager but your external-secrets version has a known incompatibility with cert-manager >1.14"
- Contextual analysis — knows your CNI, GitOps tool, cloud provider

#### 🔴 Full (cluster-connected)

**Input:** Everything from Standard + live cluster state.

**Additional data from cluster:**
- `kubectl version` → actual K8s server version (ground truth)
- Helm releases → `helm list -A -o json` (actual deployed versions, not just what's in git)
- Running pod images → detect drift between git state and cluster state
- CRD versions → check if CRD updates are needed before/after the upgrade
- Node info → instance types, OS version, containerd version

**Requires:** kubeconfig access configured in `upgrade-guard.yaml`:

```yaml
level: full

cluster:
  kubeconfig_path: ""          # uses default if empty
  context: "my-cluster"        # specific context to use
```

**What it adds over Standard:**
- Detects git ↔ cluster drift — "your repo pins cert-manager 1.14.5 but the cluster is actually running 1.13.2"
- CRD pre-flight checks — "this upgrade changes the Certificate CRD, you have 47 Certificate resources that will be affected"
- Node-level compatibility — "this component requires containerd ≥1.7 but your nodes run 1.6"

### 4.3 LLM Analyst

The core intelligence. Takes the assembled context and produces a risk assessment.

**LLM provider:** Anthropic Claude SDK only. No multi-provider abstraction — keeps the codebase simple and the Anthropic SDK supports web search natively via tool use.

**Model selection — auto by default, overridable:**

The amount of context and the required reasoning depth scales with guard level, so model selection should too.

| Guard Level | Default Model | Rationale |
|-------------|--------------|-----------|
| 🟢 Basic | Sonnet 4.6 | Small context (diff + web search). Fast, cheap, more than capable. |
| 🟡 Standard | Sonnet 4.6 | Medium context (diff + repo state). Sonnet handles this well. |
| 🔴 Full | Opus 4.6 | Large context (full cluster state, many YAMLs, CRDs). Needs best reasoning over complex interdependencies. Trustworthiness matters most here. |

Users can override via config:

```yaml
settings:
  llm_model: auto             # auto | claude-sonnet-4-6 | claude-opus-4-6
```

`auto` (default) picks based on guard level as above. Explicit model name forces that model for all levels.

**Cost estimate per PR analysis:**
- Basic (Sonnet): ~$0.01–0.03
- Standard (Sonnet): ~$0.03–0.08
- Full (Opus): ~$0.10–0.30

**Key capability: web search.** The LLM must have access to web search to:
- Find official changelogs and release notes for the version range
- Look up version compatibility matrices (e.g. cluster-autoscaler's README has a version matrix)
- Find known issues, CVEs, or breaking changes
- Check if there are upgrade guides for major version jumps

**Prompt adapts to guard level:**
- Basic → "Here's what changed. Research the changelog and list risks across all supported K8s versions."
- Standard → "Here's what changed and here's our current state. Research the changelog and tell us specifically what will break in our setup."
- Full → "Here's what changed, our declared state, and our live cluster state. Give us the complete picture including any drift."

**Prompt structure:**

```
You are a Kubernetes platform engineering expert reviewing a dependency
update PR. Your job is to identify risks, breaking changes, and
compatibility issues that the team should be aware of before merging.

## What Changed
{component_changes}

## Current Platform State
{platform_state}

## Your Task
For each changed component:
1. Search for the official changelog/release notes between the from and
   to versions
2. Check if there are version compatibility requirements with Kubernetes
   or other components in the platform state
3. Identify breaking changes, deprecations, or behavioral changes
4. Flag any upgrade ordering requirements
5. Note any required manual steps (CRD updates, migration scripts, etc.)

Rate each change:
- 🟢 LOW RISK — routine patch, no compatibility concerns
- 🟡 MEDIUM RISK — minor version bump with notable changes, review recommended
- 🔴 HIGH RISK — breaking changes, version skew risk, or manual steps required

Output as structured markdown suitable for a PR comment.
```

### 4.4 Output Formatter

Transforms the LLM response into the appropriate format.

**PR Comment format (markdown):**

```markdown
## 🛡️ Upgrade Guard — Risk Assessment

### cluster-autoscaler `1.29.0` → `1.30.1`  🔴 HIGH RISK

**Compatibility:**
- ⚠️ cluster-autoscaler 1.30.x requires Kubernetes ≥1.30.
  Your cluster is running **Kubernetes 1.29**. This upgrade will
  break cluster autoscaling.

**Breaking Changes:**
- Priority expander config format changed in 1.30.0 — check if
  you're using priority-based expander.

**Action Required:**
- Upgrade Kubernetes to 1.30 before merging this PR.
- Review priority expander config if applicable.

---

### cert-manager `1.14.5` → `1.14.7`  🟢 LOW RISK

**Summary:** Patch release with bug fixes only.
No compatibility concerns with current platform state.

---

*Generated by [Upgrade Guard](https://github.com/...) •
Platform state: inferred from repo files*
```

**Additional output formats:**
- `json` — structured output for programmatic consumption / pipelines
- `stdout` — for local CLI usage and debugging

---

## 5. CI Adapters

### 5.1 GitHub Actions

```yaml
# .github/workflows/upgrade-guard.yml
name: Upgrade Guard
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  risk-assessment:
    # Only run on Renovate PRs (optional — can run on all PRs)
    if: contains(github.event.pull_request.labels.*.name, 'dependencies')
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
      contents: read
    steps:
      - uses: actions/checkout@v4

      - name: Run Upgrade Guard
        uses: docker://ghcr.io/your-org/upgrade-guard:latest
        env:
          UPGRADE_GUARD_PR_DIFF: ${{ github.event.pull_request.diff_url }}
          UPGRADE_GUARD_PR_NUMBER: ${{ github.event.pull_request.number }}
          UPGRADE_GUARD_REPO: ${{ github.repository }}
          UPGRADE_GUARD_OUTPUT: github-pr-comment
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

### 5.2 Azure DevOps

```yaml
# azure-pipelines/upgrade-guard.yml
trigger: none

pr:
  branches:
    include:
      - main

pool:
  vmImage: ubuntu-latest

steps:
  - checkout: self

  - script: |
      docker run --rm \
        -v $(Build.SourcesDirectory):/workspace \
        -e UPGRADE_GUARD_PR_DIFF="$(System.PullRequest.PullRequestId)" \
        -e UPGRADE_GUARD_OUTPUT=azure-pr-comment \
        -e UPGRADE_GUARD_ADO_ORG="$(System.CollectionUri)" \
        -e UPGRADE_GUARD_ADO_PROJECT="$(System.TeamProject)" \
        -e UPGRADE_GUARD_ADO_REPO="$(Build.Repository.Name)" \
        -e UPGRADE_GUARD_ADO_TOKEN="$(System.AccessToken)" \
        -e ANTHROPIC_API_KEY="$(ANTHROPIC_API_KEY)" \
        ghcr.io/your-org/upgrade-guard:latest
    displayName: Run Upgrade Guard
```

---

## 6. Configuration

### 6.1 `upgrade-guard.yaml` (repo root)

```yaml
# Full example with all options
level: standard               # basic | standard | full

platform:
  kubernetes_version: "1.29"
  provider: eks

components:
  cluster-autoscaler: "1.29.0"
  cert-manager: "1.14.5"

context: |
  Custom context for the LLM to consider.

# Only needed for 'full' level
cluster:
  kubeconfig_path: ""
  context: ""

settings:
  # LLM
  llm_model: auto             # auto | claude-sonnet-4-6 | claude-opus-4-6
  # API key via env var: ANTHROPIC_API_KEY

  # Output
  output_format: pr-comment   # pr-comment | json | stdout

  # Filtering
  ignore_components: []       # components to skip analysis on
  risk_threshold: low         # low | medium | high — minimum risk to report
```

### 6.2 Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ANTHROPIC_API_KEY` | Yes | Claude API key |
| `UPGRADE_GUARD_OUTPUT` | No | Output mode override |
| `UPGRADE_GUARD_CONFIG` | No | Path to config file (default: `./upgrade-guard.yaml`) |
| `GITHUB_TOKEN` | Auto | Set by GitHub Actions for PR comments |
| `UPGRADE_GUARD_ADO_TOKEN` | Yes† | Azure DevOps PAT (†if using ADO) |

---

## 7. Container Design

```dockerfile
# Build stage
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /upgrade-guard ./cmd/upgrade-guard

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /upgrade-guard /upgrade-guard
ENTRYPOINT ["/upgrade-guard"]
```

**Why Go:**
- Single static binary — no runtime dependencies, trivial to distribute
- Distroless/scratch container image — target <15MB
- Fast cold starts — ideal for FaaS (Lambda, Cloud Run, Azure Functions)
- Strong stdlib for HTTP clients, YAML/JSON parsing, and text templating
- Excellent cross-compilation for multi-arch images
- Native concurrency if we ever want to parallelize analysis of multi-component PRs

**Key dependencies:**
- `github.com/anthropics/anthropic-sdk-go` — official Anthropic Go SDK (Claude API + web search tool)
- `github.com/google/go-github/v68` — GitHub API SDK (PR comments, diff retrieval)
- `github.com/microsoft/azure-devops-go-api` — Azure DevOps SDK (PR comments, v0.2)
- `github.com/spf13/cobra` — CLI framework
- `github.com/charmbracelet/log` — structured logging
- `gopkg.in/yaml.v3` — config + manifest parsing
- `k8s.io/client-go` — Kubernetes client (only for Full guard level)

---

## 8. Project Structure

```
upgrade-guard/
├── cmd/
│   └── upgrade-guard/
│       └── main.go              # CLI entrypoint (cobra)
├── internal/
│   ├── config/
│   │   └── config.go            # Config loading + validation
│   ├── parser/
│   │   ├── diff.go              # Unified diff parsing
│   │   ├── renovate.go          # Renovate PR body/metadata parsing
│   │   └── detect.go            # Component type detection
│   ├── state/
│   │   ├── infer.go             # Tier 1: scan repo files
│   │   ├── configfile.go        # Tier 2: upgrade-guard.yaml
│   │   └── cluster.go           # Tier 3: kubectl queries
│   ├── analyst/
│   │   ├── prompt.go            # Prompt assembly + templates
│   │   ├── llm.go               # LLM client interface + model selection
│   │   ├── anthropic.go         # Claude API implementation (Sonnet + Opus)
│   │   └── models.go            # Risk assessment types
│   └── output/
│       ├── github.go            # GitHub PR comment poster
│       ├── azuredevops.go       # ADO PR comment poster
│       ├── json.go              # JSON output
│       └── stdout.go            # Terminal output
├── adapters/
│   ├── github-action/
│   │   └── action.yml
│   └── azure-devops/
│       └── task.json
├── Dockerfile
├── go.mod
├── go.sum
├── .goreleaser.yaml             # Multi-platform binary releases
├── upgrade-guard.yaml.example
└── README.md
```

---

## 9. Risks & Open Questions

| # | Topic | Question | Current Thinking |
|---|-------|----------|-----------------|
| 1 | LLM accuracy | What if the LLM hallucinates a compatibility issue? | Use structured output + web search grounding. Add disclaimer to comments. Could add a verification step later. |
| 2 | Cost | API cost per PR analysis? | Sonnet + web search should be <$0.05 per PR. Acceptable. |
| 3 | Rate limiting | Renovate can create many PRs at once | Debounce or batch — run once per push, not per PR event. |
| 4 | Scope creep | Should this also validate Terraform plans? | Out of scope for v0.1. Focus on K8s ecosystem first. |
| 5 | Naming | `upgrade-guard`? | Working name. Open to alternatives. |
| 6 | Repo ownership | Personal, VH & Co, or new org? | TBD. |
| 7 | License | Open source? What license? | Leaning open source — Apache 2.0 or MIT. |

---

## 10. MVP Scope (v0.1)

**In scope:**
- Parse Renovate PRs on GitHub
- Detect Helm chart and container image version bumps
- Infer state from repo files + `upgrade-guard.yaml`
- Claude API with web search for risk analysis
- Post markdown comment on GitHub PRs
- Docker container image

**Out of scope for v0.1:**
- Azure DevOps adapter (v0.2)
- Cluster query state loader (v0.2)
- Terraform module analysis
- Custom plugin/hook system
- Caching / deduplication of analysis
- UI dashboard

---

## 11. Next Steps

1. **Validate design** — review this doc, poke holes
2. **Prototype the prompt** — test with real Renovate PR diffs to see if the LLM produces useful output
3. **Scaffold the project** — repo, basic structure, Dockerfile
4. **Build PR parser** — start with Renovate Helm chart PRs
5. **Build state inferrer** — scan repo for K8s version + component versions
6. **Integrate LLM** — Claude API with web search
7. **GitHub Action adapter** — thin wrapper to trigger + post comment
8. **Test on real PRs** — your own repos as guinea pig
