package cmd

type bisectedRecord struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Code    int    `json:"code"`
	Outcome string `json:"outcome"`
	Culprit bool   `json:"culprit"`
}
