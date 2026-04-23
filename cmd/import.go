package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/cincyjosh/reterraformer/pkg/gcp"
	"github.com/cincyjosh/reterraformer/pkg/output"
	"github.com/cincyjosh/reterraformer/pkg/runner"
	"github.com/spf13/cobra"
)

var (
	project   string
	resources []string
	outDir    string
	generate  bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import cloud resources and generate Terraform import blocks",
	RunE:  runImport,
}

func init() {
	importCmd.Flags().StringVarP(&project, "project", "p", "", "GCP project ID (required)")
	importCmd.Flags().StringSliceVarP(&resources, "resources", "r", nil, "Resource types to import, comma-separated or * for all")
	importCmd.Flags().StringVarP(&outDir, "output", "o", "./generated", "Output directory")
	importCmd.Flags().BoolVar(&generate, "generate", false, "Run terraform init and plan -generate-config-out after writing import blocks")
	_ = importCmd.MarkFlagRequired("project")
}

func runImport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	client, err := gcp.NewHTTPClient(ctx)
	if err != nil {
		return fmt.Errorf("authentication: %w\n\nRun: gcloud auth application-default login", err)
	}

	registry, err := gcp.NewRegistry(client)
	if err != nil {
		return fmt.Errorf("initializing registry: %w", err)
	}

	enumerators, err := selectEnumerators(registry, resources)
	if err != nil {
		return err
	}

	var all []gcp.ResourceImport
	for _, e := range enumerators {
		log.Printf("enumerating %s...", e.TerraformType())
		found, err := e.Enumerate(ctx, project)
		if err != nil {
			log.Printf("warning: %s: %v", e.TerraformType(), err)
			continue
		}
		log.Printf("  found %d resource(s)", len(found))
		all = append(all, found...)
	}

	if len(all) == 0 {
		fmt.Println("no resources found")
		return nil
	}

	w := output.NewWriter(outDir, project)
	if err := w.Write(all); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	fmt.Printf("wrote %d import block(s) to %s\n", len(all), filepath.Join(outDir, "imports.tf"))

	if generate {
		r := runner.New(outDir)
		fmt.Println("running terraform init...")
		if err := r.Init(ctx); err != nil {
			return fmt.Errorf("terraform init: %w", err)
		}
		fmt.Println("running terraform plan -generate-config-out...")
		if err := r.GenerateConfig(ctx); err != nil {
			return fmt.Errorf("terraform plan: %w", err)
		}
		fmt.Printf("generated HCL written to %s\n", filepath.Join(outDir, "generated.tf"))
	}

	return nil
}

func selectEnumerators(registry map[string]gcp.Enumerator, requested []string) ([]gcp.Enumerator, error) {
	if len(requested) == 0 || (len(requested) == 1 && requested[0] == "*") {
		out := make([]gcp.Enumerator, 0, len(registry))
		for _, e := range registry {
			out = append(out, e)
		}
		return out, nil
	}
	var out []gcp.Enumerator
	for _, name := range requested {
		e, ok := registry[name]
		if !ok {
			return nil, fmt.Errorf("unknown resource type %q — run 'reterraformer list' to see supported types", name)
		}
		out = append(out, e)
	}
	return out, nil
}
