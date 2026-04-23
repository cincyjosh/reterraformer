package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Config controls model selection and effort level for the agentic loop.
type Config struct {
	// Model is the Claude model ID. Defaults to claude-opus-4-7.
	Model string
	// Effort controls reasoning depth. Valid values: low, medium, high, xhigh, max.
	// xhigh and max require Opus. Defaults to xhigh.
	Effort anthropic.OutputConfigEffort
	// MaxTokens is the per-response output ceiling. Defaults to 32000.
	MaxTokens int64
}

func (c *Config) setDefaults() {
	if c.Model == "" {
		c.Model = "claude-opus-4-7"
	}
	if c.Effort == "" {
		c.Effort = anthropic.OutputConfigEffortXhigh
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 32000
	}
}

// Run executes the agentic import loop.
func Run(ctx context.Context, project, repoDir, outputDir, apiKey string, cfg Config) error {
	cfg.setDefaults()
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	h := &handler{
		repoDir:   repoDir,
		outputDir: outputDir,
		project:   project,
	}

	system := []anthropic.TextBlockParam{
		{
			Text: fmt.Sprintf(`You are an expert Terraform engineer generating production-quality Terraform HCL for a GCP project.

Your job:
1. Read the target repository structure to understand existing module conventions, variable patterns, and file layout.
2. Enumerate GCP resources in project %q.
3. Write idiomatic Terraform HCL to the output directory that:
   - Organises resources into logical modules matching the repo's existing structure where possible.
   - Uses descriptive resource names derived from the actual GCP resource names.
   - Includes required provider and versions blocks.
   - Generates import blocks (Terraform 1.5+ syntax) so existing resources are adopted rather than recreated.
4. Run terraform fmt, then terraform validate, then terraform plan to verify correctness.
5. Fix any errors reported by those commands and re-validate until the output is clean.

Rules:
- Only write files to the output directory; never modify the source repository.
- Only run terraform subcommands: init, validate, plan, fmt.
- When writing import blocks, match the exact ID format required by the google Terraform provider for each resource type.
- Prefer locals and variables over hard-coded values where it makes the code more maintainable.
- Do not include sensitive values in generated files.

Start by listing the repository root, then read relevant files to understand conventions, then enumerate resources, then generate the HCL.`, project),
		},
	}
	system[len(system)-1].CacheControl = anthropic.NewCacheControlEphemeralParam()

	tools := toolDefs()
	messages := []anthropic.MessageParam{}

	messages = append(messages, anthropic.NewUserMessage(
		anthropic.NewTextBlock(fmt.Sprintf(
			"Generate Terraform HCL for all resources in GCP project %q. Write the output to the output directory and ensure terraform validate passes.",
			project,
		)),
	))

	fmt.Printf("model=%s effort=%s max_tokens=%d\n", cfg.Model, cfg.Effort, cfg.MaxTokens)

	for {
		resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     cfg.Model,
			MaxTokens: cfg.MaxTokens,
			System:    system,
			Tools:     tools,
			Messages:  messages,
			Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{}},
			OutputConfig: anthropic.OutputConfigParam{
				Effort: cfg.Effort,
			},
		})
		if err != nil {
			return fmt.Errorf("claude API error: %w", err)
		}

		for _, block := range resp.Content {
			switch v := block.AsAny().(type) {
			case anthropic.TextBlock:
				fmt.Fprintln(os.Stdout, v.Text)
			case anthropic.ThinkingBlock:
				_ = v
			}
		}

		messages = append(messages, resp.ToParam())

		if resp.StopReason != anthropic.StopReasonToolUse {
			break
		}

		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
			if !ok {
				continue
			}

			rawInput := toolUse.Input
			if len(rawInput) == 0 {
				rawInput = json.RawMessage("{}")
			}

			result, isError := h.dispatch(ctx, toolUse.Name, rawInput)
			fmt.Fprintf(os.Stdout, "[tool] %s → %s\n", toolUse.Name, truncate(result, 200))
			results = append(results, anthropic.NewToolResultBlock(toolUse.ID, result, isError))
		}

		messages = append(messages, anthropic.NewUserMessage(results...))
	}

	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
