package config

import "testing"

func TestSourceString(t *testing.T) {
	tests := []struct {
		source Source
		want   string
	}{
		{SourceDefault, "default"},
		{SourcePersonal, "personal"},
		{SourceRepo, "repo"},
	}

	for _, test := range tests {
		if got := test.source.String(); got != test.want {
			t.Errorf("%d.String() = %q, queria %q", test.source, got, test.want)
		}
	}
}
