package cmd

type repositoryCommitCountsDocument struct {
	Repositories []repositoryCommitCountRecord `json:"repositories"`
}
