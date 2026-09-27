package config

import (
	"context"
	"os"
	"path/filepath"

	"github.com/LHPalma/gitarias/internal/git"
)

func personalConfigPath() (string, bool) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", false
	}

	return filepath.Join(dir, "gtr", "config.yaml"), true
}

func repoConfigPath(ctx context.Context, runner git.Runner) (string, bool) {
	root, err := runner.Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil || root == "" {
		return "", false
	}

	return filepath.Join(root, ".gtr.yaml"), true
}
