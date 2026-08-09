package cmd

type checkedDocument struct {
	Base    string         `json:"base"`
	Command []string       `json:"command"`
	Commits []commitRecord `json:"commits"`
}
