package cmd

type diagnosisDocument struct {
	Checks []checkRecord `json:"checks"`
}
