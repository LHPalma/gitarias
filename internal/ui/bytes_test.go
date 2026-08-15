package ui

import "testing"

func TestDescribeBytesClimbsTheUnits(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "zero", size: 0, want: "0 B"},
		{name: "abaixo de um KB fica em bytes", size: 1023, want: "1023 B"},
		{name: "um KB", size: 1024, want: "1.0 KB"},
		{name: "quase um MB", size: 1024*1024 - 1, want: "1024.0 KB"},
		{name: "um MB", size: 1024 * 1024, want: "1.0 MB"},
		{name: "tres megas", size: 3000000, want: "2.9 MB"},
		{name: "um GB", size: 1024 * 1024 * 1024, want: "1.0 GB"},
		{name: "um TB", size: 1 << 40, want: "1.0 TB"},
		{name: "um PB", size: 1 << 50, want: "1.0 PB"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DescribeBytes(test.size); got != test.want {
				t.Errorf("tamanho = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDescribeResidenceSeparatesWhatLeftTheTree(t *testing.T) {
	if DescribeResidence(true) == DescribeResidence(false) {
		t.Fatal("estar na arvore e so estar no historico e a distincao que o comando existe para mostrar")
	}
	if got := DescribeResidence(false); got != "só no histórico" {
		t.Errorf("rotulo = %q", got)
	}
}
