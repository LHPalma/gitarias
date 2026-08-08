package ignore

type Entry struct {
	Source    string
	Line      int
	Pattern   string
	Path      string
	Directory bool
}
