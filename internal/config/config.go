package config

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

type Config struct {
	Branches Branches
}

type Entry struct {
	Key    string
	Value  string
	Source Source
}

func (cfg Config) Entries() []Entry {
	return []Entry{
		{Key: "branches.protected", Value: strings.Join(cfg.Branches.Protected, ", "), Source: cfg.Branches.ProtectedSource},
		{Key: "branches.base", Value: cfg.Branches.Base, Source: cfg.Branches.BaseSource},
	}
}

func Load(ctx context.Context, runner git.Runner) (Config, error) {
	var personal, repo fileConfig

	if personalPath, ok := personalConfigPath(); ok {
		parsed, err := readFile(personalPath)
		if err != nil {
			return Config{}, err
		}
		personal = parsed
	}

	if repoPath, ok := repoConfigPath(ctx, runner); ok {
		parsed, err := readFile(repoPath)
		if err != nil {
			return Config{}, err
		}
		repo = parsed
	}

	return merge(personal, repo), nil
}

func readFile(path string) (fileConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileConfig{}, nil
		}
		return fileConfig{}, err
	}

	return parseFile(path, content)
}

func merge(personal, repo fileConfig) Config {
	var cfg Config

	cfg.Branches.Protected, cfg.Branches.ProtectedSource = mergeList(repo.Branches.Protected, personal.Branches.Protected)
	cfg.Branches.Base, cfg.Branches.BaseSource = mergeString(repo.Branches.Base, personal.Branches.Base)

	return cfg
}

func mergeList(repoValue, personalValue []string) ([]string, Source) {
	if len(repoValue) > 0 {
		return repoValue, SourceRepo
	}
	if len(personalValue) > 0 {
		return personalValue, SourcePersonal
	}
	return nil, SourceDefault
}

func mergeString(repoValue, personalValue string) (string, Source) {
	if repoValue != "" {
		return repoValue, SourceRepo
	}
	if personalValue != "" {
		return personalValue, SourcePersonal
	}
	return "", SourceDefault
}
