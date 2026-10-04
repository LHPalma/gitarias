package redate

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	date        = "2026-10-03T14:00:00"
	shortHead   = "rev-parse --short HEAD"
	headLog     = "log -1 --format=%h%x00%s%x00%aI"
	symbolicRef = "symbolic-ref --short HEAD"
	amend       = "commit --amend --no-edit --allow-empty --date=" + date
)

var errNotARepository = errors.New("fatal: not a git repository")

func windowCall(since string, until string) string {
	call := "log --format=%H"
	if since != "" {
		call += " --since=" + since + " 00:00:00"
	}
	if until != "" {
		call += " --until=" + until + " 23:59:59"
	}

	return call
}

func parents(commit string) string { return "rev-list --parents -n 1 " + commit }

func rangeLog(spec string) string { return "log --format=%h%x00%s%x00%aI " + spec }

func rebaseExec(base string, newest string) string {
	return "rebase --rebase-merges " + base + " " + newest + " --exec git " + amend
}

func rebaseRootExec(newest string) string {
	return "rebase --rebase-merges --root " + newest + " --exec git " + amend
}

func rebaseOnto(newNewest string, oldNewest string, ref string) string {
	return "rebase --onto " + newNewest + " " + oldNewest + " " + ref
}

func commitLine(hash string, subject string, iso string) string {
	return hash + fieldSep + subject + fieldSep + iso
}

func TestParseNormalisesTheDate(t *testing.T) {
	got, err := Parse(date)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if got != date {
		t.Errorf("data = %q, queria %q", got, date)
	}
}

func TestParseRefusesWhatIsNotTheLayout(t *testing.T) {
	for _, input := range []string{"", "2026-10-03", "2026-10-03 14:00:00", "ontem", "2026-13-40T00:00:00", "2026-10-03T14:00:00; rm -rf /"} {
		if _, err := Parse(input); err == nil {
			t.Errorf("Parse(%q) aceitou, queria erro", input)
		}
	}
}

func TestPlanForTheHeadOnly(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead: {Output: "abc1234"},
		headLog:   {Output: commitLine("abc1234", "feat: algo", "2026-09-07T10:00:00+00:00")},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := []Commit{{Hash: "abc1234", Subject: "feat: algo", Date: "2026-09-07T10:00:00+00:00"}}
	if plan.Head != "abc1234" || !slices.Equal(plan.Commits, want) || plan.Tail != 0 {
		t.Errorf("plano = %+v, queria o HEAD abc1234 com %+v", plan, want)
	}
}

func TestPlanForAPeriodToHEAD(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                         {Output: "e5e5e5e"},
		windowCall("2026-09-04", ""):      {Output: "e5e5e5e5\nc4c4c4c4\nc3c3c3c3"},
		parents("c3c3c3c3"):               {Output: "c3c3c3c3 c2c2c2c2"},
		rangeLog("c3c3c3c3^..e5e5e5e5"):   {Output: commitLine("e5e5e5e", "c5", "x") + "\n" + commitLine("c4c4c4c", "c4", "y") + "\n" + commitLine("c3c3c3c", "c3", "z")},
		"rev-list --count e5e5e5e5..HEAD": {Output: "0"},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "2026-09-04", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if len(plan.Commits) != 3 || plan.Tail != 0 {
		t.Errorf("plano = %+v, queria 3 commits e cauda 0", plan)
	}
}

func TestPlanForAPeriodWithAnEmptyWindow(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                    {Output: "abc1234"},
		windowCall("2026-12-01", ""): {Output: ""},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "2026-12-01", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(plan.Commits) != 0 {
		t.Errorf("plano = %+v, queria nenhum commit", plan)
	}
}

func TestPlanCoversTheRootCommit(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                         {Output: "bbb"},
		windowCall("2026-01-01", ""):      {Output: "bbbbbbbb\naaaaaaaa"},
		parents("aaaaaaaa"):               {Output: "aaaaaaaa"},
		rangeLog("bbbbbbbb"):              {Output: commitLine("bbb", "b", "x") + "\n" + commitLine("aaa", "a", "y")},
		"rev-list --count bbbbbbbb..HEAD": {Output: "0"},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "2026-01-01", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(plan.Commits) != 2 {
		t.Errorf("plano = %+v, queria os 2 commits, raiz incluída", plan)
	}
}

func TestPlanCountsALinearTail(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                                  {Output: "ccc"},
		windowCall("", "2026-09-04"):               {Output: "bbbbbbbb\naaaaaaaa"},
		parents("aaaaaaaa"):                        {Output: "aaaaaaaa"},
		rangeLog("bbbbbbbb"):                       {Output: commitLine("bbb", "b", "x")},
		"rev-list --count bbbbbbbb..HEAD":          {Output: "2"},
		"rev-list --merges --count bbbbbbbb..HEAD": {Output: "0"},
	})

	plan, err := NewRepo(runner).Plan(t.Context(), "", "2026-09-04")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if plan.Tail != 2 {
		t.Errorf("cauda = %d, queria 2", plan.Tail)
	}
}

func TestPlanRefusesAMergeInTheTail(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                                  {Output: "ccc"},
		windowCall("", "2026-09-04"):               {Output: "bbbbbbbb\naaaaaaaa"},
		parents("aaaaaaaa"):                        {Output: "aaaaaaaa"},
		rangeLog("bbbbbbbb"):                       {Output: commitLine("bbb", "b", "x")},
		"rev-list --count bbbbbbbb..HEAD":          {Output: "3"},
		"rev-list --merges --count bbbbbbbb..HEAD": {Output: "1"},
	})

	_, err := NewRepo(runner).Plan(t.Context(), "", "2026-09-04")
	if !errors.Is(err, ErrMergesInTail) {
		t.Errorf("erro = %v, queria ErrMergesInTail", err)
	}
}

func TestPlanPropagatesTheFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Plan(t.Context(), "", ""); !errors.Is(err, errNotARepository) {
		t.Errorf("erro = %v, queria %v", err, errNotARepository)
	}
}

func TestRewriteTheHeadAmendsWithBothDates(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{amend: {}})

	if err := NewRepo(runner).Rewrite(t.Context(), "", "", date); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if want := []string{"GIT_COMMITTER_DATE=" + date}; !slices.Equal(runner.Envs[amend], want) {
		t.Errorf("ambiente = %v, queria %v", runner.Envs[amend], want)
	}
}

func TestRewriteAPeriodRebasesThenReplaysTheTail(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-09-04", ""):               {Output: "e5e5e5e5\nc3c3c3c3"},
		symbolicRef:                                {Output: "main"},
		parents("c3c3c3c3"):                        {Output: "c3c3c3c3 c2c2c2c2"},
		rebaseExec("c3c3c3c3^", "e5e5e5e5"):        {},
		"rev-parse HEAD":                           {Output: "f0f0f0f0"},
		rebaseOnto("f0f0f0f0", "e5e5e5e5", "main"): {},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "2026-09-04", "", date); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if want := []string{"GIT_COMMITTER_DATE=" + date}; !slices.Equal(runner.Envs[rebaseExec("c3c3c3c3^", "e5e5e5e5")], want) {
		t.Errorf("ambiente da rebase = %v, queria %v", runner.Envs[rebaseExec("c3c3c3c3^", "e5e5e5e5")], want)
	}
}

func TestRewriteAPeriodReachingTheRootUsesRoot(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-01-01", ""):               {Output: "bbbbbbbb\naaaaaaaa"},
		symbolicRef:                                {Output: "main"},
		parents("aaaaaaaa"):                        {Output: "aaaaaaaa"},
		rebaseRootExec("bbbbbbbb"):                 {},
		"rev-parse HEAD":                           {Output: "f0f0f0f0"},
		rebaseOnto("f0f0f0f0", "bbbbbbbb", "main"): {},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "2026-01-01", "", date); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}

func TestRewriteWithADetachedHeadFallsBackToTheSHA(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-09-04", ""):                   {Output: "e5e5e5e5\nc3c3c3c3"},
		symbolicRef:                                    {Err: &git.ExitError{Code: 128}},
		"rev-parse HEAD":                               {Output: "f0f0f0f0"},
		parents("c3c3c3c3"):                            {Output: "c3c3c3c3 c2c2c2c2"},
		rebaseExec("c3c3c3c3^", "e5e5e5e5"):            {},
		rebaseOnto("f0f0f0f0", "e5e5e5e5", "f0f0f0f0"): {},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "2026-09-04", "", date); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
}

func TestRewriteAnEmptyWindowTouchesNothing(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-12-01", ""): {Output: ""},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "2026-12-01", "", date); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(runner.Calls) != 1 {
		t.Errorf("chamadas = %v, queria só a leitura da janela", runner.Calls)
	}
}

func TestRewritePropagatesTheRebaseFailure(t *testing.T) {
	failure := errors.New("conflito")
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-09-04", ""):        {Output: "e5e5e5e5\nc3c3c3c3"},
		symbolicRef:                         {Output: "main"},
		parents("c3c3c3c3"):                 {Output: "c3c3c3c3 c2c2c2c2"},
		rebaseExec("c3c3c3c3^", "e5e5e5e5"): {Err: failure},
	})

	if err := NewRepo(runner).Rewrite(t.Context(), "2026-09-04", "", date); !errors.Is(err, failure) {
		t.Errorf("erro = %v, queria %v", err, failure)
	}
}

func TestEnsureOutsideARepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	})

	if err := NewRepo(runner).Ensure(t.Context()); err == nil || !strings.Contains(err.Error(), "repositório") {
		t.Errorf("erro = %v, queria o aviso de fora de repositório", err)
	}
}
