package branch

import "testing"

func TestMergeKindToken(t *testing.T) {
	tests := []struct {
		kind MergeKind
		want string
	}{
		{kind: MergedByAncestry, want: "ancestry"},
		{kind: MergedBySquash, want: "squash"},
		{kind: MergedByRebase, want: "rebase"},
		{kind: MergeKind(99), want: "ancestry"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			if got := test.kind.String(); got != test.want {
				t.Errorf("token = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestBaseSourceToken(t *testing.T) {
	tests := []struct {
		source BaseSource
		want   string
	}{
		{source: BaseFromFlag, want: "flag"},
		{source: BaseFromOriginHead, want: "origin-head"},
		{source: BaseFromLocal, want: "local"},
		{source: BaseSource(99), want: "local"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			if got := test.source.String(); got != test.want {
				t.Errorf("token = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestTokensStayInEnglishWhileTheLabelsCanBeTranslated(t *testing.T) {
	for _, kind := range []MergeKind{MergedByAncestry, MergedBySquash, MergedByRebase} {
		for _, portuguese := range []string{"mergeada", "squashada", "rebaseada"} {
			if kind.String() == portuguese {
				t.Errorf("token = %q; token é contrato de máquina e não pode ser o rótulo de tela, que o roadmap 9 vai traduzir", kind.String())
			}
		}
	}
}
