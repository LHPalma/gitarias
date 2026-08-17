package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/LHPalma/gitarias/internal/forge"
	"github.com/LHPalma/gitarias/internal/ui"
)

type pullRequestsTable struct {
	requests []forge.PullRequest
}

func (data pullRequestsTable) header() []string {
	return []string{"número", "estado", "branch", "base", "autor", "título"}
}

func (data pullRequestsTable) rows() [][]string {
	rows := make([][]string, 0, len(data.requests))
	for _, request := range data.requests {
		rows = append(rows, []string{
			strconv.Itoa(request.Number),
			ui.DescribePullRequest(request),
			request.Head,
			request.Base,
			request.Author,
			request.Title,
		})
	}

	return rows
}

func (data pullRequestsTable) document() any {
	records := make([]pullRequestRecord, 0, len(data.requests))
	for _, request := range data.requests {
		records = append(records, pullRequestRecord{
			Number: request.Number,
			Title:  request.Title,
			Head:   request.Head,
			Base:   request.Base,
			Author: request.Author,
			State:  request.State,
			Draft:  request.Draft,
			URL:    request.URL,
		})
	}

	return pullRequestsDocument{PullRequests: records}
}

func (data pullRequestsTable) text(output io.Writer) error {
	if len(data.requests) == 0 {
		_, err := fmt.Fprintln(output, "Nenhum pull request aberto neste repositório.")
		return err
	}

	writer := columns(&trimmingWriter{output: output})
	for _, request := range data.requests {
		fmt.Fprintf(writer, "  #%d\t%s\t%s\t%s\n",
			request.Number, ui.DescribePullRequest(request), request.Head, request.Title)
	}

	return writer.Flush()
}
