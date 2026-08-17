package cmd

import (
	"strings"

	"github.com/LHPalma/gitarias/internal/ignore"
	"github.com/spf13/cobra"
)

func completeExpandDir(runner Runner) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(command *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		repo := ignore.NewRepo(runner)
		ctx := command.Context()

		if err := repo.Ensure(ctx); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		entries, err := repo.List(ctx, false, nil)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var directories []string
		for _, entry := range entries {
			if entry.Directory && strings.HasPrefix(entry.Path, toComplete) {
				directories = append(directories, entry.Path)
			}
		}

		return directories, cobra.ShellCompDirectiveNoFileComp
	}
}
