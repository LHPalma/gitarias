package cmd

type hourCountsDocument struct {
	Hours []hourCountRecord `json:"hours"`
}
