package undo

import "time"

// Entry é uma branch deletada, com a ponta que ela tinha no momento da
// deleção. É o que basta para recriá-la.
type Entry struct {
	Deleted time.Time
	SHA     string
	Name    string
}
