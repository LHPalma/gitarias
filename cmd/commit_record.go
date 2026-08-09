package cmd

type commitRecord struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Code    int    `json:"code"`
	Outcome string `json:"outcome"`
}
