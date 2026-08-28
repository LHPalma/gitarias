package cmd

type bisectedDocument struct {
	Base    string           `json:"base"`
	Command []string         `json:"command"`
	Total   int              `json:"total"`
	Tested  []bisectedRecord `json:"tested"`
	Culprit *commitRecord    `json:"culprit"`
}
