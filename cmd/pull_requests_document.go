package cmd

type pullRequestsDocument struct {
	PullRequests []pullRequestRecord `json:"pull_requests"`
}
