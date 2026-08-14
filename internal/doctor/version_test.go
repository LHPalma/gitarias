package doctor

import "testing"

func TestVersionOfTheTwoFormats(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "git", output: "git version 2.43.0", want: "2.43.0"},
		{name: "gh com data e url", output: "gh version 2.62.0 (2024-11-14)\nhttps://github.com/cli/cli", want: "2.62.0"},
		{name: "sem a palavra version cai no ultimo campo", output: "2.43.0", want: "2.43.0"},
		{name: "version no fim nao estoura o slice", output: "alguma coisa version", want: "version"},
		{name: "vazio", output: "  \n ", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := version(test.output); got != test.want {
				t.Errorf("versao = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestParseReleaseReadsWhatTheDistributionsAppend(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		want     release
		readable bool
	}{
		{name: "tres numeros", text: "2.43.0", want: release{major: 2, minor: 43}, readable: true},
		{name: "dois numeros", text: "2.43", want: release{major: 2, minor: 43}, readable: true},
		{name: "sufixo da apple", text: "2.39.5", want: release{major: 2, minor: 39}, readable: true},
		{name: "sufixo do windows", text: "2.44.0.windows.1", want: release{major: 2, minor: 44}, readable: true},
		{name: "so o maior", text: "2"},
		{name: "menor nao numerico", text: "2.x.1"},
		{name: "maior nao numerico", text: "dois.43"},
		{name: "vazio", text: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, readable := parseRelease(test.text)

			if readable != test.readable {
				t.Fatalf("legivel = %v, queria %v", readable, test.readable)
			}
			if got != test.want {
				t.Errorf("versao = %+v, queria %+v", got, test.want)
			}
		})
	}
}

func TestReleaseComparesTheMinorOnlyInsideTheSameMajor(t *testing.T) {
	tests := []struct {
		name    string
		version release
		other   release
		want    bool
	}{
		{name: "menor no mesmo maior", version: release{2, 20}, other: release{2, 22}, want: true},
		{name: "igual", version: release{2, 22}, other: release{2, 22}},
		{name: "maior no mesmo maior", version: release{2, 43}, other: release{2, 22}},
		{name: "maior menor vence o minor maior", version: release{1, 99}, other: release{2, 22}, want: true},
		{name: "maior maior vence o minor menor", version: release{3, 0}, other: release{2, 22}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.version.before(test.other); got != test.want {
				t.Errorf("%v antes de %v = %v, queria %v", test.version, test.other, got, test.want)
			}
		})
	}
}

func TestMinimumGitIsPrintedWithoutAPatch(t *testing.T) {
	if got := minimumGit.String(); got != "2.22" {
		t.Errorf("minima = %q; o patch nao entra no criterio e nao pode aparecer como se entrasse", got)
	}
}
