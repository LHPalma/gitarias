package ui

import (
	"testing"

	"github.com/LHPalma/gitarias/internal/branch"
)

func TestDescribeMerge(t *testing.T) {
	tests := []struct {
		name string
		kind branch.MergeKind
		want string
	}{
		{name: "ancestralidade", kind: branch.MergedByAncestry, want: "mergeada"},
		{name: "squash", kind: branch.MergedBySquash, want: "squashada"},
		{name: "rebase", kind: branch.MergedByRebase, want: "rebaseada"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DescribeMerge(test.kind); got != test.want {
				t.Errorf("rótulo = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDescribeSource(t *testing.T) {
	tests := []struct {
		name   string
		source branch.BaseSource
		want   string
	}{
		{name: "via flag", source: branch.BaseFromFlag, want: "informada via --base"},
		{name: "via origin HEAD", source: branch.BaseFromOriginHead, want: "detectada via origin/HEAD"},
		{name: "encontrada no repositorio", source: branch.BaseFromLocal, want: "encontrada localmente"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DescribeSource(test.source); got != test.want {
				t.Errorf("texto = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDescribeLayer(t *testing.T) {
	tests := []struct {
		name  string
		layer branch.Layer
		want  string
	}{
		{
			name:  "nao mergeada ignora o tipo de merge",
			layer: branch.Layer{Branch: branch.Branch{Merge: branch.MergedBySquash}},
			want:  "não mergeada",
		},
		{
			name:  "mergeada por ancestralidade",
			layer: branch.Layer{Merged: true, Branch: branch.Branch{Merge: branch.MergedByAncestry}},
			want:  "mergeada",
		},
		{
			name:  "squashada",
			layer: branch.Layer{Merged: true, Branch: branch.Branch{Merge: branch.MergedBySquash}},
			want:  "squashada",
		},
		{
			name:  "rebaseada",
			layer: branch.Layer{Merged: true, Branch: branch.Branch{Merge: branch.MergedByRebase}},
			want:  "rebaseada",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DescribeLayer(test.layer); got != test.want {
				t.Errorf("rotulo = %q, queria %q", got, test.want)
			}
		})
	}
}
