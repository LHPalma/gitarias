package cmd

import (
	"fmt"
	"io"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/ui"
)

type branchesTable struct {
	base branch.Base
	free []branch.Branch
	held []heldBranch
}

func (data branchesTable) header() []string {
	return []string{"branch", "merge", "presa", "worktree"}
}

func (data branchesTable) rows() [][]string {
	rows := make([][]string, 0, len(data.free)+len(data.held))
	for _, candidate := range data.free {
		rows = append(rows, []string{candidate.Name, ui.DescribeMerge(candidate.Merge), yesNo(false), ""})
	}
	for _, entry := range data.held {
		rows = append(rows, []string{entry.Branch.Name, ui.DescribeMerge(entry.Branch.Merge), yesNo(true), entry.Path})
	}

	return rows
}

func (data branchesTable) document() any {
	records := make([]branchRecord, 0, len(data.free)+len(data.held))
	for _, candidate := range data.free {
		records = append(records, branchRecord{Branch: candidate.Name, Merge: candidate.Merge.String()})
	}
	for _, entry := range data.held {
		records = append(records, branchRecord{
			Branch:   entry.Branch.Name,
			Merge:    entry.Branch.Merge.String(),
			Held:     true,
			Worktree: entry.Path,
		})
	}

	return branchesDocument{
		Base:     baseRecord{Name: data.base.Name, Source: data.base.Source.String()},
		Branches: records,
	}
}

func (data branchesTable) text(output io.Writer) error {
	if data.empty() {
		_, err := fmt.Fprintln(output, "Nenhuma branch local mergeada para limpar.")
		return err
	}

	if len(data.free) > 0 {
		if err := printMerged(output, data.free); err != nil {
			return err
		}
	}

	if len(data.held) > 0 {
		if err := printHeld(output, data.held); err != nil {
			return err
		}
	}

	return nil
}

func (data branchesTable) empty() bool {
	return len(data.free) == 0 && len(data.held) == 0
}
