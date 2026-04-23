package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cincyjosh/reterraformer/pkg/gcp"
)

// Writer writes Terraform import blocks and supporting files to a directory.
type Writer struct {
	dir     string
	project string
}

func NewWriter(dir, project string) *Writer {
	return &Writer{dir: dir, project: project}
}

// Write creates the output directory and writes versions.tf, provider.tf, and imports.tf.
func (w *Writer) Write(resources []gcp.ResourceImport) error {
	if err := os.MkdirAll(w.dir, 0750); err != nil {
		return err
	}
	for name, content := range map[string]string{
		"versions.tf": w.versionsContent(),
		"provider.tf": w.providerContent(),
		"imports.tf":  w.importsContent(resources),
	} {
		if err := os.WriteFile(filepath.Join(w.dir, name), []byte(content), 0600); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) versionsContent() string {
	return `terraform {
  required_version = ">= 1.5"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0"
    }
  }
}
`
}

func (w *Writer) providerContent() string {
	return fmt.Sprintf("provider \"google\" {\n  project = %q\n}\n", w.project)
}

func (w *Writer) importsContent(resources []gcp.ResourceImport) string {
	// Track used names per Terraform address to handle duplicate resource names.
	seen := make(map[string]int)
	var sb strings.Builder
	for _, r := range resources {
		key := r.TerraformType + "." + r.TerraformName
		n := seen[key]
		seen[key]++
		name := r.TerraformName
		if n > 0 {
			name = fmt.Sprintf("%s_%d", r.TerraformName, n)
		}
		fmt.Fprintf(&sb, "import {\n  id = %q\n  to = %s.%s\n}\n\n",
			r.ImportID, r.TerraformType, name)
	}
	return sb.String()
}
