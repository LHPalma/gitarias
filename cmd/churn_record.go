package cmd

type churnRecord struct {
	Path    string `json:"path"`
	Commits int    `json:"commits"`
	InTree  bool   `json:"in_tree"`
}
