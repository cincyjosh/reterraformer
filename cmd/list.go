package cmd

import (
	"fmt"
	"sort"

	"github.com/cincyjosh/reterraformer/pkg/gcp"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List supported resource types",
	RunE: func(cmd *cobra.Command, args []string) error {
		types := gcp.SupportedTypes()
		sort.Strings(types)
		fmt.Println("Supported GCP resource types:")
		for _, t := range types {
			fmt.Printf("  %s\n", t)
		}
		return nil
	},
}
