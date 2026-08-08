package cmd

type ignoredRecord struct {
	Source  string `json:"source"`
	Line    int    `json:"line"`
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}
