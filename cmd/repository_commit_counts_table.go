package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/forge"
)

type repositoryCommitCountsTable struct {
	counts []forge.RepositoryCommitCount
}

func (data repositoryCommitCountsTable) header() []string {
	return []string{"repositório", "privado", "commits"}
}

func (data repositoryCommitCountsTable) rows() [][]string {
	rows := make([][]string, 0, len(data.counts))
	for _, count := range data.counts {
		rows = append(rows, []string{count.Repository, strconv.FormatBool(count.Private), strconv.Itoa(count.Count)})
	}

	return rows
}

func (data repositoryCommitCountsTable) document() any {
	records := make([]repositoryCommitCountRecord, 0, len(data.counts))
	for _, count := range data.counts {
		records = append(records, repositoryCommitCountRecord{
			Repository: count.Repository,
			Private:    count.Private,
			Commits:    count.Count,
		})
	}

	return repositoryCommitCountsDocument{Repositories: records}
}

func (data repositoryCommitCountsTable) text(output io.Writer) error {
	if len(data.counts) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum commit contribuído nesse período.")
		return err
	}

	writer := columns(&trimmingWriter{output: output})
	for _, count := range data.counts {
		fmt.Fprintf(writer, "  %s\t%d\n", count.Repository, count.Count)
	}

	return writer.Flush()
}
