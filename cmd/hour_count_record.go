package cmd

type hourCountRecord struct {
	Hour    int `json:"hour"`
	Commits int `json:"commits"`
}
