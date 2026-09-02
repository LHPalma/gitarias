package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
	"github.com/LHPalma/gitarias/internal/platform"
	"github.com/LHPalma/gitarias/internal/platform/platformtest"
	"github.com/LHPalma/gitarias/internal/web"
	"github.com/LHPalma/gitarias/internal/web/webtest"
)

var errNotARepository = errors.New("fatal: not a git repository")

const noNotices = "resumo\n" + noticesSeparator + "texto completo\n"

func noCommands() *exectest.Runner {
	return exectest.NewRunner()
}

func noWeb() web.Client {
	return webtest.NewClient()
}

func deletesABranch(call string) bool {
	return strings.HasPrefix(call, "branch -d ") || strings.HasPrefix(call, "branch -D ")
}

type execution struct {
	stdout string
	stderr string
	err    error
	calls  []string
}

func execute(t *testing.T, responses map[string]gittest.Response, answer string, args ...string) execution {
	t.Helper()

	runner := gittest.NewRunner(responses)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetIn(strings.NewReader(answer))
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}

func repository(base string, ancestors string, all string, current string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":                                          {Output: "true"},
		"symbolic-ref --short refs/remotes/origin/HEAD":                            {Output: "origin/" + base},
		"rev-parse --verify --quiet refs/heads/" + base:                            {Output: "abc123"},
		"for-each-ref refs/heads/ --merged " + base + " --format=%(refname:short)": {Output: ancestors},
		"for-each-ref refs/heads/ --format=%(refname:short)":                       {Output: all},
		"branch --show-current":                                                    {Output: current},
		"worktree list --porcelain":                                                {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/" + current + "\n"},
		"rev-parse --show-toplevel":                                                {Output: "/repo"},
	}
}

func TestBranchesCommandLists(t *testing.T) {
	result := execute(t, repository("main", "main\nfeat-a", "main\nfeat-a", "main"), "", "branches")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "Base: main (detectada via origin/HEAD)\n\n" +
		"Branches locais já mergeadas (1):\n" +
		"  feat-a  mergeada\n\n" +
		"Use --clean para deletar.\n"

	if result.stdout != want {
		t.Errorf("saída:\n%q\nqueria:\n%q", result.stdout, want)
	}
	if result.stderr != "" {
		t.Errorf("RN-09: stderr deveria estar vazio, veio %q", result.stderr)
	}
}

func TestBranchesCommandWithNothingToClean(t *testing.T) {
	result := execute(t, repository("main", "main", "main", "main"), "", "branches")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhuma branch local mergeada para limpar.") {
		t.Errorf("saída = %q, queria a mensagem específica", result.stdout)
	}
	for _, call := range result.calls {
		if deletesABranch(call) {
			t.Fatalf("sem candidata nada pode ser deletado, mas rodou %q", call)
		}
	}
}

func TestBranchesCommandWithoutCleanNeverDeletes(t *testing.T) {
	result := execute(t, repository("main", "main\nfeat-a", "main\nfeat-a", "main"), "y\n", "branches")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	for _, call := range result.calls {
		if deletesABranch(call) {
			t.Fatalf("sem --clean nada pode ser deletado, mas rodou %q", call)
		}
	}
	if strings.Contains(result.stdout, "[y/N]") {
		t.Error("sem --clean não pode nem perguntar")
	}
}

func TestBranchesCommandCleanDeletes(t *testing.T) {
	responses := repository("main", "main\nfeat-a", "main\nfeat-a", "main")
	responses["branch -d feat-a"] = gittest.Response{Output: "Deleted branch feat-a"}

	result := execute(t, responses, "y\n", "branches", "--clean")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "feat-a deletada") {
		t.Errorf("saída = %q, queria confirmar a deleção", result.stdout)
	}
}

func TestBranchesCommandCleanCancelled(t *testing.T) {
	result := execute(t, repository("main", "main\nfeat-a", "main\nfeat-a", "main"), "\n", "branches", "--clean")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Cancelado, nada foi deletado.") {
		t.Errorf("saída = %q, queria o cancelamento", result.stdout)
	}
	for _, call := range result.calls {
		if deletesABranch(call) {
			t.Fatalf("RF-04: Enter vazio cancela, mas rodou %q", call)
		}
	}
}

func squashedRepository() map[string]gittest.Response {
	responses := repository("main", "main", "main\nsquashada", "main")
	responses["merge-base main squashada"] = gittest.Response{Output: "mb"}
	responses["rev-parse squashada^{tree}"] = gittest.Response{Output: "tree"}
	responses["commit-tree tree -p mb -m gtr equivalence probe"] = gittest.Response{Output: "probe"}
	responses["cherry main probe"] = gittest.Response{Output: "- probe"}

	return responses
}

func TestBranchesCommandHoldsEquivalentsWithoutForce(t *testing.T) {
	result := execute(t, squashedRepository(), "y\n", "branches", "--clean")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "squashada") {
		t.Errorf("a branch tem de aparecer na listagem, veio %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "Use --force para forçar.") {
		t.Errorf("saída = %q, queria o aviso de que ficaram de fora", result.stdout)
	}
	for _, call := range result.calls {
		if deletesABranch(call) {
			t.Fatalf("RN-01: sem --force nada pode ser apagado, mas rodou %q", call)
		}
	}
}

func TestBranchesCommandForceDeletesEquivalents(t *testing.T) {
	responses := squashedRepository()
	responses["branch -D squashada"] = gittest.Response{Output: ""}

	result := execute(t, responses, "y\n", "branches", "--clean", "--force")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	var forced bool
	for _, call := range result.calls {
		if call == "branch -D squashada" {
			forced = true
		}
		if call == "branch -d squashada" {
			t.Error("o git recusaria -d numa squashada; o comando tem de usar -D")
		}
	}
	if !forced {
		t.Errorf("esperava a deleção forçada entre %v", result.calls)
	}
}

func TestBranchesCommandOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "branches")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestWorktreesCommandLists(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain":       {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n"},
		"rev-parse --show-toplevel":       {Output: "/repo"},
	}

	result := execute(t, responses, "", "worktrees")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}

	want := "Working trees (1):\n* /repo  main\n"
	if result.stdout != want {
		t.Errorf("saída = %q, queria %q", result.stdout, want)
	}
}

func TestCommandsNeverShareFlagState(t *testing.T) {
	responses := squashedRepository()
	responses["branch -D squashada"] = gittest.Response{Output: ""}

	execute(t, responses, "y\n", "branches", "--clean", "--force")
	segundo := execute(t, squashedRepository(), "y\n", "branches", "--clean")

	for _, call := range segundo.calls {
		if deletesABranch(call) {
			t.Fatalf("o --force do teste anterior vazou: rodou %q sem a flag", call)
		}
	}
}

func TestBranchesCommandPropagatesResolveBaseFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":               {Output: "true"},
		"symbolic-ref --short refs/remotes/origin/HEAD": {Err: errNotARepository},
		"rev-parse --verify --quiet refs/heads/main":    {Err: errNotARepository},
		"rev-parse --verify --quiet refs/heads/master":  {Err: errNotARepository},
	}

	result := execute(t, responses, "", "branches")

	if result.err == nil {
		t.Fatal("base indeterminável tem de virar erro")
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada no stdout, veio %q", result.stdout)
	}
}

func TestBranchesCommandPropagatesListingFailure(t *testing.T) {
	responses := repository("main", "", "", "main")
	responses["for-each-ref refs/heads/ --merged main --format=%(refname:short)"] = gittest.Response{Err: errNotARepository}

	if execute(t, responses, "", "branches").err == nil {
		t.Fatal("falha do git na listagem tem de virar erro")
	}
}

func TestBranchesCommandPropagatesWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(repository("main", "main\nfeat-a", "main\nfeat-a", "main")), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"branches"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestBranchesCommandPropagatesReadFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(repository("main", "main\nfeat-a", "main\nfeat-a", "main")), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"branches", "--clean"})

	if command.Execute() == nil {
		t.Fatal("falha na leitura da confirmação tem de virar erro")
	}
}

func TestWorktreesCommandOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Err: errNotARepository}}

	err := execute(t, responses, "", "worktrees").err
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", err)
	}
}

func TestWorktreesCommandPropagatesListFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain":       {Err: errNotARepository},
	}

	if execute(t, responses, "", "worktrees").err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestWorktreesRemoveListsIgnoredFilesAndAsks(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Output: ".env\x00node_modules/\x00",
		},
		"worktree remove " + fix: {Output: ""},
	}

	result := execute(t, responses, "y\n", "worktrees", "remove", fix)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "2 arquivos ignorados serão perdidos, sem chance de recuperação") {
		t.Errorf("saída = %q, queria o aviso dos ignorados", result.stdout)
	}
	if !strings.Contains(result.stdout, ".env") || !strings.Contains(result.stdout, "node_modules/") {
		t.Errorf("saída = %q, queria os dois arquivos listados", result.stdout)
	}
	if !strings.Contains(result.stdout, "Remover "+fix+"? [y/N] ") {
		t.Errorf("saída = %q, queria a pergunta de confirmação", result.stdout)
	}
	if !strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, queria a confirmação de sucesso", result.stdout)
	}

	var removed bool
	for _, call := range result.calls {
		if call == "worktree remove "+fix {
			removed = true
		}
	}
	if !removed {
		t.Errorf("esperava a chamada de remoção entre %v", result.calls)
	}
}

func TestWorktreesRemoveSingularMessage(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Output: ".env\x00",
		},
	}

	result := execute(t, responses, "n\n", "worktrees", "remove", fix)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "1 arquivo ignorado será perdido, sem chance de recuperação") {
		t.Errorf("saída = %q, queria a mensagem no singular", result.stdout)
	}
}

func TestWorktreesRemoveWithoutIgnoredFiles(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {Output: ""},
		"worktree remove " + fix: {Output: ""},
	}

	result := execute(t, responses, "y\n", "worktrees", "remove", fix)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, fix+" não tem arquivo ignorado a perder.") {
		t.Errorf("saída = %q, queria a mensagem de nada a perder", result.stdout)
	}
}

func TestWorktreesRemoveCancelledNeverCallsGit(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {Output: ".env\x00"},
	}

	result := execute(t, responses, "\n", "worktrees", "remove", fix)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Cancelado, nada foi removido.") {
		t.Errorf("saída = %q, queria o cancelamento", result.stdout)
	}
	for _, call := range result.calls {
		if strings.HasPrefix(call, "worktree remove ") {
			t.Fatalf("RF-04: Enter vazio cancela, mas rodou %q", call)
		}
	}
}

func TestWorktreesRemoveRejectsPathThatIsNotAWorktree(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain":       {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n"},
		"rev-parse --show-toplevel":       {Output: "/repo"},
	}

	result := execute(t, responses, "", "worktrees", "remove", "/nao-existe")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um working tree deste repositório") {
		t.Errorf("erro = %v, queria a mensagem específica", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestWorktreesRemovePropagatesGitRefusal(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {Output: ""},
		"worktree remove " + fix: {Err: errors.New("fatal: contains modified or untracked files, use --force to delete it")},
	}

	result := execute(t, responses, "y\n", "worktrees", "remove", fix)

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "use --force to delete it") {
		t.Errorf("erro deveria carregar a mensagem do git, veio %v", result.err)
	}
}

func TestWorktreesRemovePropagatesIgnoredFilesFailure(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Err: errNotARepository,
		},
	}

	result := execute(t, responses, "", "worktrees", "remove", fix)

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	for _, call := range result.calls {
		if strings.HasPrefix(call, "worktree remove ") {
			t.Fatalf("não pode remover sem antes conseguir listar os ignorados, mas rodou %q", call)
		}
	}
}

func TestWorktreesRemovePropagatesListFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain":       {Err: errNotARepository},
	}

	result := execute(t, responses, "", "worktrees", "remove", "/repo-fix")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestWorktreesRemovePropagatesWriteFailure(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Output: ".env\x00",
		},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"worktrees", "remove", fix})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestWorktreesRemovePropagatesReadFailure(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
		"-C " + fix + " ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z": {
			Output: "",
		},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"worktrees", "remove", fix})

	if command.Execute() == nil {
		t.Fatal("falha na leitura da confirmação tem de virar erro")
	}
}

func TestWorktreesRemoveOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Err: errNotARepository}}

	result := execute(t, responses, "", "worktrees", "remove", "/qualquer")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
}

func TestWorktreesReleaseWarnsAndAsks(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Output: " M arquivo.go\n"},
		"-C " + fix + " checkout --detach":  {Output: ""},
	}

	result := execute(t, responses, "y\n", "worktrees", "release", "fix")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, fix) {
		t.Errorf("saída = %q, queria o caminho do working tree", result.stdout)
	}
	if !strings.Contains(result.stdout, "tem trabalho não commitado") {
		t.Errorf("saída = %q, queria o aviso de trabalho não commitado", result.stdout)
	}
	if !strings.Contains(result.stdout, "Soltar fix? [y/N] ") {
		t.Errorf("saída = %q, queria a pergunta de confirmação", result.stdout)
	}
	if !strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, queria a confirmação de sucesso", result.stdout)
	}

	var released bool
	for _, call := range result.calls {
		if call == "-C "+fix+" checkout --detach" {
			released = true
		}
	}
	if !released {
		t.Errorf("esperava a chamada de detach entre %v", result.calls)
	}
}

func TestWorktreesReleaseWithoutUncommittedWork(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Output: ""},
		"-C " + fix + " checkout --detach":  {Output: ""},
	}

	result := execute(t, responses, "y\n", "worktrees", "release", "fix")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "trabalho não commitado") {
		t.Errorf("saída = %q, não devia avisar sobre trabalho não commitado que não existe", result.stdout)
	}
}

func TestWorktreesReleasePropagatesWriteFailureOfTheDirtyWarning(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Output: " M arquivo.go\n"},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&countingWriter{allowed: 1})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"worktrees", "release", "fix"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita no aviso de trabalho não commitado tem de virar erro")
	}
}

func TestWorktreesReleaseCancelledNeverCallsGit(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Output: ""},
	}

	result := execute(t, responses, "\n", "worktrees", "release", "fix")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Cancelado, nada foi solto.") {
		t.Errorf("saída = %q, queria o cancelamento", result.stdout)
	}
	for _, call := range result.calls {
		if strings.Contains(call, "checkout --detach") {
			t.Fatalf("RF-04: Enter vazio cancela, mas rodou %q", call)
		}
	}
}

func TestWorktreesReleaseRejectsBranchNotHeldByAnyWorktree(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain":       {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n"},
		"rev-parse --show-toplevel":       {Output: "/repo"},
	}

	result := execute(t, responses, "", "worktrees", "release", "fantasma")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não está presa em nenhum working tree") {
		t.Errorf("erro = %v, queria a mensagem específica", result.err)
	}
	if result.stdout != "" {
		t.Errorf("RN-09: nada pode ir para o stdout, veio %q", result.stdout)
	}
}

func TestWorktreesReleasePropagatesGitRefusal(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Output: ""},
		"-C " + fix + " checkout --detach":  {Err: errors.New("error: you need to resolve your current index first")},
	}

	result := execute(t, responses, "y\n", "worktrees", "release", "fix")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "resolve your current index first") {
		t.Errorf("erro deveria carregar a mensagem do git, veio %v", result.err)
	}
}

func TestWorktreesReleasePropagatesDirtyCheckFailure(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "worktrees", "release", "fix")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	for _, call := range result.calls {
		if strings.Contains(call, "checkout --detach") {
			t.Fatalf("não pode soltar sem antes conseguir checar trabalho não commitado, mas rodou %q", call)
		}
	}
}

func TestWorktreesReleasePropagatesListFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain":       {Err: errNotARepository},
	}

	result := execute(t, responses, "", "worktrees", "release", "fix")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestWorktreesReleasePropagatesWriteFailure(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Output: ""},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"worktrees", "release", "fix"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita tem de virar erro")
	}
}

func TestWorktreesReleasePropagatesReadFailure(t *testing.T) {
	fix := absolute(t, "/repo-fix")
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + fix + "\nHEAD def\nbranch refs/heads/fix\n"},
		"rev-parse --show-toplevel":         {Output: "/repo"},
		"-C " + fix + " status --porcelain": {Output: ""},
	}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"worktrees", "release", "fix"})

	if command.Execute() == nil {
		t.Fatal("falha na leitura da confirmação tem de virar erro")
	}
}

func TestWorktreesReleaseOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Err: errNotARepository}}

	result := execute(t, responses, "", "worktrees", "release", "fix")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
}

func TestWorktreesCommandShowsTheStateColumn(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		"worktree list --porcelain": {Output: "worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree /outro\nHEAD def\nbranch refs/heads/velha\nlocked hd externo\n"},
		"rev-parse --show-toplevel": {Output: "/repo"},
	}

	result := execute(t, responses, "", "worktrees")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "trancado: hd externo") {
		t.Errorf("saída = %q, queria o estado do worktree", result.stdout)
	}
	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func TestBranchesCommandUsesTheRequestedBase(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":                                     {Output: "true"},
		"rev-parse --verify --quiet refs/heads/develop":                       {Output: "abc123"},
		"for-each-ref refs/heads/ --merged develop --format=%(refname:short)": {Output: "develop\nfeat-a"},
		"for-each-ref refs/heads/ --format=%(refname:short)":                  {Output: "develop\nfeat-a"},
		"branch --show-current":                                               {Output: "develop"},
	}

	result := execute(t, responses, "", "branches", "--base", "develop")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Base: develop (informada via --base)") {
		t.Errorf("saída = %q, queria a base da flag", result.stdout)
	}
	for _, call := range result.calls {
		if strings.Contains(call, "symbolic-ref") {
			t.Fatalf("a flag deveria curto-circuitar a detecção, mas rodou %q", call)
		}
	}
}

func heldRepository() map[string]gittest.Response {
	responses := repository("main", "main\nlivre\npresa", "main\nlivre\npresa", "main")
	responses["worktree list --porcelain"] = gittest.Response{Output: "" +
		"worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
		"worktree /outro\nHEAD def\nbranch refs/heads/presa\n"}

	return responses
}

func TestBranchesCommandHoldsBranchUsedByAnotherWorktree(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "  livre\tmergeada") && !strings.Contains(result.stdout, "livre") {
		t.Errorf("a branch livre deveria estar listada, veio %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "em uso por outro working tree") {
		t.Errorf("saída = %q, queria a seção das presas", result.stdout)
	}
	if !strings.Contains(result.stdout, "/outro") {
		t.Errorf("saída = %q, queria o caminho do working tree", result.stdout)
	}
	if !strings.Contains(result.stdout, "checkout --detach") {
		t.Errorf("saída = %q, queria ensinar como soltar", result.stdout)
	}
}

func TestBranchesCommandNeverDeletesBranchUsedByAnotherWorktree(t *testing.T) {
	responses := heldRepository()
	responses["branch -d livre"] = gittest.Response{Output: ""}

	for _, args := range [][]string{
		{"branches", "--clean"},
		{"branches", "--clean", "--force"},
	} {
		result := execute(t, responses, "y\n", args...)

		if result.err != nil {
			t.Fatalf("%v: não esperava erro, veio %v", args, result.err)
		}
		for _, call := range result.calls {
			if strings.Contains(call, "presa") && deletesABranch(call) {
				t.Fatalf("%v: o git recusaria, mas o comando tentou %q", args, call)
			}
		}
	}
}

func TestBranchesCommandCountsOnlyFreeBranches(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches")

	if !strings.Contains(result.stdout, "Branches locais já mergeadas (1):") {
		t.Errorf("a presa não pode entrar na contagem, veio %q", result.stdout)
	}
}

func TestBranchesCommandDegradesWhenWorktreesCannotBeListed(t *testing.T) {
	responses := heldRepository()
	responses["worktree list --porcelain"] = gittest.Response{Err: errNotARepository}

	result := execute(t, responses, "", "branches")

	if result.err != nil {
		t.Fatalf("falha ao listar working trees não pode derrubar o comando, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "presa") {
		t.Errorf("sem a lista de working trees a proteção some, e a branch volta a aparecer; veio %q", result.stdout)
	}
	if strings.Contains(result.stdout, "em uso por outro working tree") {
		t.Errorf("sem informação não dá para afirmar que está presa, veio %q", result.stdout)
	}
}

func TestBranchesCommandCurrentWorktreeDoesNotHoldItsOwnBranch(t *testing.T) {
	responses := repository("main", "main\nfeat-a", "main\nfeat-a", "main")
	responses["worktree list --porcelain"] = gittest.Response{Output: "" +
		"worktree /repo\nHEAD abc\nbranch refs/heads/feat-a\n"}
	responses["rev-parse --show-toplevel"] = gittest.Response{Output: "/repo"}

	result := execute(t, responses, "", "branches")

	if strings.Contains(result.stdout, "em uso por outro working tree") {
		t.Errorf("o worktree atual não prende ninguém, veio %q", result.stdout)
	}
}

func TestBranchesCommandHeldSectionNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, heldRepository(), "", "branches")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}

func onlyHeldRepository() map[string]gittest.Response {
	responses := repository("main", "main\npresa", "main\npresa", "main")
	responses["worktree list --porcelain"] = gittest.Response{Output: "" +
		"worktree /repo\nHEAD abc\nbranch refs/heads/main\n\n" +
		"worktree /outro\nHEAD def\nbranch refs/heads/presa\n"}

	return responses
}

func TestBranchesCommandWithEveryBranchHeld(t *testing.T) {
	result := execute(t, onlyHeldRepository(), "y\n", "branches", "--clean", "--force")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if strings.Contains(result.stdout, "Branches locais já mergeadas") {
		t.Errorf("sem branch livre não há listagem, veio %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "em uso por outro working tree") {
		t.Errorf("saída = %q, queria a seção das presas", result.stdout)
	}
	if !strings.Contains(result.stdout, "Nenhuma branch pode ser deletada agora.") {
		t.Errorf("saída = %q, queria dizer que não há o que deletar", result.stdout)
	}
	if strings.Contains(result.stdout, "[y/N]") {
		t.Error("sem candidata não pode nem perguntar")
	}
}

func run(t *testing.T, responses map[string]gittest.Response, args ...string) (int, string) {
	t.Helper()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(gittest.NewRunner(responses), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	return Run(command), stderr.String()
}

func TestRunReturnsZeroWhenTheCommandSucceeds(t *testing.T) {
	code, stderr := run(t, repository("main", "main\nfeat-a", "main\nfeat-a", "main"), "branches")

	if code != 0 {
		t.Errorf("código = %d, queria 0", code)
	}
	if stderr != "" {
		t.Errorf("RN-09: sem falha o stderr fica vazio, veio %q", stderr)
	}
}

func TestRunReturnsOneAndReportsWhenTheCommandFails(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	code, stderr := run(t, responses, "branches")

	if code != 1 {
		t.Errorf("código = %d, queria 1", code)
	}
	if !strings.Contains(stderr, "erro: ") {
		t.Errorf("stderr = %q, queria o prefixo que o usuário vê", stderr)
	}
	if !strings.Contains(stderr, "não é um repositório git") {
		t.Errorf("stderr = %q, queria carregar o erro de origem", stderr)
	}
}

func TestBranchesCommandPropagatesHeldSectionWriteFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(onlyHeldRepository()), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(brokenWriter{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs([]string{"branches"})

	if command.Execute() == nil {
		t.Fatal("falha de escrita na seção das presas tem de virar erro")
	}
}

// noFinder descreve uma máquina sem gerenciador de pacotes, que é o padrão
// para todo teste que não é do setup.
func noFinder() platform.Finder {
	return platformtest.Finder{System: "linux"}
}

// executeWith roda o comando com respostas de git e de processo externo, para
// quem precisa dos dois roteirizados.
func executeWith(t *testing.T, responses map[string]gittest.Response, outcomes []exectest.Response, args ...string) execution {
	t.Helper()

	runner := gittest.NewRunner(responses)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	command := NewRootCommand(runner, exectest.NewRunner(outcomes...), noWeb(), noFinder(), noNotices)
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)

	err := command.Execute()

	return execution{stdout: stdout.String(), stderr: stderr.String(), err: err, calls: runner.Calls}
}
