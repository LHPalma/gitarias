package cmd

type changelogDocument struct {
	Entries []changelogRecord `json:"entries"`
}
