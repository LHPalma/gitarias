package ignore

import (
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

func (repo *Repo) Ensure() error {
	return git.EnsureRepo(repo.runner)
}

func (repo *Repo) List(expand bool) ([]Entry, error) {
	candidates, err := repo.candidates(expand)
	if err != nil {
		return nil, err
	}
	if candidates == "" {
		return nil, nil
	}

	output, err := repo.runner.RunWithInput(candidates, "check-ignore", "-z", "--stdin", "-v")
	if err != nil {
		if nothingMatched(err) {
			return nil, nil
		}
		return nil, err
	}

	return parse(output), nil
}

func (repo *Repo) candidates(expand bool) (string, error) {
	args := []string{"ls-files", "--others", "--ignored", "--exclude-standard"}
	if !expand {
		args = append(args, "--directory", "--no-empty-directory")
	}

	return repo.runner.Run(append(args, "-z")...)
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
