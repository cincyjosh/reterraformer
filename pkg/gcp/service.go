package gcp

import (
	"context"
	"regexp"
	"strings"
)

// ResourceImport represents a single cloud resource to be imported into Terraform.
type ResourceImport struct {
	TerraformType string // e.g., "google_compute_network"
	TerraformName string // sanitized Terraform identifier, e.g., "my_network"
	ImportID      string // the ID string Terraform's import block expects
}

// Enumerator lists cloud resources of a specific type.
type Enumerator interface {
	TerraformType() string
	Enumerate(ctx context.Context, project string) ([]ResourceImport, error)
}

var invalidIdentChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// SanitizeName converts a cloud resource name to a valid Terraform identifier.
func SanitizeName(name string) string {
	s := strings.NewReplacer("-", "_", ".", "_").Replace(name)
	s = invalidIdentChars.ReplaceAllString(s, "_")
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		return "r_" + s
	}
	return s
}

// SupportedTypes returns the Terraform resource type names this tool can enumerate.
func SupportedTypes() []string {
	return []string{
		"google_compute_firewall",
		"google_compute_instance",
		"google_compute_network",
		"google_compute_subnetwork",
		"google_container_cluster",
		"google_service_account",
		"google_sql_database_instance",
		"google_storage_bucket",
	}
}
