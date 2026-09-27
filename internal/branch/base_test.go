package branch

import "testing"

func TestBaseSourceString(t *testing.T) {
	tests := []struct {
		source BaseSource
		want   string
	}{
		{BaseFromFlag, "flag"},
		{BaseFromConfig, "config"},
		{BaseFromOriginHead, "origin-head"},
		{BaseFromLocal, "local"},
	}

	for _, test := range tests {
		if got := test.source.String(); got != test.want {
			t.Errorf("%d.String() = %q, queria %q", test.source, got, test.want)
		}
	}
}
