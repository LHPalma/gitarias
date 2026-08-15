package cmd

type weightDocument struct {
	Total  int64          `json:"total_bytes"`
	Weight []weightRecord `json:"weight"`
}
