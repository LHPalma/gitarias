package ignore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

const separator = "\x00"

type Repo struct {
	runner Runner
}

func NewRepo(runner Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

func (repo *Repo) List(ctx context.Context, expand bool, expandDirs []string) ([]Entry, error) {
	candidates, err := repo.candidates(ctx, expand, expandDirs)
	if err != nil {
		return nil, err
	}
	if candidates == "" {
		return nil, nil
	}

	output, err := repo.runner.RunWithInput(ctx, candidates, "check-ignore", "-z", "--stdin", "-v")
	if err != nil {
		if nothingMatched(err) {
			return nil, nil
		}
		return nil, err
	}

	return parse(output), nil
}

func (repo *Repo) candidates(ctx context.Context, expand bool, expandDirs []string) (string, error) {
	if len(expandDirs) > 0 {
		return repo.candidatesExpandingDirs(ctx, expandDirs)
	}

	args := []string{"ls-files", "--others", "--ignored", "--exclude-standard"}
	if !expand {
		args = append(args, "--directory", "--no-empty-directory")
	}

	return repo.runner.Run(ctx, append(args, "-z")...)
}

func (repo *Repo) candidatesExpandingDirs(ctx context.Context, expandDirs []string) (string, error) {
	collapsed, err := repo.runner.Run(ctx, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "--no-empty-directory", "-z")
	if err != nil {
		return "", err
	}

	wanted := make(map[string]bool, len(expandDirs))
	for _, dir := range expandDirs {
		if !strings.HasSuffix(dir, "/") {
			dir += "/"
		}
		wanted[dir] = true
	}

	var combined strings.Builder
	var collapsedDirs []string
	found := make(map[string]bool, len(wanted))

	for _, path := range splitCandidates(collapsed) {
		if strings.HasSuffix(path, "/") {
			collapsedDirs = append(collapsedDirs, path)
		}

		if wanted[path] {
			found[path] = true

			expanded, err := repo.runner.Run(ctx, "ls-files", "--others", "--ignored", "--exclude-standard", "-z", "--", path)
			if err != nil {
				return "", err
			}

			combined.WriteString(expanded)
			continue
		}

		combined.WriteString(path)
		combined.WriteString(separator)
	}

	var missing []string
	for dir := range wanted {
		if !found[dir] {
			missing = append(missing, dir)
		}
	}
	if len(missing) > 0 {
		return "", notCollapsedError(missing, collapsedDirs)
	}

	return combined.String(), nil
}

func notCollapsedError(missing []string, collapsedDirs []string) error {
	sort.Strings(missing)

	if len(collapsedDirs) == 0 {
		return fmt.Errorf("--expand-dir %s: nada está colapsado nesta listagem", strings.Join(missing, ", "))
	}

	sort.Strings(collapsedDirs)

	return fmt.Errorf("--expand-dir %s: não está entre os diretórios colapsados; hoje são %s",
		strings.Join(missing, ", "), strings.Join(collapsedDirs, ", "))
}

func splitCandidates(output string) []string {
	if output == "" {
		return nil
	}

	fields := strings.Split(output, separator)
	if fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}

	return fields
}

func nothingMatched(err error) bool {
	var exitError *git.ExitError

	return errors.As(err, &exitError) && exitError.Code == 1
}

func parse(output string) []Entry {
	fields := strings.Split(output, separator)

	var entries []Entry
	for index := 0; index+3 < len(fields); index += 4 {
		line, err := strconv.Atoi(fields[index+1])
		if err != nil {
			continue
		}

		path := fields[index+3]
		entries = append(entries, Entry{
			Source:    fields[index],
			Line:      line,
			Pattern:   fields[index+2],
			Path:      path,
			Directory: strings.HasSuffix(path, "/"),
		})
	}

	return entries
}
