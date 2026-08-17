package author

import (
	"errors"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	shortHead   = "rev-parse --short HEAD"
	lastAuthor  = "log -1 --format=%an <%ae>"
	amend       = "commit --amend --no-edit --reset-author"
	symbolicRef = "symbolic-ref --short HEAD"
)

var errNotARepository = errors.New("fatal: not a git repository")

func rangeCount(from string, to string) string {
	return "rev-list --count " + from + ".." + to
}

func rangeAuthors(from string, to string) string {
	return "log " + from + ".." + to + " --format=%an <%ae>"
}

func rebaseExec(base string, untilSHA string) string {
	return "rebase " + base + " " + untilSHA + " --exec git commit --amend --no-edit --reset-author"
}

func rebaseOnto(newUntilSHA string, oldUntilSHA string, ref string) string {
	return "rebase --onto " + newUntilSHA + " " + oldUntilSHA + " " + ref
}

func identity(name string, email string) []string {
	return []string{
		"GIT_AUTHOR_NAME=" + name,
		"GIT_AUTHOR_EMAIL=" + email,
		"GIT_COMMITTER_NAME=" + name,
		"GIT_COMMITTER_EMAIL=" + email,
	}
}

func TestPlanForTheLastCommit(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:  {Output: "abc1234"},
		lastAuthor: {Output: "Real Person <real@real.com>"},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := Plan{Head: "abc1234", Count: 1, Author: "Real Person <real@real.com>"}
	if plan.Head != want.Head || plan.Count != want.Count || plan.Author != want.Author || len(plan.Authors) != 0 || plan.Tail != 0 {
		t.Errorf("plano = %+v, queria %+v", plan, want)
	}
}

func TestPlanForARangeToHEAD(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                  {Output: "abc1234"},
		rangeCount("main", "HEAD"): {Output: "3\n"},
		rangeAuthors("main", "HEAD"): {Output: "" +
			"Real Person <real@real.com>\n" +
			"Other Person <other@other.com>\n" +
			"Real Person <real@real.com>\n"},
		rangeCount("HEAD", "HEAD"): {Output: "0"},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "main", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if plan.Count != 3 {
		t.Errorf("count = %d, queria 3", plan.Count)
	}
	if plan.Author != "" {
		t.Errorf("author = %q, numa faixa não pode afirmar um autor só", plan.Author)
	}
	if plan.Tail != 0 {
		t.Errorf("tail = %d, uma faixa que vai até o HEAD não preserva nada", plan.Tail)
	}

	want := []string{"Other Person <other@other.com>", "Real Person <real@real.com>"}
	if len(plan.Authors) != len(want) {
		t.Fatalf("autores = %v, queria %v", plan.Authors, want)
	}
	for index := range want {
		if plan.Authors[index] != want[index] {
			t.Errorf("autores[%d] = %q, queria %q", index, plan.Authors[index], want[index])
		}
	}
}

func TestPlanForARangeWithUntilKeepsATail(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                        {Output: "abc1234"},
		rangeCount("main", "deadbeef"):   {Output: "2"},
		rangeAuthors("main", "deadbeef"): {Output: "Real Person <real@real.com>\n"},
		rangeCount("deadbeef", "HEAD"):   {Output: "5"},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "main", "deadbeef")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if plan.Count != 2 {
		t.Errorf("count = %d, queria 2", plan.Count)
	}
	if plan.Tail != 5 {
		t.Errorf("tail = %d, queria 5 — o que vem depois de until tem de ser contado à parte", plan.Tail)
	}
}

func TestPlanRefusesAnEmptyRange(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                  {Output: "abc1234"},
		rangeCount("main", "HEAD"): {Output: "0"},
	})

	_, err := NewRepo(runner).Plan(t.Context(), "main", "")
	if err == nil {
		t.Fatal("intervalo sem nenhum commit tem de virar erro")
	}
}

func TestPlanRejectsAnUnreadableCount(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                  {Output: "abc1234"},
		rangeCount("main", "HEAD"): {Output: "não é um número"},
	})

	if _, err := NewRepo(runner).Plan(t.Context(), "main", ""); err == nil {
		t.Fatal("contagem ilegível tem de virar erro, não zero silencioso")
	}
}

func TestPlanPropagatesRevParseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Plan(t.Context(), "", ""); err == nil {
		t.Fatal("falha do rev-parse tem de virar erro")
	}
}

func TestPlanPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:  {Output: "abc1234"},
		lastAuthor: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Plan(t.Context(), "", ""); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestPlanPropagatesRevListFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                  {Output: "abc1234"},
		rangeCount("main", "HEAD"): {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Plan(t.Context(), "main", ""); err == nil {
		t.Fatal("falha do rev-list tem de virar erro")
	}
}

func TestPlanPropagatesAuthorsFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                    {Output: "abc1234"},
		rangeCount("main", "HEAD"):   {Output: "1"},
		rangeAuthors("main", "HEAD"): {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Plan(t.Context(), "main", ""); err == nil {
		t.Fatal("falha ao listar autores tem de virar erro")
	}
}

func TestPlanPropagatesTailFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                    {Output: "abc1234"},
		rangeCount("main", "HEAD"):   {Output: "1"},
		rangeAuthors("main", "HEAD"): {Output: "Real Person <real@real.com>\n"},
		rangeCount("HEAD", "HEAD"):   {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Plan(t.Context(), "main", ""); err == nil {
		t.Fatal("falha ao contar o rabo tem de virar erro")
	}
}

func TestRewriteTheLastCommit(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{amend: {Output: ""}})

	err := NewRepo(runner).Rewrite(t.Context(), "", "", "Fake Name", "fake@fake.com")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	got := runner.Envs[amend]
	want := identity("Fake Name", "fake@fake.com")
	if len(got) != len(want) {
		t.Fatalf("ambiente = %v, queria %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("ambiente[%d] = %q, queria %q", index, got[index], want[index])
		}
	}
}

func TestRewriteARangeToHEAD(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse HEAD":               {Output: "headsha1"},
		symbolicRef:                    {Output: "feature"},
		rebaseExec("main", "headsha1"): {Output: ""},
		rebaseOnto("headsha1", "headsha1", "feature"): {Output: ""},
	})

	err := NewRepo(runner).Rewrite(t.Context(), "main", "", "Fake Name", "fake@fake.com")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	got := runner.Envs[rebaseExec("main", "headsha1")]
	want := identity("Fake Name", "fake@fake.com")
	if len(got) != len(want) {
		t.Fatalf("ambiente = %v, queria %v", got, want)
	}
}

func TestRewriteARangeWithUntilPreservesTheTail(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse deadbeef":               {Output: "untilsha-old"},
		symbolicRef:                        {Output: "feature"},
		rebaseExec("main", "untilsha-old"): {Output: ""},
		"rev-parse HEAD":                   {Output: "untilsha-new"},
		rebaseOnto("untilsha-new", "untilsha-old", "feature"): {Output: ""},
	})

	err := NewRepo(runner).Rewrite(t.Context(), "main", "deadbeef", "Fake Name", "fake@fake.com")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	var rebasedOnto bool
	for _, call := range runner.Calls {
		if call == rebaseOnto("untilsha-new", "untilsha-old", "feature") {
			rebasedOnto = true
		}
	}
	if !rebasedOnto {
		t.Errorf("chamadas = %v, o reencaixe do rabo tinha de ter rodado", runner.Calls)
	}
}

func TestRewriteFallsBackToTheHeadSHAWhenDetached(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse HEAD":               {Output: "headsha1"},
		symbolicRef:                    {Err: errNotARepository},
		rebaseExec("main", "headsha1"): {Output: ""},
		rebaseOnto("headsha1", "headsha1", "headsha1"): {Output: ""},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "main", "", "Fake Name", "fake@fake.com"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	var rebasedOnto bool
	for _, call := range runner.Calls {
		if call == rebaseOnto("headsha1", "headsha1", "headsha1") {
			rebasedOnto = true
		}
	}
	if !rebasedOnto {
		t.Errorf("chamadas = %v, com HEAD destacado o reencaixe tem de usar o SHA como referência", runner.Calls)
	}
}

// TestRewriteNeverInterpolatesTheIdentityIntoTheExecString prova a defesa
// contra injeção: mesmo com metacaracteres de shell no nome, a chamada de
// --exec continua sendo o literal fixo — a identidade só existe no
// ambiente, nunca dentro da string que o rebase manda pro shell.
func TestRewriteNeverInterpolatesTheIdentityIntoTheExecString(t *testing.T) {
	dangerous := "$(touch /tmp/pwned)`id`"
	call := rebaseExec("main", "headsha1")
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse HEAD": {Output: "headsha1"},
		symbolicRef:      {Output: "feature"},
		call:             {Output: ""},
		rebaseOnto("headsha1", "headsha1", "feature"): {Output: ""},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "main", "", dangerous, "a@a.com"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	var found bool
	for _, executed := range runner.Calls {
		if executed == call {
			found = true
		}
	}
	if !found {
		t.Fatalf("chamadas = %v, o --exec tem de ser sempre o mesmo literal, veio diferente do esperado", runner.Calls)
	}
}

func TestRewritePropagatesTheLastCommitFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{amend: {Err: errNotARepository}})

	if err := NewRepo(runner).Rewrite(t.Context(), "", "", "Fake Name", "fake@fake.com"); err == nil {
		t.Fatal("falha do amend tem de virar erro")
	}
}

func TestRewritePropagatesTheUntilResolutionFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse HEAD": {Err: errNotARepository},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "main", "", "Fake Name", "fake@fake.com"); err == nil {
		t.Fatal("falha ao resolver o until tem de virar erro")
	}
}

func TestRewritePropagatesTheDetachedFallbackFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse deadbeef": {Output: "untilsha-old"},
		symbolicRef:          {Err: errNotARepository},
		"rev-parse HEAD":     {Err: errNotARepository},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "main", "deadbeef", "Fake Name", "fake@fake.com"); err == nil {
		t.Fatal("falha ao resolver a ref atual, mesmo destacado, tem de virar erro")
	}
}

func TestRewritePropagatesTheNewUntilResolutionFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse deadbeef":               {Output: "untilsha-old"},
		symbolicRef:                        {Output: "feature"},
		rebaseExec("main", "untilsha-old"): {Output: ""},
		"rev-parse HEAD":                   {Err: errNotARepository},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "main", "deadbeef", "Fake Name", "fake@fake.com"); err == nil {
		t.Fatal("falha ao capturar o novo until depois do rebase tem de virar erro")
	}
}

func TestRewritePropagatesTheRebaseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse HEAD":               {Output: "headsha1"},
		symbolicRef:                    {Output: "feature"},
		rebaseExec("main", "headsha1"): {Err: errNotARepository},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "main", "", "Fake Name", "fake@fake.com"); err == nil {
		t.Fatal("falha do rebase tem de virar erro")
	}
}

func TestRewritePropagatesTheOntoFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse HEAD":               {Output: "headsha1"},
		symbolicRef:                    {Output: "feature"},
		rebaseExec("main", "headsha1"): {Output: ""},
		rebaseOnto("headsha1", "headsha1", "feature"): {Err: errNotARepository},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "main", "", "Fake Name", "fake@fake.com"); err == nil {
		t.Fatal("falha do reencaixe tem de virar erro")
	}
}

func TestEnsureRejectsWhatIsNotARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	})

	err := NewRepo(runner).Ensure(t.Context())
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestEnsureAcceptsARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
	})

	if err := NewRepo(runner).Ensure(t.Context()); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}
