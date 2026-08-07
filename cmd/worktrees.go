package cmd

import (
	"fmt"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/worktree"
	"github.com/spf13/cobra"
)

func newWorktreesCommand(runner git.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "worktrees",
		Short: "Lista os working trees do repositório",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runWorktrees(command, worktree.NewRepo(runner))
		},
	}
}

func runWorktrees(command *cobra.Command, repo *worktree.Repo) error {
	output := command.OutOrStdout()

	if err := repo.Ensure(); err != nil {
		return err
	}

	worktrees, err := repo.List()
	if err != nil {
		return err
	}

	fmt.Fprintf(output, "Working trees (%d):\n", len(worktrees))

	writer := columns(output)
	for _, entry := range worktrees {
		marker := " "
		if entry.Current {
			marker = "*"
		}
		line := fmt.Sprintf("%s %s\t%s", marker, entry.Path, describeCheckout(entry))
		if state := describeState(entry); state != "" {
			line += "\t" + state
		}
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}

func describeCheckout(entry worktree.Worktree) string {
	switch {
	case entry.Bare:
		return "(bare)"
	case entry.Detached:
		return "(HEAD destacado em " + shortHead(entry.Head) + ")"
	case entry.Branch == "":
		return "(sem branch)"
	default:
		return entry.Branch
	}
}

func describeState(entry worktree.Worktree) string {
	var states []string
	if entry.Locked {
		states = append(states, withReason("trancado", entry.LockedReason))
	}
	if entry.Prunable {
		states = append(states, withReason("podável", entry.PrunableReason))
	}
	return strings.Join(states, ", ")
}

func withReason(state string, reason string) string {
	if reason == "" {
		return state
	}
	return state + ": " + reason
}

func shortHead(head string) string {
	if len(head) > 7 {
		return head[:7]
	}
	return head
}
