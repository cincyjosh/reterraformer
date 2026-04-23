# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -v ./...          # build
go test ./...              # run tests
go mod tidy                # sync dependencies after changes

# Static import (requires gcloud auth application-default login)
go run . import --project=my-gcp-project --resources=google_compute_network,google_storage_bucket
go run . import --project=my-gcp-project --resources='*' --generate
go run . list

# Agentic import (requires ANTHROPIC_API_KEY + gcloud ADC)
go run . agent --project=my-gcp-project --repo=. --output=tf-output
```

## Architecture

reterraformer has two modes for generating Terraform from live cloud infrastructure:

**Static mode** (`import` subcommand): deterministic — enumerates resources, writes import blocks, optionally shells out to `terraform plan -generate-config-out` for full HCL.

**Agentic mode** (`agent` subcommand): uses a Claude claude-opus-4-7 agent that reads the target repo to understand module conventions, enumerates resources, writes organized HCL files, and iterates with `terraform fmt/validate/plan` until the output is clean.

### Static import flow

1. `cmd/import.go` authenticates via Application Default Credentials and calls `gcp.NewRegistry`.
2. For each requested resource type, `Enumerator.Enumerate(ctx, project)` calls the GCP API and returns `[]ResourceImport` (type + sanitized name + import ID).
3. `pkg/output.Writer` writes `versions.tf`, `provider.tf`, and `imports.tf` to the output directory.
4. If `--generate`: `pkg/runner.Runner` runs `terraform init` then `terraform plan -generate-config-out`.

### Agentic loop (`pkg/agent/`)

`pkg/agent/agent.go:Run` is a manual Claude tool-use loop:
- Sends system prompt (prompt-cached) + 5 tool definitions (prompt-cached on last tool) to `claude-opus-4-7` with adaptive thinking.
- Loops: execute tool calls → feed results back → repeat until `StopReason != tool_use`.
- Prints Claude's text blocks to stdout as progress; `[tool] name → result` lines show each tool call.

`pkg/agent/tools.go` defines the 5 tools and their handlers:
- `enumerate_resources` — calls `gcp.NewRegistry` + ADC, same enumerators as static mode.
- `read_file` / `list_directory` — repo-scoped; path traversal rejected by `safeJoin`.
- `write_file` — output-dir-scoped only; cannot touch the source repo.
- `run_terraform` — allowlisted to `init`, `validate`, `plan`, `fmt` only.

### Adding a new GCP resource type

1. Add an unexported `*Enumerator` struct and its constructor in the appropriate file under `pkg/gcp/` (or a new file for a new API surface). Implement `TerraformType() string` and `Enumerate(ctx, project) ([]ResourceImport, error)`.
2. Register it in `pkg/gcp/provider.go:NewRegistry` — create the API service client if one doesn't exist yet, and add a map entry.
3. Add the type string to the slice in `pkg/gcp/service.go:SupportedTypes`.

### Adding a new cloud provider

Create a new package under `pkg/` (e.g., `pkg/aws/`). It needs the same `Enumerator` interface — either import it from `pkg/gcp` (rename the package) or move the shared types to `pkg/types`. Wire up a new `import` subcommand in `cmd/`.

### Key design decisions

- **Flat `pkg/gcp` package**: all GCP enumerators live in one package (compute.go, storage.go, gke.go, iam.go, sql.go) to avoid circular imports between parent and sub-packages.
- **No field mapping**: `Enumerate` returns only the ID; Terraform's provider handles all attribute resolution via `generate-config-out`.
- **Name deduplication**: `output.Writer.importsContent` appends `_N` suffixes when two resources sanitize to the same identifier.
- **Import ID formats** vary by resource type — check the Terraform google provider import docs when adding a new type.
- **Agent security**: tool handlers enforce path confinement (`safeJoin`) and a terraform subcommand allowlist — never relax these without understanding the blast radius.
