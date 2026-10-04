package cmd

import (
	"slices"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	redateDate     = "2026-10-03T14:00:00"
	redateAmend    = "commit --amend --no-edit --allow-empty --date=" + redateDate
	redateHeadLog  = "log -1 --format=%h%x00%s%x00%aI"
	redateShort    = "rev-parse --short HEAD"
	redateInsideOK = "rev-parse --is-inside-work-tree"
)

func redateRangeResponses() map[string]gittest.Response {
	window := "log --format=%H --since=2026-09-04 00:00:00"

	return map[string]gittest.Response{
		redateInsideOK:                   {Output: "true"},
		redateShort:                      {Output: "e5e5e5e"},
		window:                           {Output: "newest\noldest"},
		"rev-list --parents -n 1 oldest": {Output: "oldest parent"},
		"log --format=%h%x00%s%x00%aI oldest^..newest":                    {Output: "e5e5e5e\x00c5\x002026-09-07T10:00:00+00:00\nc3c3c3c\x00c3\x002026-09-04T10:00:00+00:00"},
		"rev-list --count newest..HEAD":                                   {Output: "0"},
		"symbolic-ref --short HEAD":                                       {Output: "main"},
		"rebase --rebase-merges oldest^ newest --exec git " + redateAmend: {Output: ""},
		"rev-parse HEAD":                                                  {Output: "rewritten"},
		"rebase --onto rewritten newest main":                             {Output: ""},
	}
}

func TestRedateTheHeadAsksAndAmends(t *testing.T) {
	responses := map[string]gittest.Response{
		redateInsideOK: {Output: "true"},
		redateShort:    {Output: "abc1234"},
		redateHeadLog:  {Output: "abc1234\x00feat: algo\x002026-09-07T10:00:00+00:00"},
		redateAmend:    {Output: ""},
	}

	result := execute(t, responses, "y\n", "redate", "--date", redateDate)

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	for _, want := range []string{"1 commit", redateDate, "abc1234", "feat: algo", "2026-09-07T10:00:00", "Recuperável com: git reset --hard abc1234", "Pronto."} {
		if !strings.Contains(result.stdout, want) {
			t.Errorf("saída = %q, queria %q", result.stdout, want)
		}
	}
	if !slices.Contains(result.calls, redateAmend) {
		t.Errorf("chamadas = %v, o amend tinha de ter rodado", result.calls)
	}
}

func TestRedateCancelledLeavesNothingRewritten(t *testing.T) {
	responses := map[string]gittest.Response{
		redateInsideOK: {Output: "true"},
		redateShort:    {Output: "abc1234"},
		redateHeadLog:  {Output: "abc1234\x00feat: algo\x002026-09-07T10:00:00+00:00"},
	}

	for _, answer := range []string{"n\n", "\n", ""} {
		result := execute(t, responses, answer, "redate", "--date", redateDate)

		if result.err != nil {
			t.Fatalf("resposta %q: não esperava erro, veio %v", answer, result.err)
		}
		if !strings.Contains(result.stdout, "Cancelado, nada foi reescrito.") {
			t.Errorf("resposta %q: saída = %q, queria o aviso de cancelamento", answer, result.stdout)
		}
		if slices.Contains(result.calls, redateAmend) {
			t.Errorf("resposta %q: o amend rodou apesar do cancelamento", answer)
		}
	}
}

func TestRedateSinceAloneGoesUpToToday(t *testing.T) {
	result := execute(t, redateRangeResponses(), "y\n", "redate", "--date", redateDate, "--since", "2026-09-04")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "2 commits") || !strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, queria os 2 commits do período e o Pronto", result.stdout)
	}
	if !slices.Contains(result.calls, "log --format=%H --since=2026-09-04 00:00:00") {
		t.Errorf("chamadas = %v, o período devia ter só --since, sem teto", result.calls)
	}
	for _, call := range result.calls {
		if strings.Contains(call, "--until") {
			t.Errorf("chamada %q: --until não foi pedida", call)
		}
	}
}

func TestRedateNothingInThePeriod(t *testing.T) {
	responses := map[string]gittest.Response{
		redateInsideOK: {Output: "true"},
		redateShort:    {Output: "abc1234"},
		"log --format=%H --since=2030-01-01 00:00:00": {Output: ""},
	}

	result := execute(t, responses, "y\n", "redate", "--date", redateDate, "--since", "2030-01-01")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Nenhum commit no período") {
		t.Errorf("saída = %q, queria o aviso de período vazio", result.stdout)
	}
}

func TestRedateRefusesAMergeAfterThePeriod(t *testing.T) {
	responses := redateRangeResponses()
	delete(responses, "log --format=%H --since=2026-09-04 00:00:00")
	responses["log --format=%H --until=2026-09-05 23:59:59"] = gittest.Response{Output: "newest\noldest"}
	responses["rev-list --count newest..HEAD"] = gittest.Response{Output: "2"}
	responses["rev-list --merges --count newest..HEAD"] = gittest.Response{Output: "1"}

	result := execute(t, responses, "y\n", "redate", "--date", redateDate, "--until", "2026-09-05")

	if result.err == nil || !strings.Contains(result.err.Error(), "merge") {
		t.Fatalf("erro = %v, queria a recusa por merge na cauda", result.err)
	}
	if strings.Contains(result.stdout, "Confirma?") {
		t.Errorf("saída = %q, a recusa vem antes de perguntar", result.stdout)
	}
}

func TestRedateRequiresTheDate(t *testing.T) {
	result := execute(t, map[string]gittest.Response{}, "", "redate")

	if result.err == nil || !strings.Contains(result.err.Error(), "--date") {
		t.Fatalf("erro = %v, queria a exigência de --date", result.err)
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestRedateRefusesInvalidInput(t *testing.T) {
	cases := [][]string{
		{"redate", "--date", "2026-10-03"},
		{"redate", "--date", "2026-10-03T14:00:00; rm -rf /"},
		{"redate", "--date", redateDate, "--since", "04-09-2026"},
		{"redate", "--date", redateDate, "--until", "ontem"},
	}

	for _, args := range cases {
		result := execute(t, map[string]gittest.Response{}, "y\n", args...)

		if result.err == nil {
			t.Errorf("%v: aceitou, queria erro", args)
		}
		if len(result.calls) != 0 {
			t.Errorf("%v: chamadas = %v, a validação vem antes de tocar no git", args, result.calls)
		}
	}
}

func TestRedateOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{redateInsideOK: {Err: errNotARepository}}

	result := execute(t, responses, "y\n", "redate", "--date", redateDate)

	if result.err == nil {
		t.Fatal("fora de repositório tem de virar erro")
	}
}

func TestRedatePropagatesTheRewriteFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		redateInsideOK: {Output: "true"},
		redateShort:    {Output: "abc1234"},
		redateHeadLog:  {Output: "abc1234\x00feat: algo\x002026-09-07T10:00:00+00:00"},
		redateAmend:    {Err: errNotARepository},
	}

	result := execute(t, responses, "y\n", "redate", "--date", redateDate)

	if result.err == nil {
		t.Fatal("falha do amend tem de virar erro")
	}
	if strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, não pode dizer Pronto depois de falhar", result.stdout)
	}
}

func TestRedateTakesNoArguments(t *testing.T) {
	result := execute(t, map[string]gittest.Response{}, "", "redate", "--date", redateDate, "extra")

	if result.err == nil {
		t.Fatal("argumento solto tem de virar erro")
	}
}

func TestReissueIsTheSameCommand(t *testing.T) {
	responses := func() map[string]gittest.Response {
		return map[string]gittest.Response{
			redateInsideOK: {Output: "true"},
			redateShort:    {Output: "abc1234"},
			redateHeadLog:  {Output: "abc1234\x00feat: algo\x002026-09-07T10:00:00+00:00"},
			redateAmend:    {Output: ""},
		}
	}

	byName := execute(t, responses(), "y\n", "redate", "--date", redateDate)
	byAlias := execute(t, responses(), "y\n", "reissue", "--date", redateDate)

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestReissueStaysOutOfTheRootHelp(t *testing.T) {
	result := execute(t, nil, "", "--help")

	if strings.Contains(result.stdout, "reissue") {
		t.Errorf("saída = %q; o apelido é um easter egg e não entra na lista de comandos", result.stdout)
	}
	if !strings.Contains(result.stdout, "redate") {
		t.Errorf("saída = %q, o nome de verdade continua anunciado", result.stdout)
	}
}
