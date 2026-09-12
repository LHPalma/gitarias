package cmd

type scannedPullRequestRecord struct {
	Number       int                 `json:"number"`
	Title        string              `json:"title"`
	URL          string              `json:"url"`
	Attributions []attributionRecord `json:"attributions"`
}

type attributionRecord struct {
	Tool string `json:"tool"`
	Line string `json:"line"`
}
