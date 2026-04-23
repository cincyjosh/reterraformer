package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/cincyjosh/reterraformer/pkg/gcp"
	"golang.org/x/oauth2/google"
)

// toolDefs returns the tool definitions sent to Claude with prompt-caching markers.
func toolDefs() []anthropic.ToolUnionParam {
	tools := []anthropic.ToolParam{
		{
			Name:        "enumerate_resources",
			Description: anthropic.String("List all GCP resources of specified types in the project. Returns resource names and import IDs."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]interface{}{
					"project": map[string]interface{}{
						"type":        "string",
						"description": "GCP project ID",
					},
					"resource_types": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Terraform resource types to enumerate, e.g. [\"google_compute_network\", \"google_storage_bucket\"]. Use [\"*\"] for all.",
					},
				},
				Required: []string{"project", "resource_types"},
			},
		},
		{
			Name:        "read_file",
			Description: anthropic.String("Read a file from the local repository. Paths must be relative to the repo root."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path relative to the repo root",
					},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "list_directory",
			Description: anthropic.String("List files and directories at a path relative to the repo root."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path relative to the repo root (use \".\" for root)",
					},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: anthropic.String("Write content to a file in the output directory. Paths must be relative to the output directory."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path relative to the output directory",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "File content to write",
					},
				},
				Required: []string{"path", "content"},
			},
		},
		{
			Name:        "run_terraform",
			Description: anthropic.String("Run a terraform command in the output directory. Allowed commands: init, validate, plan, fmt."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]interface{}{
					"args": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Terraform subcommand and arguments, e.g. [\"validate\"] or [\"fmt\", \"-check\"]",
					},
				},
				Required: []string{"args"},
			},
		},
	}

	// Wrap in ToolUnionParam and add cache_control to the last tool so the
	// entire tool list is cached as one prefix.
	out := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		tool := t
		if i == len(tools)-1 {
			tool.CacheControl = anthropic.NewCacheControlEphemeralParam()
		}
		out[i] = anthropic.ToolUnionParam{OfTool: &tool}
	}
	return out
}

// handler holds the dependencies needed to execute tool calls.
type handler struct {
	repoDir   string
	outputDir string
	project   string
}

// dispatch executes the named tool and returns (result, isError).
func (h *handler) dispatch(ctx context.Context, name string, rawInput json.RawMessage) (string, bool) {
	switch name {
	case "enumerate_resources":
		return h.enumerateResources(ctx, rawInput)
	case "read_file":
		return h.readFile(rawInput)
	case "list_directory":
		return h.listDirectory(rawInput)
	case "write_file":
		return h.writeFile(rawInput)
	case "run_terraform":
		return h.runTerraform(ctx, rawInput)
	default:
		return fmt.Sprintf("unknown tool: %s", name), true
	}
}

func (h *handler) enumerateResources(ctx context.Context, raw json.RawMessage) (string, bool) {
	var input struct {
		Project       string   `json:"project"`
		ResourceTypes []string `json:"resource_types"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return err.Error(), true
	}

	httpClient, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform.read-only")
	if err != nil {
		return fmt.Sprintf("auth error: %v", err), true
	}

	registry, err := gcp.NewRegistry(httpClient)
	if err != nil {
		return fmt.Sprintf("registry error: %v", err), true
	}

	types := input.ResourceTypes
	if len(types) == 1 && types[0] == "*" {
		types = gcp.SupportedTypes()
	}

	var results []gcp.ResourceImport
	var errs []string
	for _, t := range types {
		enum, ok := registry[t]
		if !ok {
			errs = append(errs, fmt.Sprintf("unsupported type: %s", t))
			continue
		}
		res, err := enum.Enumerate(ctx, input.Project)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", t, err))
			continue
		}
		results = append(results, res...)
	}

	b, _ := json.MarshalIndent(map[string]interface{}{
		"resources": results,
		"errors":    errs,
	}, "", "  ")
	return string(b), false
}

func (h *handler) readFile(raw json.RawMessage) (string, bool) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return err.Error(), true
	}
	safe, err := h.safeRepoPath(input.Path)
	if err != nil {
		return err.Error(), true
	}
	content, err := os.ReadFile(safe)
	if err != nil {
		return err.Error(), true
	}
	return string(content), false
}

func (h *handler) listDirectory(raw json.RawMessage) (string, bool) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return err.Error(), true
	}
	safe, err := h.safeRepoPath(input.Path)
	if err != nil {
		return err.Error(), true
	}
	entries, err := os.ReadDir(safe)
	if err != nil {
		return err.Error(), true
	}
	var lines []string
	for _, e := range entries {
		suffix := ""
		if e.IsDir() {
			suffix = "/"
		}
		lines = append(lines, e.Name()+suffix)
	}
	return strings.Join(lines, "\n"), false
}

func (h *handler) writeFile(raw json.RawMessage) (string, bool) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return err.Error(), true
	}
	safe, err := h.safeOutputPath(input.Path)
	if err != nil {
		return err.Error(), true
	}
	if err := os.MkdirAll(filepath.Dir(safe), 0750); err != nil {
		return err.Error(), true
	}
	if err := os.WriteFile(safe, []byte(input.Content), 0600); err != nil {
		return err.Error(), true
	}
	return fmt.Sprintf("wrote %s", input.Path), false
}

var allowedTerraformSubcommands = map[string]bool{
	"init":     true,
	"validate": true,
	"plan":     true,
	"fmt":      true,
}

func (h *handler) runTerraform(ctx context.Context, raw json.RawMessage) (string, bool) {
	var input struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return err.Error(), true
	}
	if len(input.Args) == 0 {
		return "args must not be empty", true
	}
	if !allowedTerraformSubcommands[input.Args[0]] {
		return fmt.Sprintf("subcommand %q not allowed; permitted: init, validate, plan, fmt", input.Args[0]), true
	}

	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		return "terraform binary not found in PATH", true
	}
	absDir, err := filepath.Abs(h.outputDir)
	if err != nil {
		return err.Error(), true
	}

	cmd := exec.CommandContext(ctx, tfPath, input.Args...) //nolint:gosec
	cmd.Dir = absDir
	out, err := cmd.CombinedOutput()
	result := string(out)
	if err != nil {
		return result, true
	}
	return result, false
}

// safeRepoPath resolves a relative path under the repo root, rejecting traversals.
func (h *handler) safeRepoPath(rel string) (string, error) {
	return safeJoin(h.repoDir, rel)
}

// safeOutputPath resolves a relative path under the output dir, rejecting traversals.
func (h *handler) safeOutputPath(rel string) (string, error) {
	return safeJoin(h.outputDir, rel)
}

func safeJoin(base, rel string) (string, error) {
	abs, err := filepath.Abs(filepath.Join(base, rel))
	if err != nil {
		return "", err
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, absBase+string(filepath.Separator)) && abs != absBase {
		return "", fmt.Errorf("path %q escapes base directory", rel)
	}
	return abs, nil
}
