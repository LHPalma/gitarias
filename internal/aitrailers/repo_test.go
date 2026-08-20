package aitrailers

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	verifyHead  = "rev-parse --verify --quiet HEAD"
	logFormat   = "log --format=%H%x00%s%x00%(trailers:only,unfold)%x01"
	shortHead   = "rev-parse --short HEAD"
	headLog     = "log -1 --format=%H%x00%s%x00%(trailers:only,unfold)"
	stripLog    = "log -1 --format=%B%x00%(trailers:only)%x00%(trailers:only,unfold)"
	amend       = "commit --amend --allow-empty -F -"
	symbolicRef = "symbolic-ref --short HEAD"
	windowLog   = "log --format=%H"
)

var errNotARepository = errors.New("fatal: not a git repository")

// record monta um registro exatamente como o git devolveria: hash, assunto e
// o bloco de trailers (vazio ou não), separados por \x00 e fechados por
// \x01. trailerLines vazio simula um commit sem trailer nenhum.
func record(hash string, subject string, trailerLines ...string) string {
	return hash + fieldSep + subject + fieldSep + strings.Join(trailerLines, "\n") + recordSep
}

func logCall(since string, until string) string {
	call := logFormat
	if since != "" {
		call += " --since=" + since + " 00:00:00"
	}
	if until != "" {
		call += " --until=" + until + " 23:59:59"
	}

	return call
}

func windowCall(since string, until string) string {
	call := windowLog
	if since != "" {
		call += " --since=" + since + " 00:00:00"
	}
	if until != "" {
		call += " --until=" + until + " 23:59:59"
	}

	return call
}

func rebaseExec(base string, newest string) string {
	return "rebase " + base + " " + newest + " --exec gtr " + StripStepCommand
}

func rebaseOnto(newNewest string, oldNewest string, ref string) string {
	return "rebase --onto " + newNewest + " " + oldNewest + " " + ref
}

func withLog(records ...string) map[string]gittest.Response {
	return map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: strings.Join(records, "")},
	}
}

func TestListFindsClaudeCodeCoAuthoredBy(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: Claude <noreply@anthropic.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || findings[0].Trailers[0].Tool != "Claude Code" {
		t.Errorf("achados = %+v, queria um achado de Claude Code", findings)
	}
}

func TestListFindsClaudeSessionRegardlessOfCoAuthor(t *testing.T) {
	log := record("a", "feat: algo", "Claude-Session: https://claude.ai/code/session_123")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || findings[0].Trailers[0].Tool != "Claude Code" {
		t.Errorf("achados = %+v, queria um achado de Claude Code via Claude-Session", findings)
	}
}

func TestListFindsGitHubCopilot(t *testing.T) {
	log := record("a", "feat: algo", "Co-authored-by: Copilot <198982749+Copilot@users.noreply.github.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || findings[0].Trailers[0].Tool != "GitHub Copilot" {
		t.Errorf("achados = %+v, queria um achado de GitHub Copilot", findings)
	}
}

func TestListMatchesCoAuthoredByKeyCaseInsensitively(t *testing.T) {
	log := record("a", "feat: algo", "Co-authored-by: Claude <noreply@anthropic.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("achados = %+v, a chave do trailer não pode depender de caixa", findings)
	}
}

func TestListIgnoresARealHumanCoAuthor(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: Real Human <human@example.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, co-autor humano não é achado", findings)
	}
}

func TestListIgnoresAHumanUsingAGitHubNoreplyEmail(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: Real Human <123+realhuman@users.noreply.github.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, e-mail noreply do GitHub sozinho não basta — é comum entre humanos", findings)
	}
}

func TestListIgnoresAnUnrelatedTrailerKey(t *testing.T) {
	log := record("a", "feat: algo", "Signed-off-by: Real Human <human@example.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, Signed-off-by não é assinatura reconhecida", findings)
	}
}

func TestListIgnoresACoAuthorValueWithoutAnEmail(t *testing.T) {
	log := record("a", "feat: algo", "Co-Authored-By: sem e-mail nenhum")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, valor sem <e-mail> não tem como bater em assinatura nenhuma", findings)
	}
}

func TestListIgnoresAMalformedTrailerLineWithoutAColon(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: "a" + fieldSep + "feat: algo" + fieldSep + "linha sem dois-pontos" + recordSep},
	})

	findings, err := NewRepo(runner).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, linha sem \": \" não é trailer nenhum", findings)
	}
}

func TestListIgnoresARecordMissingAField(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: "registro sem separador nenhum" + recordSep},
	})

	findings, err := NewRepo(runner).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if findings != nil {
		t.Errorf("achados = %v, registro malformado não pode virar achado", findings)
	}
}

func TestListIgnoresARecordMissingTheTrailerField(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Output: "a" + fieldSep + "feat: algo, sem o segundo separador" + recordSep},
	})

	findings, err := NewRepo(runner).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if findings != nil {
		t.Errorf("achados = %v, registro sem o campo de trailers não pode virar achado", findings)
	}
}

func TestListSkipsCommitsWithoutMatchingTrailers(t *testing.T) {
	log := record("a", "feat: sem trailer nenhum") + record("b", "feat: outro")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("achados = %+v, nenhum dos dois tem trailer reconhecido", findings)
	}
}

func TestListOnlyKeepsTheMatchingTrailerAlongsideAHumanOne(t *testing.T) {
	log := record("a", "feat: algo",
		"Co-Authored-By: Real Human <human@example.com>",
		"Co-Authored-By: Claude <noreply@anthropic.com>")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 || len(findings[0].Trailers) != 1 {
		t.Fatalf("achados = %+v, o trailer humano não pode aparecer junto", findings)
	}
	if findings[0].Trailers[0].Value != "Claude <noreply@anthropic.com>" {
		t.Errorf("trailer = %+v, queria só o de Claude", findings[0].Trailers[0])
	}
}

func TestListKeepsNewestFirst(t *testing.T) {
	log := record("b", "feat: segundo", "Claude-Session: https://claude.ai/code/session_2") +
		record("a", "feat: primeiro", "Claude-Session: https://claude.ai/code/session_1")

	findings, err := NewRepo(gittest.NewRunner(withLog(log))).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 2 || findings[0].Hash != "b" || findings[1].Hash != "a" {
		t.Errorf("achados = %+v, queria a ordem do git log preservada", findings)
	}
}

func TestListFiltersSince(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                {Output: "abc123"},
		logCall("2026-08-10", ""): {Output: record("a", "feat: algo", "Claude-Session: x")},
	})

	findings, err := NewRepo(runner).List(t.Context(), "2026-08-10", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("achados = %v, queria um", findings)
	}
}

func TestListFiltersUntil(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                {Output: "abc123"},
		logCall("", "2026-08-10"): {Output: record("a", "feat: algo", "Claude-Session: x")},
	})

	findings, err := NewRepo(runner).List(t.Context(), "", "2026-08-10")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("achados = %v, queria um", findings)
	}
}

func TestListFiltersSinceAndUntil(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                          {Output: "abc123"},
		logCall("2026-08-01", "2026-08-10"): {Output: record("a", "feat: algo", "Claude-Session: x")},
	})

	findings, err := NewRepo(runner).List(t.Context(), "2026-08-01", "2026-08-10")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("achados = %v, queria um", findings)
	}
}

func TestStripHeadAmendsWhenAnAITrailerIsPresent(t *testing.T) {
	block := "Co-Authored-By: Claude <noreply@anthropic.com>\n"
	raw := "feat: algo\n\nbody\n\n" + block

	runner := gittest.NewRunner(map[string]gittest.Response{
		stripLog: {Output: raw + fieldSep + block + fieldSep + block},
		amend:    {Output: ""},
	})

	changed, err := NewRepo(runner).StripHead(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !changed {
		t.Fatal("esperava changed = true")
	}

	want := "feat: algo\n\nbody"
	if got := runner.Inputs[amend]; got != want {
		t.Errorf("mensagem enviada ao amend = %q, queria %q", got, want)
	}
}

func TestStripHeadDoesNothingWithoutAnAITrailer(t *testing.T) {
	block := "Co-Authored-By: Real Human <human@example.com>\n"
	raw := "feat: algo\n\nbody\n\n" + block

	runner := gittest.NewRunner(map[string]gittest.Response{
		stripLog: {Output: raw + fieldSep + block + fieldSep + block},
	})

	changed, err := NewRepo(runner).StripHead(t.Context())
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if changed {
		t.Fatal("esperava changed = false")
	}

	for _, call := range runner.Calls {
		if call == amend {
			t.Fatalf("chamadas = %v, sem trailer de IA o amend não podia rodar", runner.Calls)
		}
	}
}

func TestStripHeadRejectsAnIncompleteRecord(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		stripLog: {Output: "só um campo, sem separador nenhum"},
	})

	if _, err := NewRepo(runner).StripHead(t.Context()); err == nil {
		t.Fatal("registro incompleto tem de virar erro")
	}
}

func TestStripHeadPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		stripLog: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).StripHead(t.Context()); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestStripHeadPropagatesAmendFailure(t *testing.T) {
	block := "Claude-Session: x\n"
	raw := "feat: algo\n\n" + block

	runner := gittest.NewRunner(map[string]gittest.Response{
		stripLog: {Output: raw + fieldSep + block + fieldSep + block},
		amend:    {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).StripHead(t.Context()); err == nil {
		t.Fatal("falha do amend tem de virar erro")
	}
}

func TestPlanStripOnTheHeadWithAMatch(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead: {Output: "abc123"},
		headLog:   {Output: "abc123" + fieldSep + "feat: algo" + fieldSep + "Claude-Session: x"},
	})

	plan, err := NewRepo(runner).PlanStrip(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if plan.Head != "abc123" || len(plan.Findings) != 1 {
		t.Errorf("plan = %+v, queria um achado no HEAD", plan)
	}
}

func TestPlanStripOnTheHeadWithoutAMatch(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead: {Output: "abc123"},
		headLog:   {Output: "abc123" + fieldSep + "feat: algo" + fieldSep + ""},
	})

	plan, err := NewRepo(runner).PlanStrip(t.Context(), "", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if plan.Head != "abc123" || plan.Findings != nil {
		t.Errorf("plan = %+v, HEAD sem trailer de IA não pode virar achado", plan)
	}
}

func TestPlanStripWithARangeDelegatesToList(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                 {Output: "abc123"},
		verifyHead:                {Output: "abc123"},
		logCall("2026-08-10", ""): {Output: record("a", "feat: algo", "Claude-Session: x")},
	})

	plan, err := NewRepo(runner).PlanStrip(t.Context(), "2026-08-10", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(plan.Findings) != 1 {
		t.Errorf("plan = %+v, queria o achado que o List traria pro mesmo período", plan)
	}
}

func TestPlanStripPropagatesShortHeadFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).PlanStrip(t.Context(), "", ""); err == nil {
		t.Fatal("falha do rev-parse --short tem de virar erro")
	}
}

func TestPlanStripWithARangePropagatesListFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                 {Output: "abc123"},
		verifyHead:                {Output: "abc123"},
		logCall("2026-08-10", ""): {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).PlanStrip(t.Context(), "2026-08-10", ""); err == nil {
		t.Fatal("falha do List no período tem de virar erro")
	}
}

func TestWindowReturnsOldestAndNewest(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"): {Output: "newest\nmiddle\noldest"},
	})

	oldest, newest, err := NewRepo(runner).window(t.Context(), "2026-08-01", "2026-08-10")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if oldest != "oldest" || newest != "newest" {
		t.Errorf("oldest = %q, newest = %q", oldest, newest)
	}
}

func TestWindowWithNothingInRange(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"): {Output: ""},
	})

	oldest, newest, err := NewRepo(runner).window(t.Context(), "2026-08-01", "2026-08-10")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if oldest != "" || newest != "" {
		t.Errorf("oldest = %q, newest = %q, período vazio tem de devolver os dois vazios", oldest, newest)
	}
}

func TestWindowPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", ""): {Err: errNotARepository},
	})

	if _, _, err := NewRepo(runner).window(t.Context(), "2026-08-01", ""); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestStripOnTheHeadDelegatesToStripHead(t *testing.T) {
	block := "Claude-Session: x\n"
	raw := "feat: algo\n\n" + block

	runner := gittest.NewRunner(map[string]gittest.Response{
		stripLog: {Output: raw + fieldSep + block + fieldSep + block},
		amend:    {Output: ""},
	})

	if err := NewRepo(runner).Strip(t.Context(), "", ""); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if runner.Inputs[amend] == "" {
		t.Error("sem since/until o Strip tinha de reescrever só o HEAD, e o amend não rodou")
	}
}

func TestStripARangeRebasesAndReattachesTheTail(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"):              {Output: "newest\noldest"},
		symbolicRef:                                         {Output: "feature"},
		rebaseExec("oldest^", "newest"):                     {Output: ""},
		"rev-parse HEAD":                                    {Output: "newest-rewritten"},
		rebaseOnto("newest-rewritten", "newest", "feature"): {Output: ""},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", "2026-08-10"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	var reattached bool
	for _, call := range runner.Calls {
		if call == rebaseOnto("newest-rewritten", "newest", "feature") {
			reattached = true
		}
	}
	if !reattached {
		t.Errorf("chamadas = %v, o reencaixe do rabo tinha de ter rodado", runner.Calls)
	}
}

func TestStripARangeFallsBackToTheHeadSHAWhenDetached(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"): {Output: "newest\noldest"},
		symbolicRef:                            {Err: errNotARepository},
		"rev-parse HEAD":                       {Output: "newest-rewritten"},
		rebaseExec("oldest^", "newest"):        {Output: ""},
		rebaseOnto("newest-rewritten", "newest", "newest-rewritten"): {Output: ""},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", "2026-08-10"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	var reattached bool
	for _, call := range runner.Calls {
		if call == rebaseOnto("newest-rewritten", "newest", "newest-rewritten") {
			reattached = true
		}
	}
	if !reattached {
		t.Errorf("chamadas = %v, com HEAD destacado o reencaixe tem de usar o SHA como referência", runner.Calls)
	}
}

func TestStripARangeWithNothingInWindowDoesNothing(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"): {Output: ""},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", "2026-08-10"); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "rebase") {
			t.Fatalf("chamadas = %v, sem commit no período não há o que reescrever", runner.Calls)
		}
	}
}

func TestStripARangePropagatesWindowFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", ""): {Err: errNotARepository},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", ""); err == nil {
		t.Fatal("falha ao calcular a janela tem de virar erro")
	}
}

func TestStripARangePropagatesTheRebaseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"): {Output: "newest\noldest"},
		symbolicRef:                            {Output: "feature"},
		rebaseExec("oldest^", "newest"):        {Err: errNotARepository},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", "2026-08-10"); err == nil {
		t.Fatal("falha da rebase tem de virar erro")
	}
}

func TestStripARangePropagatesTheNewestSHAFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"): {Output: "newest\noldest"},
		symbolicRef:                            {Output: "feature"},
		rebaseExec("oldest^", "newest"):        {Output: ""},
		"rev-parse HEAD":                       {Err: errNotARepository},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", "2026-08-10"); err == nil {
		t.Fatal("falha ao ler o novo SHA de newest tem de virar erro")
	}
}

func TestStripARangePropagatesTheReattachFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"):              {Output: "newest\noldest"},
		symbolicRef:                                         {Output: "feature"},
		rebaseExec("oldest^", "newest"):                     {Output: ""},
		"rev-parse HEAD":                                    {Output: "newest-rewritten"},
		rebaseOnto("newest-rewritten", "newest", "feature"): {Err: errNotARepository},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", "2026-08-10"); err == nil {
		t.Fatal("falha no reencaixe do rabo tem de virar erro")
	}
}

func TestStripARangePropagatesTheDetachedHeadFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		windowCall("2026-08-01", "2026-08-10"): {Output: "newest\noldest"},
		symbolicRef:                            {Err: errNotARepository},
		"rev-parse HEAD":                       {Err: errNotARepository},
	})

	if err := NewRepo(runner).Strip(t.Context(), "2026-08-01", "2026-08-10"); err == nil {
		t.Fatal("falha nos dois jeitos de achar o ref tem de virar erro")
	}
}

func TestPlanStripPropagatesHeadFindingFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead: {Output: "abc123"},
		headLog:   {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).PlanStrip(t.Context(), "", ""); err == nil {
		t.Fatal("falha ao checar o HEAD tem de virar erro")
	}
}

func TestListWithNoCommitsYet(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	findings, err := NewRepo(runner).List(t.Context(), "", "")
	if err != nil {
		t.Fatalf("repositório sem commit ainda não é erro, veio %v", err)
	}
	if findings != nil {
		t.Errorf("achados = %v, queria nenhum", findings)
	}

	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "log") {
			t.Fatalf("sem HEAD não há o que perguntar ao log, mas rodou %q", call)
		}
	}
}

func TestListPropagatesRevParseFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).List(t.Context(), "", ""); err == nil {
		t.Fatal("falha real do rev-parse não pode virar lista vazia")
	}
}

func TestListPropagatesLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		logFormat:  {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).List(t.Context(), "", ""); err == nil {
		t.Fatal("falha do log tem de virar erro")
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
	if !strings.Contains(err.Error(), "não é um repositório git") {
		t.Errorf("erro = %v, queria o do EnsureRepo", err)
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
