package cmd

type checkRecord struct {
	Check  string `json:"check"`
	State  string `json:"state"`
	Detail string `json:"detail"`
	Hint   string `json:"hint"`
}
