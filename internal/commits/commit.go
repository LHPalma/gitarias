package commits

type Commit struct {
	SHA     string
	Subject string
}

func (commit Commit) Short() string {
	if len(commit.SHA) > 7 {
		return commit.SHA[:7]
	}

	return commit.SHA
}
