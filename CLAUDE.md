# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -v ./...          # build
go test ./...              # run tests
go mod tidy                # sync dependencies after changes

# Run the tool (requires gcloud auth application-default login)
go run . import --project=my-gcp-project --resources=google_compute_network,google_storage_bucket
go run . import --project=my-gcp-project --resources='*' --generate
go run . list
```

## Architecture

reterraformer enumerates existing cloud resources and generates Terraform [import blocks](https://developer.hashicorp.com/terraform/language/import) (Terraform 1.5+). When `--generate` is passed it shells out to `terraform plan -generate-config-out=generated.tf` so Terraform itself produces the full HCL — no manual field mapping needed.

### Import flow

1. `cmd/import.go` authenticates via Application Default Credentials and calls `gcp.NewRegistry` to get all enumerators.
2. For each requested resource type, `Enumerator.Enumerate(ctx, project)` calls the GCP API and returns `[]ResourceImport` (type + sanitized name + import ID).
3. `pkg/output.Writer` writes `versions.tf`, `provider.tf`, and `imports.tf` to the output directory.
4. If `--generate`: `pkg/runner.Runner` runs `terraform init` then `terraform plan -generate-config-out` in that directory.

### Adding a new GCP resource type

1. Add an unexported `*Enumerator` struct and its constructor in the appropriate file under `pkg/gcp/` (or a new file if it's a new API surface). Implement `TerraformType() string` and `Enumerate(ctx, project) ([]ResourceImport, error)`.
2. Register it in `pkg/gcp/provider.go:NewRegistry` — create the API service client if one doesn't exist yet, and add a map entry.
3. Add the type string to the slice in `pkg/gcp/service.go:SupportedTypes`.

### Adding a new cloud provider

Create a new package under `pkg/` (e.g., `pkg/aws/`). It needs the same `Enumerator` interface — either import it from `pkg/gcp` (rename the package) or move the shared types to `pkg/types`. Wire up a new `import` subcommand in `cmd/`.

### Key design decisions

- **Flat `pkg/gcp` package**: all GCP enumerators live in one package (compute.go, storage.go, gke.go, iam.go, sql.go) to avoid circular imports between parent and sub-packages.
- **No field mapping**: `Enumerate` returns only the ID; Terraform's provider handles all attribute resolution via `generate-config-out`.
- **Name deduplication**: `output.Writer.importsContent` appends `_N` suffixes when two resources sanitize to the same identifier.
- **Import ID formats** vary by resource type — check the Terraform google provider import docs when adding a new type.
