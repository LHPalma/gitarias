package overdub

import (
	"errors"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/exec/exectest"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

var errNotARepository = errors.New("fatal: not a git repository")

const (
	shortHead   = "rev-parse --short HEAD"
	symbolicRef = "symbolic-ref --short HEAD"

	fullTarget   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fullOldUntil = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	fullOldHead  = "cccccccccccccccccccccccccccccccccccccccc"
)

func rangeCount(from string, to string) string {
	return "rev-list --count " + from + ".." + to
}

func sequenceEditor(sha string) string {
	return "GIT_SEQUENCE_EDITOR=gtr " + SequenceStepCommand + " " + sha
}

func rebaseInteractive(sha string, untilSHA string) string {
	return "-c core.abbrev=40 rebase -i " + sha + "^ " + untilSHA
}

func rebaseOnto(newUntilSHA string, oldUntilSHA string, ref string) string {
	return "rebase --onto " + newUntilSHA + " " + oldUntilSHA + " " + ref
}

func TestPlan(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                       {Output: "abc1234"},
		"rev-parse --short deadbeef":    {Output: "dead123"},
		"log -1 --format=%s deadbeef":   {Output: "quebrou o build"},
		rangeCount("deadbeef^", "HEAD"): {Output: "5"},
	})

	plan, err := NewRepo(runner, exectest.NewRunner()).Plan(t.Context(), "deadbeef", "")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := Plan{Head: "abc1234", Target: "dead123", Subject: "quebrou o build", Count: 5}
	if plan != want {
		t.Errorf("plano = %+v, queria %+v", plan, want)
	}
}

func TestPlanWithUntil(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                           {Output: "abc1234"},
		"rev-parse --short deadbeef":        {Output: "dead123"},
		"log -1 --format=%s deadbeef":       {Output: "quebrou"},
		rangeCount("deadbeef^", "cafefeed"): {Output: "2"},
	})

	plan, err := NewRepo(runner, exectest.NewRunner()).Plan(t.Context(), "deadbeef", "cafefeed")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if plan.Count != 2 {
		t.Errorf("count = %d, queria 2", plan.Count)
	}
}

func TestPlanRejectsAnUnreadableCount(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		shortHead:                       {Output: "abc1234"},
		"rev-parse --short deadbeef":    {Output: "dead123"},
		"log -1 --format=%s deadbeef":   {Output: "quebrou"},
		rangeCount("deadbeef^", "HEAD"): {Output: "não é um número"},
	})

	if _, err := NewRepo(runner, exectest.NewRunner()).Plan(t.Context(), "deadbeef", ""); err == nil {
		t.Fatal("contagem ilegível tem de virar erro, não zero silencioso")
	}
}

func TestPlanPropagatesEachFailure(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]gittest.Response
	}{
		{name: "head", responses: map[string]gittest.Response{shortHead: {Err: errNotARepository}}},
		{
			name: "sha desconhecido",
			responses: map[string]gittest.Response{
				shortHead:                    {Output: "abc1234"},
				"rev-parse --short deadbeef": {Err: errNotARepository},
			},
		},
		{
			name: "log do assunto",
			responses: map[string]gittest.Response{
				shortHead:                     {Output: "abc1234"},
				"rev-parse --short deadbeef":  {Output: "dead123"},
				"log -1 --format=%s deadbeef": {Err: errNotARepository},
			},
		},
		{
			name: "contagem",
			responses: map[string]gittest.Response{
				shortHead:                       {Output: "abc1234"},
				"rev-parse --short deadbeef":    {Output: "dead123"},
				"log -1 --format=%s deadbeef":   {Output: "quebrou"},
				rangeCount("deadbeef^", "HEAD"): {Err: errNotARepository},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewRepo(gittest.NewRunner(test.responses), exectest.NewRunner()).Plan(t.Context(), "deadbeef", ""); err == nil {
				t.Fatal("esperava erro, veio nil")
			}
		})
	}
}

// baseOverdubResponses cobre o caminho feliz com until explícito
// (fullOldUntil), distinto de HEAD — o que mantém "rev-parse HEAD" fora da
// resolução de untilSHA, só aparecendo depois, para o sha pós-amend e o sha
// final. O fake responde ao mesmo texto de comando sempre com o mesmo
// valor, então os dois coincidem em fullOldHead nestes testes; a ordem e
// os argumentos de cada chamada são o que os testes conferem, e o
// mecanismo real (rebase de verdade, com hash em cascata) foi validado à
// parte, manualmente, contra um repositório real.
func baseOverdubResponses() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse " + fullTarget:                   {Output: fullTarget},
		"rev-parse " + fullOldUntil:                 {Output: fullOldUntil},
		"rev-parse HEAD":                            {Output: fullOldHead},
		symbolicRef:                                 {Output: "main"},
		rebaseInteractive(fullTarget, fullOldUntil): {Output: ""},
		"add -A":                                 {Output: ""},
		"commit --amend --no-edit --allow-empty": {Output: ""},
		"rebase --continue":                      {Output: ""},
		"rebase --abort":                         {Output: ""},
		"checkout main":                          {Output: ""},
		rebaseOnto(fullOldHead, fullOldUntil, "main"): {Output: ""},
	}
}

func TestOverdubRunsTheFixOnlyOnTheTargetAndReattachesTheRest(t *testing.T) {
	runner := gittest.NewRunner(baseOverdubResponses())
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 0}})

	result, err := NewRepo(runner, commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", []string{"-w", "f.go"})
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if result.NewTarget == "" || result.NewHead == "" {
		t.Errorf("resultado = %+v, os shas novos têm de vir preenchidos", result)
	}

	call := commands.Calls[0]
	if call.Name != "gofmt" || call.Directory != "" {
		t.Errorf("chamada = %+v, o conserto roda direto na árvore de trabalho real, não numa extração", call)
	}

	env := runner.Envs[rebaseInteractive(fullTarget, fullOldUntil)]
	if len(env) != 1 || env[0] != sequenceEditor(fullTarget) {
		t.Errorf("ambiente = %v, queria o editor de sequência apontando pro alvo", env)
	}

	var reattached bool
	for _, recorded := range runner.Calls {
		if recorded == rebaseOnto(fullOldHead, fullOldUntil, "main") {
			reattached = true
		}
	}
	if !reattached {
		t.Errorf("chamadas = %v, o reencaixe do rabo tinha de ter rodado", runner.Calls)
	}
}

func TestOverdubFallsBackToTheHeadSHAWhenDetached(t *testing.T) {
	responses := baseOverdubResponses()
	responses[symbolicRef] = gittest.Response{Err: errNotARepository}
	responses[rebaseOnto(fullOldHead, fullOldUntil, fullOldHead)] = gittest.Response{Output: ""}
	runner := gittest.NewRunner(responses)
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 0}})

	if _, err := NewRepo(runner, commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", nil); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	var reattached bool
	for _, recorded := range runner.Calls {
		if recorded == rebaseOnto(fullOldHead, fullOldUntil, fullOldHead) {
			reattached = true
		}
	}
	if !reattached {
		t.Errorf("chamadas = %v, com HEAD destacado o reencaixe tem de usar o sha como referência", runner.Calls)
	}
}

func TestOverdubPropagatesTheDetachedFallbackFailure(t *testing.T) {
	responses := baseOverdubResponses()
	responses[symbolicRef] = gittest.Response{Err: errNotARepository}
	responses["rev-parse HEAD"] = gittest.Response{Err: errNotARepository}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 0}})

	if _, err := NewRepo(gittest.NewRunner(responses), commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", nil); err == nil {
		t.Fatal("sem symbolic-ref nem rev-parse HEAD não há como saber pra onde reencaixar")
	}
}

func TestOverdubPropagatesTheFixCommandFailure(t *testing.T) {
	runner := gittest.NewRunner(baseOverdubResponses())
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 1, Output: "não formatou"}})

	_, err := NewRepo(runner, commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", []string{"-w", "quebrado.go"})
	if err == nil {
		t.Fatal("comando de conserto que falha tem de abortar, não seguir para o amend")
	}

	for _, recorded := range runner.Calls {
		if recorded == "commit --amend --no-edit --allow-empty" {
			t.Error("o amend não pode rodar depois de um conserto que falhou")
		}
	}
}

// TestOverdubAbortsARebaseStuckMidwayAfterTheFixFails prova o achado do
// relatório de campo: um comando de conserto que falha (executável ausente
// do PATH, no caso real) deixava a rebase parada em HEAD destacado, com
// .git/rebase-merge no lugar, sem nada desfazer isso. Overdub agora chama
// git rebase --abort sozinho, e o erro que sobe continua sendo o do
// conserto — abortar com sucesso não pode esconder a causa original.
func TestOverdubAbortsARebaseStuckMidwayAfterTheFixFails(t *testing.T) {
	runner := gittest.NewRunner(baseOverdubResponses())
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 1, Output: "não formatou"}})

	_, err := NewRepo(runner, commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", []string{"-w", "quebrado.go"})
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "não formatou") {
		t.Errorf("erro = %v, o abort não pode esconder a causa original do conserto", err)
	}
	if strings.Contains(err.Error(), "também falhou") {
		t.Errorf("erro = %v, o abort deu certo aqui, não podia soar como se também tivesse falhado", err)
	}

	var aborted, checkedOut bool
	for _, recorded := range runner.Calls {
		if recorded == "rebase --abort" {
			aborted = true
		}
		if recorded == "checkout main" {
			checkedOut = true
		}
	}
	if !aborted {
		t.Errorf("chamadas = %v, a rebase presa tinha de ser desfeita sozinha", runner.Calls)
	}
	if !checkedOut {
		t.Errorf("chamadas = %v, git rebase -i com um sha cru destaca o HEAD antes do primeiro passo — abortar sozinho não basta, precisa voltar pra ref", runner.Calls)
	}
}

// TestOverdubMentionsWhenTheAbortItselfFails prova o outro lado: se o
// próprio git rebase --abort falhar (ex.: ctx cancelado por Ctrl+C durante
// o conserto), quem chama precisa saber — nunca engolir em silêncio e
// deixar o repositório preso sem dizer nada.
func TestOverdubMentionsWhenTheAbortItselfFails(t *testing.T) {
	responses := baseOverdubResponses()
	responses["rebase --abort"] = gittest.Response{Err: errNotARepository}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 1, Output: "não formatou"}})

	_, err := NewRepo(gittest.NewRunner(responses), commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", nil)
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "não formatou") || !strings.Contains(err.Error(), "também falhou") {
		t.Errorf("erro = %v, queria os dois: a causa original e o aviso de que o abort não resolveu", err)
	}
}

// TestOverdubMentionsWhenTheCheckoutBackAlsoFails cobre o abort que
// funcionou mas não conseguiu devolver a ref original — cenário distinto
// do abort em si falhar, com sua própria mensagem.
func TestOverdubMentionsWhenTheCheckoutBackAlsoFails(t *testing.T) {
	responses := baseOverdubResponses()
	responses["checkout main"] = gittest.Response{Err: errNotARepository}
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 1, Output: "não formatou"}})

	_, err := NewRepo(gittest.NewRunner(responses), commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", nil)
	if err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(err.Error(), "não formatou") {
		t.Errorf("erro = %v, a causa original não pode se perder", err)
	}
	if !strings.Contains(err.Error(), "main") {
		t.Errorf("erro = %v, tem de nomear pra onde não conseguiu voltar", err)
	}
}

func TestOverdubAbortsWhenTheCommandCannotStart(t *testing.T) {
	runner := gittest.NewRunner(baseOverdubResponses())
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found")})

	if _, err := NewRepo(runner, commands).Overdub(t.Context(), fullTarget, fullOldUntil, "inexistente", nil); err == nil {
		t.Fatal("esperava erro, veio nil")
	}

	var aborted bool
	for _, recorded := range runner.Calls {
		if recorded == "rebase --abort" {
			aborted = true
		}
	}
	if !aborted {
		t.Errorf("chamadas = %v, comando de conserto que nem começa também deixa a rebase presa, e também precisa abortar", runner.Calls)
	}
}

func TestOverdubPropagatesTheCommandThatCannotStart(t *testing.T) {
	runner := gittest.NewRunner(baseOverdubResponses())
	commands := exectest.NewRunner(exectest.Response{Err: errors.New("executable file not found")})

	if _, err := NewRepo(runner, commands).Overdub(t.Context(), fullTarget, fullOldUntil, "inexistente", nil); err == nil {
		t.Fatal("comando que nem começa tem de virar erro")
	}
}

func TestOverdubPropagatesEachGitFailure(t *testing.T) {
	base := baseOverdubResponses()

	tests := []struct {
		name    string
		breakOn string
	}{
		{name: "resolucao do alvo", breakOn: "rev-parse " + fullTarget},
		{name: "resolucao do until", breakOn: "rev-parse " + fullOldUntil},
		{name: "rebase interativa", breakOn: rebaseInteractive(fullTarget, fullOldUntil)},
		{name: "add", breakOn: "add -A"},
		{name: "amend", breakOn: "commit --amend --no-edit --allow-empty"},
		{name: "resolucao do alvo emendado ou do fim da rebase", breakOn: "rev-parse HEAD"},
		{name: "continue", breakOn: "rebase --continue"},
		{name: "onto", breakOn: rebaseOnto(fullOldHead, fullOldUntil, "main")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responses := map[string]gittest.Response{}
			for key, value := range base {
				if key == test.breakOn {
					responses[key] = gittest.Response{Err: errNotARepository}
					continue
				}
				responses[key] = value
			}

			commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 0}})
			if _, err := NewRepo(gittest.NewRunner(responses), commands).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", nil); err == nil {
				t.Fatalf("falha em %q tinha de virar erro", test.breakOn)
			}
		})
	}
}

func TestEnsure(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]gittest.Response
		wantError bool
	}{
		{
			name:      "dentro de um repositorio",
			responses: map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Output: "true"}},
		},
		{
			name:      "fora de um repositorio",
			responses: map[string]gittest.Response{"rev-parse --is-inside-work-tree": {Err: errNotARepository}},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewRepo(gittest.NewRunner(test.responses), exectest.NewRunner()).Ensure(t.Context())

			if test.wantError && err == nil {
				t.Fatal("esperava erro, veio nil")
			}
			if !test.wantError && err != nil {
				t.Fatalf("nao esperava erro, veio %v", err)
			}
		})
	}
}

func TestOverdubWithoutUntilDefaultsToHEAD(t *testing.T) {
	responses := baseOverdubResponses()
	responses["rev-parse "+fullOldUntil] = gittest.Response{Output: fullOldUntil}
	// Sem --until, resolvedUntil vira "HEAD" e a rebase interativa fecha em
	// HEAD, não em fullOldUntil — refletido só nesta chamada específica.
	delete(responses, rebaseInteractive(fullTarget, fullOldUntil))
	responses[rebaseInteractive(fullTarget, fullOldHead)] = gittest.Response{Output: ""}
	delete(responses, rebaseOnto(fullOldHead, fullOldUntil, "main"))
	responses[rebaseOnto(fullOldHead, fullOldHead, "main")] = gittest.Response{Output: ""}
	runner := gittest.NewRunner(responses)
	commands := exectest.NewRunner(exectest.Response{Result: exec.Result{Code: 0}})

	if _, err := NewRepo(runner, commands).Overdub(t.Context(), fullTarget, "", "gofmt", nil); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	var usedHead bool
	for _, recorded := range runner.Calls {
		if recorded == rebaseInteractive(fullTarget, fullOldHead) {
			usedHead = true
		}
	}
	if !usedHead {
		t.Errorf("chamadas = %v, sem --until o intervalo tem de fechar em HEAD", runner.Calls)
	}
}

func TestOverdubRefusesAnAbbreviatedResolution(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]gittest.Response
	}{
		{
			name: "sha do alvo",
			responses: map[string]gittest.Response{
				"rev-parse " + fullTarget: {Output: "curto"},
			},
		},
		{
			name: "sha do until",
			responses: map[string]gittest.Response{
				"rev-parse " + fullTarget:   {Output: fullTarget},
				"rev-parse " + fullOldUntil: {Output: "curto"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := gittest.NewRunner(test.responses)
			if _, err := NewRepo(runner, exectest.NewRunner()).Overdub(t.Context(), fullTarget, fullOldUntil, "gofmt", nil); err == nil {
				t.Fatal("rev-parse que não devolve sha completo tem de virar erro, nunca virar parte do GIT_SEQUENCE_EDITOR")
			}
		})
	}
}
