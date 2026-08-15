package cmd

type weightRecord struct {
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Versions int    `json:"versions"`
	InTree   bool   `json:"in_tree"`
}
