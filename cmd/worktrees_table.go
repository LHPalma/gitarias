package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/LHPalma/gitarias/internal/worktree"
)

type worktreesTable struct {
	worktrees []worktree.Worktree
}

func (data worktreesTable) header() []string {
	return []string{
		"caminho", "atual", "branch", "head", "destacado", "bare",
		"trancado", "motivo_trancado", "podável", "motivo_podável",
	}
}

func (data worktreesTable) rows() [][]string {
	rows := make([][]string, 0, len(data.worktrees))
	for _, entry := range data.worktrees {
		rows = append(rows, []string{
			entry.Path,
			yesNo(entry.Current),
			entry.Branch,
			entry.Head,
			yesNo(entry.Detached),
			yesNo(entry.Bare),
			yesNo(entry.Locked),
			entry.LockedReason,
			yesNo(entry.Prunable),
			entry.PrunableReason,
		})
	}

	return rows
}

func (data worktreesTable) document() any {
	records := make([]worktreeRecord, 0, len(data.worktrees))
	for _, entry := range data.worktrees {
		records = append(records, worktreeRecord{
			Path:           entry.Path,
			Current:        entry.Current,
			Branch:         entry.Branch,
			Head:           entry.Head,
			Detached:       entry.Detached,
			Bare:           entry.Bare,
			Locked:         entry.Locked,
			LockedReason:   entry.LockedReason,
			Prunable:       entry.Prunable,
			PrunableReason: entry.PrunableReason,
		})
	}

	return worktreesDocument{Worktrees: records}
}

func (data worktreesTable) text(output io.Writer) error {
	fmt.Fprintf(output, "Working trees (%d):\n", len(data.worktrees))

	writer := columns(output)
	for _, entry := range data.worktrees {
		marker := " "
		if entry.Current {
			marker = "*"
		}
		line := fmt.Sprintf("%s %s\t%s", marker, entry.Path, ui.DescribeCheckout(entry))
		if state := ui.DescribeState(entry); state != "" {
			line += "\t" + state
		}
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}
