# reterraformer

A reverse-Terraform tool that generates Terraform configurations from existing cloud infrastructure. Built as a replacement for the archived [GoogleCloudPlatform/terraformer](https://github.com/GoogleCloudPlatform/terraformer), using Terraform's native import blocks (1.5+) instead of embedding provider binaries — so it doesn't have the protocol-version lock-in that killed the original.

## How it works

reterraformer enumerates live cloud resources via provider APIs and produces Terraform [import blocks](https://developer.hashicorp.com/terraform/language/import). When `--generate` is passed it shells out to `terraform plan -generate-config-out` so Terraform itself produces the full HCL — no manual field mapping needed.

The `agent` subcommand goes further: a Claude claude-opus-4-7 agent reads your repository to understand existing module conventions, then generates idiomatic HCL organized by resource type and iterates with `terraform validate` and `plan` until the output is clean.

## Prerequisites

- Go 1.25+
- Terraform 1.5+ (in PATH)
- GCP: `gcloud auth application-default login`
- Anthropic API key (for `agent` subcommand)

## Installation

```bash
git clone https://github.com/cincyjosh/reterraformer
cd reterraformer
go build -o reterraformer .
```

## Usage

### List supported resource types

```bash
reterraformer list
```

### Generate import blocks

```bash
# Specific resource types
reterraformer import --project=my-gcp-project --resources=google_compute_network,google_storage_bucket

# All supported types, with full HCL generation
reterraformer import --project=my-gcp-project --resources='*' --generate
```

Output goes to `./tf-output/` by default (`--output` to change). Contains `versions.tf`, `provider.tf`, and `imports.tf`.

### Agentic import (Claude-powered)

```bash
export ANTHROPIC_API_KEY=sk-ant-...

reterraformer agent \
  --project=my-gcp-project \
  --repo=/path/to/your/terraform-repo \
  --output=tf-output
```

The agent:
1. Reads `--repo` to understand your existing module structure and conventions.
2. Enumerates all GCP resources in the project.
3. Writes organized HCL files to `--output` with import blocks.
4. Runs `terraform fmt`, `validate`, and `plan`, fixing errors until the output is clean.

`--repo` defaults to `.` (current directory). The agent never modifies the source repo — it only writes to `--output`.

## Supported GCP resources

| Terraform type | API |
|---|---|
| `google_compute_network` | Compute |
| `google_compute_subnetwork` | Compute |
| `google_compute_instance` | Compute |
| `google_compute_firewall` | Compute |
| `google_storage_bucket` | Storage |
| `google_container_cluster` | GKE |
| `google_service_account` | IAM |
| `google_sql_database_instance` | Cloud SQL |

## Adding a new GCP resource type

1. Add an `*Enumerator` struct in the appropriate file under `pkg/gcp/` (or a new file for a new API surface). Implement `TerraformType() string` and `Enumerate(ctx, project) ([]ResourceImport, error)`.
2. Register it in `pkg/gcp/provider.go:NewRegistry`.
3. Add the type string to `pkg/gcp/service.go:SupportedTypes`.

Import ID formats vary by resource — check the [Terraform google provider import docs](https://registry.terraform.io/providers/hashicorp/google/latest/docs) for each type.

## Adding a new cloud provider

Create a new package under `pkg/` (e.g., `pkg/aws/`). Implement the same `Enumerator` interface and wire up a new subcommand in `cmd/`. Move shared types to `pkg/types` if needed to avoid circular imports.
