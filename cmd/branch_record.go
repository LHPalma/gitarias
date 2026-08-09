package cmd

type branchRecord struct {
	Branch   string `json:"branch"`
	Merge    string `json:"merge"`
	Held     bool   `json:"held"`
	Worktree string `json:"worktree"`
}
