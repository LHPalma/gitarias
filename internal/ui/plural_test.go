package ui

import "testing"

func TestPluralTurnsOnlyOneIntoTheSingular(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  string
	}{
		{name: "nenhum", count: 0, want: "commits"},
		{name: "um", count: 1, want: "commit"},
		{name: "dois", count: 2, want: "commits"},
		{name: "muitos", count: 42, want: "commits"},
		{name: "negativo nao existe, mas nao pode virar singular", count: -1, want: "commits"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Plural(test.count, "commit", "commits"); got != test.want {
				t.Errorf("forma = %q, queria %q", got, test.want)
			}
		})
	}
}
