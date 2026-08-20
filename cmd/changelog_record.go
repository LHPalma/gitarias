package cmd

type changelogRecord struct {
	Hash     string `json:"hash"`
	Type     string `json:"type"`
	Scope    string `json:"scope"`
	Breaking bool   `json:"breaking"`
	Subject  string `json:"subject"`
}
