package branch

type DeleteResult struct {
	Branch Branch
	SHA    string
	Err    error
}
