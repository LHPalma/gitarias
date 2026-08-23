package cmd

type repositoryCommitCountRecord struct {
	Repository string `json:"repository"`
	Private    bool   `json:"private"`
	Commits    int    `json:"commits"`
}
