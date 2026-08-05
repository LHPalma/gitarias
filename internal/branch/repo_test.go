package branch

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const insideWorkTree = "rev-parse --is-inside-work-tree"

func exists(name string) string {
	return "rev-parse --verify --quiet refs/heads/" + name
}

func TestEnsure(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]gittest.Response
		wantError bool
	}{
		{
			name:      "dentro de um repositorio",
			responses: map[string]gittest.Response{insideWorkTree: {Output: "true"}},
		},
		{
			name:      "fora de um repositorio",
			responses: map[string]gittest.Response{insideWorkTree: {Err: errors.New("fatal: not a git repository")}},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewRepo(gittest.NewRunner(test.responses)).Ensure()

			if test.wantError && err == nil {
				t.Fatal("esperava erro, veio nil")
			}
			if !test.wantError && err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}
			if test.wantError && !strings.Contains(err.Error(), "não é um repositório git") {
				t.Fatalf("mensagem inesperada: %v", err)
			}
		})
	}
}

func TestResolveBase(t *testing.T) {
	tests := []struct {
		name       string
		requested  string
		responses  map[string]gittest.Response
		wantName   string
		wantSource BaseSource
		wantError  string
	}{
		{
			name:       "informada via flag",
			requested:  "develop",
			responses:  map[string]gittest.Response{exists("develop"): {Output: "abc123"}},
			wantName:   "develop",
			wantSource: BaseFromFlag,
		},
		{
			name:      "informada via flag mas inexistente",
			requested: "fantasma",
			responses: map[string]gittest.Response{exists("fantasma"): {Err: errors.New("")}},
			wantError: `a branch base "fantasma" não existe`,
		},
		{
			name: "detectada via origin/HEAD",
			responses: map[string]gittest.Response{
				"symbolic-ref --short refs/remotes/origin/HEAD": {Output: "origin/main"},
				exists("main"): {Output: "abc123"},
			},
			wantName:   "main",
			wantSource: BaseFromOriginHead,
		},
		{
			name: "origin/HEAD aponta para branch sem copia local, cai para main",
			responses: map[string]gittest.Response{
				"symbolic-ref --short refs/remotes/origin/HEAD": {Output: "origin/trunk"},
				exists("trunk"):  {Err: errors.New("")},
				exists("main"):   {Output: "abc123"},
				exists("master"): {Err: errors.New("")},
			},
			wantName:   "main",
			wantSource: BaseFromLocal,
		},
		{
			name: "sem origin/HEAD, encontra main",
			responses: map[string]gittest.Response{
				"symbolic-ref --short refs/remotes/origin/HEAD": {Err: errors.New("")},
				exists("main"): {Output: "abc123"},
			},
			wantName:   "main",
			wantSource: BaseFromLocal,
		},
		{
			name: "sem origin/HEAD e sem main, encontra master",
			responses: map[string]gittest.Response{
				"symbolic-ref --short refs/remotes/origin/HEAD": {Err: errors.New("")},
				exists("main"):   {Err: errors.New("")},
				exists("master"): {Output: "abc123"},
			},
			wantName:   "master",
			wantSource: BaseFromLocal,
		},
		{
			name: "nada determinavel",
			responses: map[string]gittest.Response{
				"symbolic-ref --short refs/remotes/origin/HEAD": {Err: errors.New("")},
				exists("main"):   {Err: errors.New("")},
				exists("master"): {Err: errors.New("")},
			},
			wantError: "não consegui determinar a branch base",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base, err := NewRepo(gittest.NewRunner(test.responses)).ResolveBase(test.requested)

			if test.wantError != "" {
				if err == nil {
					t.Fatalf("esperava erro contendo %q, veio nil", test.wantError)
				}
				if !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("erro %q não contém %q", err, test.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}
			if base.Name != test.wantName {
				t.Errorf("nome da base = %q, queria %q", base.Name, test.wantName)
			}
			if base.Source != test.wantSource {
				t.Errorf("origem da base = %d, queria %d", base.Source, test.wantSource)
			}
		})
	}
}

func TestResolveBaseSkipsDetectionWhenFlagSet(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{exists("develop"): {Output: "abc123"}})

	if _, err := NewRepo(runner).ResolveBase("develop"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, call := range runner.Calls {
		if strings.Contains(call, "symbolic-ref") {
			t.Fatalf("a flag deveria curto-circuitar a detecção, mas rodou %q", call)
		}
	}
}

func mergedResponses(base string, refs string, current string) map[string]gittest.Response {
	return listings(base, refs, refs, current)
}

func listings(base string, ancestors string, all string, current string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"for-each-ref refs/heads/ --merged " + base + " --format=%(refname:short)": {Output: ancestors},
		"for-each-ref refs/heads/ --format=%(refname:short)":                       {Output: all},
		"branch --show-current": {Output: current},
	}
}

func probe(base string, name string, cherry gittest.Response) map[string]gittest.Response {
	mergeBase, tree, virtual := "mb-"+name, "tree-"+name, "probe-"+name

	return map[string]gittest.Response{
		"merge-base " + base + " " + name:                                             {Output: mergeBase},
		"rev-parse " + name + "^{tree}":                                               {Output: tree},
		"commit-tree " + tree + " -p " + mergeBase + " -m " + equivalenceProbeMessage: {Output: virtual},
		"cherry " + base + " " + virtual:                                              cherry,
	}
}

func combine(sources ...map[string]gittest.Response) map[string]gittest.Response {
	combined := map[string]gittest.Response{}

	for _, source := range sources {
		for command, response := range source {
			combined[command] = response
		}
	}

	return combined
}

func TestMerged(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		refs    string
		current string
		want    []string
	}{
		{
			name:    "lista as branches nao protegidas",
			base:    "main",
			refs:    "main\nfeat-a\nfeat-b",
			current: "feat-a",
			want:    []string{"feat-b"},
		},
		{
			name:    "a base sai da lista",
			base:    "develop",
			refs:    "develop\nfeat-a",
			current: "outra",
			want:    []string{"feat-a"},
		},
		{
			name:    "main e master sao protegidas mesmo sem serem a base",
			base:    "develop",
			refs:    "develop\nmain\nmaster\nfeat-a",
			current: "outra",
			want:    []string{"feat-a"},
		},
		{
			name:    "a branch atual nunca aparece",
			base:    "main",
			refs:    "main\nfeat-a",
			current: "feat-a",
			want:    nil,
		},
		{
			name:    "saida vazia devolve nada",
			base:    "main",
			refs:    "",
			current: "main",
			want:    nil,
		},
		{
			name:    "linhas em branco sao ignoradas",
			base:    "main",
			refs:    "main\n\n  feat-a  \n\n",
			current: "main",
			want:    []string{"feat-a"},
		},
		{
			name:    "HEAD destacado deixa a branch atual vazia",
			base:    "main",
			refs:    "main\nfeat-a",
			current: "",
			want:    []string{"feat-a"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := NewRepo(gittest.NewRunner(mergedResponses(test.base, test.refs, test.current)))

			merged, err := repo.Merged(Base{Name: test.base})
			if err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}

			if len(merged) != len(test.want) {
				t.Fatalf("veio %v, queria %v", names(merged), test.want)
			}
			for index, wanted := range test.want {
				if merged[index].Name != wanted {
					t.Fatalf("veio %v, queria %v", names(merged), test.want)
				}
			}
		})
	}
}

func TestMergedPropagatesGitError(t *testing.T) {
	responses := map[string]gittest.Response{
		"for-each-ref refs/heads/ --merged main --format=%(refname:short)": {Err: errors.New("malformed object name main")},
	}

	_, err := NewRepo(gittest.NewRunner(responses)).Merged(Base{Name: "main"})
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "malformed object name") {
		t.Fatalf("esperava o erro do git, veio %v", err)
	}
}

func TestMergedFindsEquivalentBranches(t *testing.T) {
	tests := []struct {
		name      string
		virtual   gittest.Response
		commits   gittest.Response
		wantFound bool
		wantKind  MergeKind
	}{
		{
			name:      "squashada: o commit virtual tem equivalente na base",
			virtual:   gittest.Response{Output: "- probe-solta"},
			wantFound: true,
			wantKind:  MergedBySquash,
		},
		{
			name:      "rebaseada: todos os commits tem equivalente na base",
			virtual:   gittest.Response{Output: "+ probe-solta"},
			commits:   gittest.Response{Output: "- aaaaaaa\n- bbbbbbb"},
			wantFound: true,
			wantKind:  MergedByRebase,
		},
		{
			name:    "trabalho de verdade nao entra",
			virtual: gittest.Response{Output: "+ probe-solta"},
			commits: gittest.Response{Output: "+ aaaaaaa\n+ bbbbbbb"},
		},
		{
			name:    "integrada pela metade nao entra",
			virtual: gittest.Response{Output: "+ probe-solta"},
			commits: gittest.Response{Output: "- aaaaaaa\n+ bbbbbbb"},
		},
		{
			name:    "cherry sem nenhum commit nao conta como equivalente",
			virtual: gittest.Response{Output: "+ probe-solta"},
			commits: gittest.Response{Output: ""},
		},
		{
			name:    "cherry do commit virtual falhando nao derruba a deteccao",
			virtual: gittest.Response{Err: errors.New("fatal: bad revision")},
			commits: gittest.Response{Output: "+ aaaaaaa"},
		},
		{
			name:    "commit virtual sozinho a frente da base: mais de uma linha nao e resposta valida",
			virtual: gittest.Response{Output: "- probe-solta\n- aaaaaaa"},
			commits: gittest.Response{Output: "+ aaaaaaa"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responses := combine(
				listings("main", "main", "main\nsolta", "main"),
				probe("main", "solta", test.virtual),
				map[string]gittest.Response{"cherry main solta": test.commits},
			)

			merged, err := NewRepo(gittest.NewRunner(responses)).Merged(Base{Name: "main"})
			if err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}

			if !test.wantFound {
				if len(merged) != 0 {
					t.Fatalf("não deveria listar nada, veio %v", names(merged))
				}
				return
			}

			if len(merged) != 1 || merged[0].Name != "solta" {
				t.Fatalf("esperava só a solta, veio %v", names(merged))
			}
			if merged[0].Merge != test.wantKind {
				t.Errorf("tipo de merge = %d, queria %d", merged[0].Merge, test.wantKind)
			}
		})
	}
}

func TestMergedDegradesWhenTheProbeCannotRun(t *testing.T) {
	tests := []struct {
		name     string
		breaking map[string]gittest.Response
	}{
		{
			name:     "rev-parse da arvore falha",
			breaking: map[string]gittest.Response{"rev-parse solta^{tree}": {Err: errors.New("fatal: bad revision")}},
		},
		{
			name:     "rev-parse da arvore devolve vazio",
			breaking: map[string]gittest.Response{"rev-parse solta^{tree}": {Output: ""}},
		},
		{
			name: "commit-tree falha por falta de identidade",
			breaking: map[string]gittest.Response{
				"commit-tree tree-solta -p mb-solta -m " + equivalenceProbeMessage: {Err: errors.New("Author identity unknown")},
			},
		},
		{
			name: "commit-tree devolve vazio",
			breaking: map[string]gittest.Response{
				"commit-tree tree-solta -p mb-solta -m " + equivalenceProbeMessage: {Output: ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := gittest.NewRunner(combine(
				listings("main", "main", "main\nsolta", "main"),
				probe("main", "solta", gittest.Response{Output: "- probe-solta"}),
				map[string]gittest.Response{"cherry main solta": {Output: "+ aaaaaaa"}},
				test.breaking,
			))

			merged, err := NewRepo(runner).Merged(Base{Name: "main"})
			if err != nil {
				t.Fatalf("sonda quebrada não pode derrubar o comando, veio %v", err)
			}
			if len(merged) != 0 {
				t.Fatalf("sem prova de equivalência nada pode ser listado, veio %v", names(merged))
			}
		})
	}
}

func TestMergedPropagatesListingError(t *testing.T) {
	responses := listings("main", "main", "", "main")
	responses["for-each-ref refs/heads/ --format=%(refname:short)"] = gittest.Response{Err: errors.New("malformed object name")}

	_, err := NewRepo(gittest.NewRunner(responses)).Merged(Base{Name: "main"})
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "malformed object name") {
		t.Fatalf("esperava o erro do git, veio %v", err)
	}
}

func TestMergedProbeFailureIsNotFatal(t *testing.T) {
	responses := combine(
		listings("main", "main", "main\nsolta", "main"),
		map[string]gittest.Response{"merge-base main solta": {Err: errors.New("fatal: no merge base")}},
	)

	merged, err := NewRepo(gittest.NewRunner(responses)).Merged(Base{Name: "main"})
	if err != nil {
		t.Fatalf("branch sem merge-base não pode derrubar o comando, veio %v", err)
	}
	if len(merged) != 0 {
		t.Fatalf("não deveria listar nada, veio %v", names(merged))
	}
}

func TestMergedDoesNotProbeProtectedBranches(t *testing.T) {
	runner := gittest.NewRunner(listings("develop", "develop", "develop\nmain\nmaster\natual", "atual"))

	merged, err := NewRepo(runner).Merged(Base{Name: "develop"})
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(merged) != 0 {
		t.Fatalf("nenhuma protegida podia ser listada, veio %v", names(merged))
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "merge-base") || strings.HasPrefix(call, "commit-tree") {
			t.Fatalf("branch protegida não pode ser sondada, mas rodou %q", call)
		}
	}
}

func TestMergedNeverReadsRemoteRefs(t *testing.T) {
	runner := gittest.NewRunner(combine(
		listings("main", "main", "main\nsolta", "main"),
		probe("main", "solta", gittest.Response{Output: "- probe-solta"}),
	))

	if _, err := NewRepo(runner).Merged(Base{Name: "main"}); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, call := range runner.Calls {
		if strings.Contains(call, "refs/remotes") || strings.Contains(call, "origin/") {
			t.Fatalf("RN-04: a consulta não pode enxergar ref remota, mas rodou %q", call)
		}
	}
}

func names(branches []Branch) []string {
	var out []string
	for _, b := range branches {
		out = append(out, b.Name)
	}
	return out
}

func TestDelete(t *testing.T) {
	falha := errors.New("cannot delete branch 'presa' used by worktree")
	responses := map[string]gittest.Response{
		"branch -d livre-a": {Output: "Deleted branch livre-a"},
		"branch -d presa":   {Err: falha},
		"branch -d livre-b": {Output: "Deleted branch livre-b"},
	}
	runner := gittest.NewRunner(responses)

	results := NewRepo(runner).Delete([]Branch{{Name: "livre-a"}, {Name: "presa"}, {Name: "livre-b"}})

	if len(results) != 3 {
		t.Fatalf("esperava 3 resultados, veio %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("livre-a deveria ter passado, veio %v", results[0].Err)
	}
	if results[1].Err == nil {
		t.Error("presa deveria ter falhado")
	}
	if results[2].Err != nil {
		t.Errorf("uma falha no meio não pode abortar as seguintes, mas livre-b veio %v", results[2].Err)
	}
	for index, wanted := range []string{"livre-a", "presa", "livre-b"} {
		if results[index].Branch.Name != wanted {
			t.Errorf("resultado %d é de %q, queria %q", index, results[index].Branch.Name, wanted)
		}
	}
	if len(runner.Calls) != 3 {
		t.Errorf("esperava 3 chamadas ao git, veio %d: %v", len(runner.Calls), runner.Calls)
	}
}

func TestDeleteWithEmptyListSkipsGit(t *testing.T) {
	runner := gittest.NewRunner(nil)

	results := NewRepo(runner).Delete(nil)

	if len(results) != 0 {
		t.Errorf("esperava nenhum resultado, veio %d", len(results))
	}
	if len(runner.Calls) != 0 {
		t.Errorf("não deveria ter chamado o git, chamou %v", runner.Calls)
	}
}

func TestDeleteNeverForces(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{"branch -d qualquer": {Output: ""}})

	NewRepo(runner).Delete([]Branch{{Name: "qualquer"}})

	for _, call := range runner.Calls {
		if strings.Contains(call, "-D") {
			t.Fatalf("RN-01: deleção nunca pode usar -D, mas rodou %q", call)
		}
	}
}
