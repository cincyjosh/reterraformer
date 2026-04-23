package cmd

import (
	"fmt"
	"os"

	"github.com/cincyjosh/reterraformer/pkg/agent"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agentic import: Claude reads your repo and generates idiomatic Terraform HCL",
	Long: `Launches a Claude claude-opus-4-7 agent that:
  1. Reads your repository to understand module conventions and layout.
  2. Enumerates all GCP resources in the target project.
  3. Writes idiomatic Terraform HCL (with import blocks) to the output directory.
  4. Iterates with terraform fmt/validate/plan until the output is clean.`,
	RunE: runAgent,
}

var (
	agentProject   string
	agentRepoDir   string
	agentOutputDir string
	agentAPIKey    string
)

func init() {
	agentCmd.Flags().StringVar(&agentProject, "project", "", "GCP project ID (required)")
	agentCmd.Flags().StringVar(&agentRepoDir, "repo", ".", "Path to the Terraform repository the agent will read for conventions")
	agentCmd.Flags().StringVar(&agentOutputDir, "output", "tf-output", "Directory where generated Terraform files are written")
	agentCmd.Flags().StringVar(&agentAPIKey, "api-key", "", "Anthropic API key (defaults to ANTHROPIC_API_KEY env var)")
	_ = agentCmd.MarkFlagRequired("project")

	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, _ []string) error {
	apiKey := agentAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("Anthropic API key required: set --api-key or ANTHROPIC_API_KEY")
	}

	fmt.Printf("Starting agent for project %q → output: %s\n", agentProject, agentOutputDir)
	return agent.Run(cmd.Context(), agentProject, agentRepoDir, agentOutputDir, apiKey)
}
