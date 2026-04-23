package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "reterraformer",
	Short: "Generate Terraform configurations from existing cloud infrastructure",
	Long: `reterraformer enumerates existing cloud resources and generates Terraform
import blocks, then optionally calls Terraform to produce complete HCL configurations.

Requires Application Default Credentials (gcloud auth application-default login)
and Terraform >= 1.5 when using --generate.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(listCmd)
}
