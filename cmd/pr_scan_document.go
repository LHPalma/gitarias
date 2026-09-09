package cmd

type pullRequestScanDocument struct {
	PullRequests []scannedPullRequestRecord `json:"pull_requests"`
}
