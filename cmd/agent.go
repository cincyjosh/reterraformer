package cmd

import (
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/cincyjosh/reterraformer/pkg/agent"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agentic import: Claude reads your repo and generates idiomatic Terraform HCL",
	Long: `Launches a Claude agent that:
  1. Reads your repository to understand module conventions and layout.
  2. Enumerates all GCP resources in the target project.
  3. Writes idiomatic Terraform HCL (with import blocks) to the output directory.
  4. Iterates with terraform fmt/validate/plan until the output is clean.

Model presets:
  --model=claude-opus-4-7   --effort=xhigh   (default, best for initial bulk imports)
  --model=claude-sonnet-4-6 --effort=high     (recommended for scheduled drift runs)`,
	RunE: runAgent,
}

var (
	agentProject   string
	agentRepoDir   string
	agentOutputDir string
	agentAPIKey    string
	agentModel     string
	agentEffort    string
	agentMaxTokens int64
)

func init() {
	agentCmd.Flags().StringVar(&agentProject, "project", "", "GCP project ID (required)")
	agentCmd.Flags().StringVar(&agentRepoDir, "repo", ".", "Path to the Terraform repository the agent will read for conventions")
	agentCmd.Flags().StringVar(&agentOutputDir, "output", "tf-output", "Directory where generated Terraform files are written")
	agentCmd.Flags().StringVar(&agentAPIKey, "api-key", "", "Anthropic API key (defaults to ANTHROPIC_API_KEY env var)")
	agentCmd.Flags().StringVar(&agentModel, "model", "", "Claude model ID (default: claude-opus-4-7)")
	agentCmd.Flags().StringVar(&agentEffort, "effort", "", "Reasoning effort: low, medium, high, xhigh, max (default: xhigh)")
	agentCmd.Flags().Int64Var(&agentMaxTokens, "max-tokens", 0, "Per-response output token ceiling (default: 32000)")
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

	cfg := agent.Config{
		Model:     agentModel,
		MaxTokens: agentMaxTokens,
	}
	if agentEffort != "" {
		cfg.Effort = anthropic.OutputConfigEffort(agentEffort)
	}

	fmt.Printf("Starting agent for project %q → output: %s\n", agentProject, agentOutputDir)
	return agent.Run(cmd.Context(), agentProject, agentRepoDir, agentOutputDir, apiKey, cfg)
}
