package ui

import (
	"testing"

	"github.com/LHPalma/gitarias/internal/config"
)

func TestDescribeConfigSource(t *testing.T) {
	tests := []struct {
		source config.Source
		want   string
	}{
		{config.SourceRepo, "arquivo do repo"},
		{config.SourcePersonal, "arquivo pessoal"},
		{config.SourceDefault, "padrão embutido"},
	}

	for _, test := range tests {
		if got := DescribeConfigSource(test.source); got != test.want {
			t.Errorf("DescribeConfigSource(%v) = %q, queria %q", test.source, got, test.want)
		}
	}
}
