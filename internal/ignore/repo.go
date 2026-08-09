package ignore

import (
	"context"
	"errors"
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

func (repo *Repo) List(ctx context.Context, expand bool) ([]Entry, error) {
	candidates, err := repo.candidates(ctx, expand)
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

func (repo *Repo) candidates(ctx context.Context, expand bool) (string, error) {
	args := []string{"ls-files", "--others", "--ignored", "--exclude-standard"}
	if !expand {
		args = append(args, "--directory", "--no-empty-directory")
	}

	return repo.runner.Run(ctx, append(args, "-z")...)
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
