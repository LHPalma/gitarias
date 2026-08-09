package cmd

type worktreeRecord struct {
	Path           string `json:"path"`
	Current        bool   `json:"current"`
	Branch         string `json:"branch"`
	Head           string `json:"head"`
	Detached       bool   `json:"detached"`
	Bare           bool   `json:"bare"`
	Locked         bool   `json:"locked"`
	LockedReason   string `json:"lockedReason"`
	Prunable       bool   `json:"prunable"`
	PrunableReason string `json:"prunableReason"`
}
