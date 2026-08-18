package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

const (
	authorShortHead  = "rev-parse --short HEAD"
	authorLastAuthor = "log -1 --format=%an <%ae>"
	authorAmend      = "commit --amend --no-edit --reset-author"
	authorSymbolic   = "symbolic-ref --short HEAD"
	authorDirtyCheck = "diff HEAD --quiet"
	authorHardReset  = "reset --hard deadbeef"
)

func authorRangeCount(from string, to string) string {
	return "rev-list --count " + from + ".." + to
}

func authorRangeAuthors(from string, to string) string {
	return "log " + from + ".." + to + " --format=%an <%ae>"
}

func authorRebaseExec(base string, untilSHA string) string {
	return "rebase " + base + " " + untilSHA + " --exec git commit --amend --no-edit --reset-author"
}

func authorRebaseOnto(newUntilSHA string, oldUntilSHA string, ref string) string {
	return "rebase --onto " + newUntilSHA + " " + oldUntilSHA + " " + ref
}

func authored() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		authorShortHead:                   {Output: "abc1234"},
		authorLastAuthor:                  {Output: "Real Person <real@real.com>"},
		authorAmend:                       {Output: ""},
	}
}

// authoredRange monta um repositório com uma branch main..HEAD de 3
// commits, dois autores distintos, e nada a preservar depois do HEAD.
func authoredRange() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":                 {Output: "true"},
		authorShortHead:                                   {Output: "abc1234"},
		authorRangeCount("main", "HEAD"):                  {Output: "3"},
		authorRangeAuthors("main", "HEAD"):                {Output: "Real Person <real@real.com>\nOther Person <other@other.com>\n"},
		authorRangeCount("HEAD", "HEAD"):                  {Output: "0"},
		"rev-parse HEAD":                                  {Output: "headsha"},
		authorSymbolic:                                    {Output: "feature"},
		authorRebaseExec("main", "headsha"):               {Output: ""},
		authorRebaseOnto("headsha", "headsha", "feature"): {Output: ""},
	}
}

// authoredCommit monta um repositório onde --commit deadbeef traduz para
// --base deadbeef^ --until deadbeef por baixo: um commit só, com três
// commits depois dele a preservar.
func authoredCommit() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":                      {Output: "true"},
		authorShortHead:                                        {Output: "abc1234"},
		authorRangeCount("deadbeef^", "deadbeef"):              {Output: "1"},
		authorRangeAuthors("deadbeef^", "deadbeef"):            {Output: "Real Person <real@real.com>\n"},
		authorRangeCount("deadbeef", "HEAD"):                   {Output: "3"},
		"rev-parse deadbeef":                                   {Output: "untilsha"},
		authorSymbolic:                                         {Output: "feature"},
		authorRebaseExec("deadbeef^", "untilsha"):              {Output: ""},
		"rev-parse HEAD":                                       {Output: "newuntilsha"},
		authorRebaseOnto("newuntilsha", "untilsha", "feature"): {Output: ""},
	}
}

// authorReset monta um repositório com árvore limpa, onde --reset deadbeef
// descartaria 3 commits e nenhuma mudança não commitada.
func authorReset() map[string]gittest.Response {
	return map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":    {Output: "true"},
		authorShortHead:                      {Output: "abc1234"},
		authorRangeCount("deadbeef", "HEAD"): {Output: "3"},
		authorDirtyCheck:                     {Output: ""},
		authorHardReset:                      {Output: ""},
	}
}

func TestAuthorReset(t *testing.T) {
	result := execute(t, authorReset(), "y\n", "author", "--reset", "deadbeef")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "descartando 3 commits") {
		t.Errorf("saída = %q, queria a contagem de commits descartados", result.stdout)
	}
	if !strings.Contains(result.stdout, "git reset --hard abc1234") {
		t.Errorf("saída = %q, queria a instrução de recuperação", result.stdout)
	}
	if !strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, queria a confirmação de que resetou", result.stdout)
	}

	var reset bool
	for _, call := range result.calls {
		if call == authorHardReset {
			reset = true
		}
	}
	if !reset {
		t.Errorf("chamadas = %v, o reset --hard tinha de ter rodado", result.calls)
	}
}

func TestAuthorResetWarnsAboutADirtyTree(t *testing.T) {
	responses := authorReset()
	responses[authorDirtyCheck] = gittest.Response{Err: &git.ExitError{Code: 1, Message: ""}}

	result := execute(t, responses, "y\n", "author", "--reset", "deadbeef")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "sem deixar rastro nenhum") {
		t.Errorf("saída = %q, queria o aviso de árvore suja", result.stdout)
	}
}

func TestAuthorResetCancelledWithoutConfirmation(t *testing.T) {
	result := execute(t, authorReset(), "\n", "author", "--reset", "deadbeef")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Cancelado, nada foi resetado.") {
		t.Errorf("saída = %q, queria a mensagem de cancelamento", result.stdout)
	}

	for _, call := range result.calls {
		if call == authorHardReset {
			t.Fatalf("sem confirmar não pode resetar, mas rodou %q", call)
		}
	}
}

func TestAuthorResetRefusesToCombineWithRewriteFlags(t *testing.T) {
	tests := [][]string{
		{"author", "--reset", "deadbeef", "--name", "Fake Name"},
		{"author", "--reset", "deadbeef", "--email", "fake@fake.com"},
		{"author", "--reset", "deadbeef", "--base", "main"},
		{"author", "--reset", "deadbeef", "--until", "main"},
		{"author", "--reset", "deadbeef", "--commit", "cafe"},
	}

	for _, args := range tests {
		result := execute(t, authorReset(), "", args...)

		if result.err == nil {
			t.Errorf("%v: --reset combinado com uma flag de reescrita tem de ser recusado", args)
		}
		if len(result.calls) != 0 {
			t.Errorf("%v: chamadas = %v, a validação vem antes de tocar no git", args, result.calls)
		}
	}
}

func TestAuthorResetOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "author", "--reset", "deadbeef")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
}

func TestAuthorResetPropagatesThePlanFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		authorShortHead:                   {Err: errNotARepository},
	}

	result := execute(t, responses, "y\n", "author", "--reset", "deadbeef")

	if result.err == nil {
		t.Fatal("falha ao planejar tem de virar erro antes de perguntar qualquer coisa")
	}
	if strings.Contains(result.stdout, "Confirma?") {
		t.Errorf("saída = %q, não pode perguntar sem saber o que vai resetar", result.stdout)
	}
}

func TestAuthorResetPropagatesTheReadFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(authorReset()), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"author", "--reset", "deadbeef"})

	if command.Execute() == nil {
		t.Fatal("falha na leitura da confirmação tem de virar erro")
	}
}

func TestAuthorResetPropagatesTheResetFailure(t *testing.T) {
	responses := authorReset()
	responses[authorHardReset] = gittest.Response{Err: errNotARepository}

	result := execute(t, responses, "y\n", "author", "--reset", "deadbeef")

	if result.err == nil {
		t.Fatal("falha do reset --hard tem de virar erro")
	}
}

func TestAuthorRewritesASingleCommit(t *testing.T) {
	result := execute(t, authoredCommit(), "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com", "--commit", "deadbeef")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Vai reescrever o commit deadbeef (hoje em Real Person <real@real.com>)") {
		t.Errorf("saída = %q, queria o commit nomeado sem vazar a sintaxe de ^", result.stdout)
	}
	if strings.Contains(result.stdout, "deadbeef^") {
		t.Errorf("saída = %q, a sintaxe interna de ^ não pode aparecer pro usuário", result.stdout)
	}
	if !strings.Contains(result.stdout, "Mais 3 commits depois de deadbeef serão preservados, só reencaixados em cima.") {
		t.Errorf("saída = %q, queria o aviso do rabo preservado", result.stdout)
	}

	var rebased bool
	for _, call := range result.calls {
		if call == authorRebaseExec("deadbeef^", "untilsha") {
			rebased = true
		}
	}
	if !rebased {
		t.Errorf("chamadas = %v, --commit tinha de traduzir para --base <commit>^", result.calls)
	}
}

func TestAuthorRefusesCommitWithBase(t *testing.T) {
	result := execute(t, authored(), "", "author", "--name", "Fake Name", "--email", "fake@fake.com", "--commit", "deadbeef", "--base", "main")

	if result.err == nil {
		t.Fatal("--commit com --base não faz sentido, e tem de ser recusado")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestAuthorRefusesCommitWithUntil(t *testing.T) {
	result := execute(t, authored(), "", "author", "--name", "Fake Name", "--email", "fake@fake.com", "--commit", "deadbeef", "--until", "cafe")

	if result.err == nil {
		t.Fatal("--commit com --until não faz sentido, e tem de ser recusado")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestAuthorRewritesTheLastCommit(t *testing.T) {
	result := execute(t, authored(), "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Real Person <real@real.com>") {
		t.Errorf("saída = %q, queria a autoria atual na prévia", result.stdout)
	}
	if !strings.Contains(result.stdout, "Fake Name <fake@fake.com>") {
		t.Errorf("saída = %q, queria a nova identidade", result.stdout)
	}
	if !strings.Contains(result.stdout, "git reset --hard abc1234") {
		t.Errorf("saída = %q, queria a instrução de recuperação", result.stdout)
	}
	if !strings.Contains(result.stdout, "Pronto.") {
		t.Errorf("saída = %q, queria a confirmação de que reescreveu", result.stdout)
	}

	var rewrote bool
	for _, call := range result.calls {
		if call == authorAmend {
			rewrote = true
		}
	}
	if !rewrote {
		t.Errorf("chamadas = %v, o amend tinha de ter rodado", result.calls)
	}
}

func TestAuthorCancelledWithoutConfirmation(t *testing.T) {
	result := execute(t, authored(), "\n", "author", "--name", "Fake Name", "--email", "fake@fake.com")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "Cancelado, nada foi reescrito.") {
		t.Errorf("saída = %q, queria a mensagem de cancelamento", result.stdout)
	}

	for _, call := range result.calls {
		if call == authorAmend {
			t.Fatalf("sem confirmar não pode reescrever, mas rodou %q", call)
		}
	}
}

func TestAuthorRewritesARange(t *testing.T) {
	result := execute(t, authoredRange(), "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com", "--base", "main")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "3 commits") {
		t.Errorf("saída = %q, queria a contagem da faixa", result.stdout)
	}
	if !strings.Contains(result.stdout, "main..HEAD") {
		t.Errorf("saída = %q, queria o intervalo nomeado", result.stdout)
	}
	if !strings.Contains(result.stdout, "Other Person <other@other.com>, Real Person <real@real.com>") {
		t.Errorf("saída = %q, queria os autores atuais listados, em ordem alfabética", result.stdout)
	}
	if strings.Contains(result.stdout, "preservados") {
		t.Errorf("saída = %q, faixa que vai até o HEAD não tem rabo a preservar", result.stdout)
	}

	var rebased bool
	for _, call := range result.calls {
		if call == authorRebaseExec("main", "headsha") {
			rebased = true
		}
	}
	if !rebased {
		t.Errorf("chamadas = %v, o rebase tinha de ter rodado", result.calls)
	}
}

func TestAuthorRewritesARangeWithUntilPreservesTheTail(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree":                      {Output: "true"},
		authorShortHead:                                        {Output: "abc1234"},
		authorRangeCount("main", "deadbeef"):                   {Output: "2"},
		authorRangeAuthors("main", "deadbeef"):                 {Output: "Real Person <real@real.com>\n"},
		authorRangeCount("deadbeef", "HEAD"):                   {Output: "5"},
		"rev-parse deadbeef":                                   {Output: "untilsha"},
		authorSymbolic:                                         {Output: "feature"},
		authorRebaseExec("main", "untilsha"):                   {Output: ""},
		"rev-parse HEAD":                                       {Output: "newuntilsha"},
		authorRebaseOnto("newuntilsha", "untilsha", "feature"): {Output: ""},
	}

	result := execute(t, responses, "y\n",
		"author", "--name", "Fake Name", "--email", "fake@fake.com", "--base", "main", "--until", "deadbeef")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "2 commits (main..deadbeef)") {
		t.Errorf("saída = %q, queria a contagem e o intervalo fechado em --until", result.stdout)
	}
	if !strings.Contains(result.stdout, "Mais 5 commits depois de deadbeef serão preservados") {
		t.Errorf("saída = %q, queria o aviso do rabo preservado", result.stdout)
	}

	var rebasedOnto bool
	for _, call := range result.calls {
		if call == authorRebaseOnto("newuntilsha", "untilsha", "feature") {
			rebasedOnto = true
		}
	}
	if !rebasedOnto {
		t.Errorf("chamadas = %v, o reencaixe do rabo tinha de ter rodado", result.calls)
	}
}

func TestAuthorRequiresNameAndEmail(t *testing.T) {
	tests := [][]string{
		{"author", "--email", "fake@fake.com"},
		{"author", "--name", "Fake Name"},
		{"author"},
	}

	for _, args := range tests {
		result := execute(t, authored(), "", args...)

		if result.err == nil {
			t.Errorf("%v: --name e --email são obrigatórios, mas não recusou", args)
		}
		if len(result.calls) != 0 {
			t.Errorf("%v: chamadas = %v, a validação vem antes de tocar no git", args, result.calls)
		}
	}
}

func TestAuthorRefusesUntilWithoutBase(t *testing.T) {
	result := execute(t, authored(), "", "author", "--name", "Fake Name", "--email", "fake@fake.com", "--until", "deadbeef")

	if result.err == nil {
		t.Fatal("--until sem --base não faz sentido, e tem de ser recusado")
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a validação vem antes de tocar no git", result.calls)
	}
}

func TestAuthorRefusesAnEmptyRange(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		authorShortHead:                   {Output: "abc1234"},
		authorRangeCount("main", "HEAD"):  {Output: "0"},
	}

	result := execute(t, responses, "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com", "--base", "main")

	if result.err == nil {
		t.Fatal("intervalo vazio tem de virar erro")
	}
}

func TestAuthorOutsideRepository(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Err: errNotARepository},
	}

	result := execute(t, responses, "", "author", "--name", "Fake Name", "--email", "fake@fake.com")

	if result.err == nil {
		t.Fatal("esperava erro, veio nil")
	}
	if !strings.Contains(result.err.Error(), "não é um repositório git") {
		t.Fatalf("erro = %v, queria o do Ensure e não o de outra etapa", result.err)
	}
}

func TestAuthorPropagatesTheRewriteFailure(t *testing.T) {
	responses := authored()
	responses[authorAmend] = gittest.Response{Err: errNotARepository}

	result := execute(t, responses, "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com")

	if result.err == nil {
		t.Fatal("falha do amend tem de virar erro")
	}
}

func TestAuthorNeverConfirmsWithoutAPlan(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --is-inside-work-tree": {Output: "true"},
		authorShortHead:                   {Err: errNotARepository},
	}

	result := execute(t, responses, "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com")

	if result.err == nil {
		t.Fatal("falha ao planejar tem de virar erro antes de perguntar qualquer coisa")
	}
	if strings.Contains(result.stdout, "Confirma?") {
		t.Errorf("saída = %q, não pode perguntar sem saber o que vai reescrever", result.stdout)
	}
}

func TestAuthorPropagatesTheReadFailure(t *testing.T) {
	command := NewRootCommand(gittest.NewRunner(authored()), noCommands(), noWeb(), noFinder(), noNotices)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetIn(brokenReader{})
	command.SetArgs([]string{"author", "--name", "Fake Name", "--email", "fake@fake.com"})

	if command.Execute() == nil {
		t.Fatal("falha na leitura da confirmação tem de virar erro")
	}
}

func TestAuthorTakesNoArguments(t *testing.T) {
	result := execute(t, authored(), "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com", "sobrando")

	if result.err == nil {
		t.Fatal("o author não recebe argumento posicional; argumento a mais tem de virar erro")
	}
}

func TestBlameSomeoneElseIsTheSameCommand(t *testing.T) {
	byName := execute(t, authored(), "y\n", "author", "--name", "Fake Name", "--email", "fake@fake.com")
	byAlias := execute(t, authored(), "y\n", "blame-someone-else", "--name", "Fake Name", "--email", "fake@fake.com")

	if byAlias.err != nil {
		t.Fatalf("o apelido tem de rodar o comando, veio %v", byAlias.err)
	}
	if byAlias.stdout != byName.stdout {
		t.Errorf("apelido = %q, nome = %q; é o mesmo comando", byAlias.stdout, byName.stdout)
	}
}

func TestBlameSomeoneElseStaysOutOfTheRootHelp(t *testing.T) {
	result := execute(t, nil, "", "--help")

	if strings.Contains(result.stdout, "blame-someone-else") {
		t.Errorf("saída = %q; o apelido é um easter egg e não entra na lista de comandos", result.stdout)
	}
	if !strings.Contains(result.stdout, "author") {
		t.Errorf("saída = %q, o nome de verdade continua anunciado", result.stdout)
	}
}
