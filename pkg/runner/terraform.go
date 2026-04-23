package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner shells out to the terraform binary to init and generate HCL configurations.
type Runner struct {
	dir string
}

func New(dir string) *Runner {
	return &Runner{dir: dir}
}

func (r *Runner) Init(ctx context.Context) error {
	return r.run(ctx, "init")
}

// GenerateConfig runs terraform plan -generate-config-out to produce generated.tf.
func (r *Runner) GenerateConfig(ctx context.Context) error {
	absDir, err := filepath.Abs(r.dir)
	if err != nil {
		return err
	}
	genPath := filepath.Join(absDir, "generated.tf")
	return r.run(ctx, "plan", "-generate-config-out="+genPath)
}

func (r *Runner) run(ctx context.Context, args ...string) error {
	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		return fmt.Errorf("terraform binary not found in PATH — install from https://developer.hashicorp.com/terraform/downloads")
	}
	absDir, err := filepath.Abs(r.dir)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, tfPath, args...) //nolint:gosec
	cmd.Dir = absDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
