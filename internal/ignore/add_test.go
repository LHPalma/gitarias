package ignore

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func notCovered(pattern string) map[string]gittest.Response {
	return map[string]gittest.Response{
		"check-ignore -v -- " + pattern: {Err: &git.ExitError{Code: 1}},
	}
}

const rulesNoIndex = rules + " --no-index"

func nothingMatches() map[string]gittest.Response {
	return map[string]gittest.Response{
		expandedListing: {Output: ""},
		"ls-files -z":   {Output: ""},
	}
}

func merge(maps ...map[string]gittest.Response) map[string]gittest.Response {
	merged := map[string]gittest.Response{}
	for _, one := range maps {
		for key, value := range one {
			merged[key] = value
		}
	}
	return merged
}

func TestAddWritesToRepositoryGitignoreByDefault(t *testing.T) {
	root := t.TempDir()
	responses := merge(
		map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}},
		notCovered("build/"),
		nothingMatches(),
	)

	result, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "build/", Repository, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	wantPath := filepath.Join(root, ".gitignore")
	if result.Path != wantPath {
		t.Errorf("path = %q, queria %q", result.Path, wantPath)
	}
	if !result.Written {
		t.Fatal("queria Written")
	}
	if !result.Unmatched {
		t.Error("nada casou com o padrão, o aviso de armadilha 2 tinha de estar marcado")
	}

	content, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("o arquivo tinha de existir: %v", err)
	}
	if string(content) != "build/\n" {
		t.Errorf("conteúdo = %q, queria %q", content, "build/\n")
	}
}

func TestAddWritesToLocalInfoExclude(t *testing.T) {
	common := t.TempDir()
	responses := merge(
		map[string]gittest.Response{"rev-parse --git-common-dir": {Output: common}},
		notCovered("app.log"),
		nothingMatches(),
	)

	result, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Local, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	wantPath := filepath.Join(common, "info", "exclude")
	if result.Path != wantPath {
		t.Errorf("path = %q, queria %q", result.Path, wantPath)
	}
	if _, err := os.ReadFile(wantPath); err != nil {
		t.Errorf("o arquivo tinha de existir, mesmo o diretório info/ não existindo antes: %v", err)
	}
}

func TestAddWritesToTheConfiguredGlobalExcludesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ignore")
	responses := merge(
		map[string]gittest.Response{"config --get --path core.excludesFile": {Output: path}},
		notCovered("*.tmp"),
		nothingMatches(),
	)

	result, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "*.tmp", Global, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if result.Path != path {
		t.Errorf("path = %q, queria o já configurado %q", result.Path, path)
	}
}

func TestAddWithoutGlobalExcludesFileConfiguredRefuses(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	responses := map[string]gittest.Response{
		"config --get --path core.excludesFile": {Err: &git.ExitError{Code: 1}},
	}

	_, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "*.tmp", Global, false)
	if err == nil {
		t.Fatal("sem core.excludesFile configurado e sem --force tinha de recusar")
	}

	fallback := filepath.Join(xdg, "git", "ignore")
	if !strings.Contains(err.Error(), fallback) {
		t.Errorf("erro = %v, queria o caminho sugerido %q", err, fallback)
	}
	if !strings.Contains(err.Error(), "git config --global core.excludesFile") {
		t.Errorf("erro = %v, queria o comando git para resolver", err)
	}
	if _, err := os.Stat(fallback); err == nil {
		t.Error("recusa não pode deixar arquivo nenhum para trás")
	}
}

func TestAddWithForceCreatesTheGlobalExcludesFile(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	fallback := filepath.Join(xdg, "git", "ignore")

	responses := merge(
		map[string]gittest.Response{
			"config --get --path core.excludesFile":         {Err: &git.ExitError{Code: 1}},
			"config --global core.excludesFile " + fallback: {Output: ""},
		},
		notCovered("*.tmp"),
		nothingMatches(),
	)

	runner := gittest.NewRunner(responses)
	result, err := NewRepo(runner).Add(t.Context(), "*.tmp", Global, true)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if result.Path != fallback {
		t.Errorf("path = %q, queria %q", result.Path, fallback)
	}

	configured := false
	for _, call := range runner.Calls {
		if call == "config --global core.excludesFile "+fallback {
			configured = true
		}
	}
	if !configured {
		t.Errorf("chamadas = %v, --force tinha de configurar o core.excludesFile", runner.Calls)
	}

	if _, err := os.ReadFile(fallback); err != nil {
		t.Errorf("o arquivo tinha de ser criado: %v", err)
	}
}

func TestAddSkipsWhenAlreadyCovered(t *testing.T) {
	root := t.TempDir()
	responses := map[string]gittest.Response{
		"rev-parse --show-toplevel":  {Output: root},
		"check-ignore -v -- app.log": {Output: ".gitignore:2:*.log\tapp.log"},
	}

	runner := gittest.NewRunner(responses)
	result, err := NewRepo(runner).Add(t.Context(), "app.log", Repository, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if result.Written {
		t.Fatal("já coberto por outra regra: não podia ter gravado")
	}
	if result.CoveredBy == nil {
		t.Fatal("queria CoveredBy preenchido")
	}
	want := Entry{Source: ".gitignore", Line: 2, Pattern: "*.log", Path: "app.log"}
	if *result.CoveredBy != want {
		t.Errorf("CoveredBy = %+v, queria %+v", *result.CoveredBy, want)
	}

	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err == nil {
		t.Error("nada podia ter sido criado no disco")
	}
	for _, call := range runner.Calls {
		if strings.HasPrefix(call, "ls-files") {
			t.Errorf("chamadas = %v, coberto por outra regra encerra antes de listar candidatos", runner.Calls)
		}
	}
}

func TestAddSkipsWhenTheLineAlreadyExists(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(path, []byte("app.log\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	responses := merge(
		map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}},
		notCovered("app.log"),
	)

	runner := gittest.NewRunner(responses)
	result, err := NewRepo(runner).Add(t.Context(), "app.log", Repository, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if result.Written {
		t.Fatal("linha idêntica já existia: não podia ter gravado de novo")
	}
	if !result.Duplicate {
		t.Error("queria Duplicate marcado")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("o arquivo tinha de continuar existindo: %v", err)
	}
	if string(content) != "app.log\n" {
		t.Errorf("conteúdo = %q, não podia ter mudado", content)
	}
}

func TestAddWarnsAboutAlreadyTrackedPaths(t *testing.T) {
	root := t.TempDir()
	responses := merge(
		map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}},
		notCovered("app.log"),
		map[string]gittest.Response{
			expandedListing: {Output: ""},
			"ls-files -z":   {Output: "app.log\x00"},
			rulesNoIndex:    {Output: ".gitignore\x001\x00app.log\x00app.log\x00"},
		},
	)

	result, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !result.Written {
		t.Fatal("queria Written")
	}
	if want := []string{"app.log"}; !equalStrings(result.AlreadyTracked, want) {
		t.Errorf("AlreadyTracked = %v, queria %v", result.AlreadyTracked, want)
	}
	if result.Unmatched {
		t.Error("a regra pegou um caminho, Unmatched não podia estar marcado")
	}
}

func TestAddTreatsNoRealMatchInTheCombinedCheckAsUnmatched(t *testing.T) {
	root := t.TempDir()
	responses := merge(
		map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}},
		notCovered("app.log"),
		map[string]gittest.Response{
			expandedListing: {Output: ""},
			"ls-files -z":   {Output: "outrofile\x00"},
			rulesNoIndex:    {Err: &git.ExitError{Code: 1}},
		},
	)

	result, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if !result.Unmatched {
		t.Error("código 1 do check-ignore combinado é resposta, não falha, e não pegou nada: Unmatched tinha de estar marcado")
	}
}

func TestAddIgnoresEntriesOfAnotherPatternInTheCombinedCheck(t *testing.T) {
	root := t.TempDir()
	responses := merge(
		map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}},
		notCovered("app.log"),
		map[string]gittest.Response{
			expandedListing: {Output: "build/\x00"},
			"ls-files -z":   {Output: ""},
			rulesNoIndex: {Output: "" +
				".gitignore\x001\x00build/\x00build/\x00" +
				".gitignore\x002\x00app.log\x00app.log\x00"},
		},
	)

	result, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if result.Unmatched {
		t.Error("app.log casou de verdade, Unmatched não podia estar marcado")
	}
	if len(result.AlreadyTracked) != 0 {
		t.Errorf("AlreadyTracked = %v, nenhum dos dois caminhos está rastreado", result.AlreadyTracked)
	}
}

func TestAddEscapesTheStoredPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{pattern: "#comment", want: `\#comment`},
		{pattern: "!keep", want: `\!keep`},
		{pattern: "build ", want: `build\ `},
		{pattern: "build   ", want: `build\ \ \ `},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			root := t.TempDir()
			responses := merge(
				map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}},
				notCovered(test.pattern),
				nothingMatches(),
			)

			result, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), test.pattern, Repository, false)
			if err != nil {
				t.Fatalf("não esperava erro, veio %v", err)
			}
			if result.Pattern != test.want {
				t.Errorf("pattern gravado = %q, queria %q", result.Pattern, test.want)
			}

			content, err := os.ReadFile(filepath.Join(root, ".gitignore"))
			if err != nil {
				t.Fatalf("o arquivo tinha de existir: %v", err)
			}
			if string(content) != test.want+"\n" {
				t.Errorf("conteúdo = %q, queria %q", content, test.want+"\n")
			}
		})
	}
}

func TestAddAppendsAfterEnsuringANewline(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(path, []byte("node_modules/"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	responses := merge(
		map[string]gittest.Response{"rev-parse --show-toplevel": {Output: root}},
		notCovered("app.log"),
		nothingMatches(),
	)

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("erro lendo: %v", err)
	}
	if string(content) != "node_modules/\napp.log\n" {
		t.Errorf("conteúdo = %q, o append sem quebra de linha antes corromperia a regra anterior", content)
	}
}

func TestAddPropagatesDestinationFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --show-toplevel": {Err: errNotARepository},
	}

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false); err == nil {
		t.Fatal("falha ao resolver a raiz tem de virar erro")
	}
}

func TestAddPropagatesLocalDestinationFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"rev-parse --git-common-dir": {Err: errNotARepository},
	}

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Local, false); err == nil {
		t.Fatal("falha ao resolver o git-common-dir tem de virar erro")
	}
}

func TestAddPropagatesARealCheckIgnoreFailure(t *testing.T) {
	root := t.TempDir()
	responses := map[string]gittest.Response{
		"rev-parse --show-toplevel":  {Output: root},
		"check-ignore -v -- app.log": {Err: &git.ExitError{Code: 128, Message: "fatal: not a git repository"}},
	}

	_, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false)
	if err == nil {
		t.Fatal("código 128 é falha de verdade e não pode virar \"não coberto\"")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("erro = %v, queria a mensagem do git — RN-07", err)
	}
}

func TestAddPropagatesAnUnreadableCoveringLine(t *testing.T) {
	root := t.TempDir()
	responses := map[string]gittest.Response{
		"rev-parse --show-toplevel":  {Output: root},
		"check-ignore -v -- app.log": {Output: "isso não é o formato esperado"},
	}

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false); err == nil {
		t.Fatal("saída fora do formato origem:linha:padrão\\tcaminho tem de virar erro, não \"não coberto\"")
	}
}

func TestAddPropagatesACoveringLineWithTheWrongNumberOfFields(t *testing.T) {
	root := t.TempDir()
	responses := map[string]gittest.Response{
		"rev-parse --show-toplevel":  {Output: root},
		"check-ignore -v -- app.log": {Output: "semdoispontos\tapp.log"},
	}

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false); err == nil {
		t.Fatal("sem origem:linha:padrão antes da tabulação tem de virar erro, não \"não coberto\"")
	}
}

func TestAddPropagatesACoveringLineWithAnUnreadableLineNumber(t *testing.T) {
	root := t.TempDir()
	responses := map[string]gittest.Response{
		"rev-parse --show-toplevel":  {Output: root},
		"check-ignore -v -- app.log": {Output: ".gitignore:não é número:*.log\tapp.log"},
	}

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false); err == nil {
		t.Fatal("linha ilegível tem de virar erro, não \"não coberto\"")
	}
}

func TestGlobalDestinationPropagatesAConfigSettingFailure(t *testing.T) {
	responses := map[string]gittest.Response{
		"config --get --path core.excludesFile": {Err: &git.ExitError{Code: 128, Message: "fatal: bad config"}},
	}

	_, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Global, false)
	if err == nil {
		t.Fatal("código diferente de 1 é falha de verdade e não pode virar \"não configurado\"")
	}
	if !strings.Contains(err.Error(), "bad config") {
		t.Errorf("erro = %v, queria a mensagem do git — RN-07", err)
	}
}

func TestDefaultGlobalExcludesPathFallsBackToHomeWhenXDGIsUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	path, err := defaultGlobalExcludesPath()
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	want := filepath.Join(home, ".config", "git", "ignore")
	if path != want {
		t.Errorf("caminho = %q, queria %q", path, want)
	}
}

func TestDefaultGlobalExcludesPathPropagatesTheHomeFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	if _, err := defaultGlobalExcludesPath(); err == nil {
		t.Fatal("sem XDG_CONFIG_HOME nem HOME não há como saber o caminho, tinha de falhar")
	}
}

func TestAddPropagatesTheIgnorableCandidatesFailure(t *testing.T) {
	root := t.TempDir()
	responses := merge(
		map[string]gittest.Response{
			"rev-parse --show-toplevel": {Output: root},
			expandedListing:             {Err: errNotARepository},
		},
		notCovered("app.log"),
	)

	_, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false)
	if err == nil {
		t.Fatal("gravou, mas a checagem pós-gravação falhou; tem de virar erro mesmo assim")
	}
	if !strings.Contains(err.Error(), "gravado em") {
		t.Errorf("erro = %v, quem lê precisa saber que a gravação já aconteceu", err)
	}
}

func TestAddPropagatesTheTrackedFilesFailure(t *testing.T) {
	root := t.TempDir()
	responses := merge(
		map[string]gittest.Response{
			"rev-parse --show-toplevel": {Output: root},
			expandedListing:             {Output: ""},
			"ls-files -z":               {Err: errNotARepository},
		},
		notCovered("app.log"),
	)

	_, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false)
	if err == nil {
		t.Fatal("gravou, mas listar os rastreados falhou; tem de virar erro mesmo assim")
	}
	if !strings.Contains(err.Error(), "gravado em") {
		t.Errorf("erro = %v, quem lê precisa saber que a gravação já aconteceu", err)
	}
}

func TestAddPropagatesTheCombinedCheckIgnoreFailure(t *testing.T) {
	root := t.TempDir()
	responses := merge(
		map[string]gittest.Response{
			"rev-parse --show-toplevel": {Output: root},
			expandedListing:             {Output: "outrofile\x00"},
			"ls-files -z":               {Output: ""},
			rulesNoIndex:                {Err: &git.ExitError{Code: 128, Message: "fatal: quebrado"}},
		},
		notCovered("app.log"),
	)

	_, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false)
	if err == nil {
		t.Fatal("gravou, mas a checagem combinada de código 128 é falha de verdade")
	}
	if !strings.Contains(err.Error(), "gravado em") {
		t.Errorf("erro = %v, quem lê precisa saber que a gravação já aconteceu", err)
	}
}

func TestAddPropagatesTheReadFailure(t *testing.T) {
	notADirectory := t.TempDir()
	rootAsAFile := filepath.Join(notADirectory, "root")
	if err := os.WriteFile(rootAsAFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	responses := merge(
		map[string]gittest.Response{"rev-parse --show-toplevel": {Output: rootAsAFile}},
		notCovered("app.log"),
	)

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Repository, false); err == nil {
		t.Fatal("raiz que não é diretório: ler o destino tem de falhar, não fingir arquivo ausente")
	}
}

func TestReadLinesPropagatesAReadFailureThatIsNotAbsence(t *testing.T) {
	directory := t.TempDir()

	if _, err := readLines(directory); err == nil {
		t.Fatal("caminho que é diretório não é \"arquivo ausente\", tem de virar erro")
	}
}

func TestAppendLinePropagatesTheMkdirFailure(t *testing.T) {
	occupied := filepath.Join(t.TempDir(), "arquivo")
	if err := os.WriteFile(occupied, nil, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := appendLine(filepath.Join(occupied, "sub", ".gitignore"), "app.log"); err == nil {
		t.Fatal("diretório que não dá para criar tem de virar erro")
	}
}

func TestAppendLinePropagatesTheOpenFailure(t *testing.T) {
	path := t.TempDir()

	if err := appendLine(path, "app.log"); err == nil {
		t.Fatal("caminho que já é diretório não dá para abrir como arquivo, tem de virar erro")
	}
}

func TestAddPropagatesTheFallbackPathFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	responses := map[string]gittest.Response{
		"config --get --path core.excludesFile": {Err: &git.ExitError{Code: 1}},
	}

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Global, false); err == nil {
		t.Fatal("sem XDG_CONFIG_HOME nem HOME não há caminho de fallback nenhum para sugerir, tinha de falhar")
	}
}

func TestAddWithForcePropagatesTheConfigWriteFailure(t *testing.T) {
	xdg := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	fallback := filepath.Join(xdg, "git", "ignore")

	responses := map[string]gittest.Response{
		"config --get --path core.excludesFile":         {Err: &git.ExitError{Code: 1}},
		"config --global core.excludesFile " + fallback: {Err: errNotARepository},
	}

	if _, err := NewRepo(gittest.NewRunner(responses)).Add(t.Context(), "app.log", Global, true); err == nil {
		t.Fatal("falha ao configurar o core.excludesFile tem de virar erro")
	}
}

type brokenStat struct{}

func (brokenStat) Stat() (os.FileInfo, error)        { return nil, errors.New("stat indisponível") }
func (brokenStat) ReadAt([]byte, int64) (int, error) { return 0, nil }

type fakeInfo struct{ size int64 }

func (info fakeInfo) Name() string       { return "" }
func (info fakeInfo) Size() int64        { return info.size }
func (info fakeInfo) Mode() fs.FileMode  { return 0 }
func (info fakeInfo) ModTime() time.Time { return time.Time{} }
func (info fakeInfo) IsDir() bool        { return false }
func (info fakeInfo) Sys() any           { return nil }

type brokenReadAt struct{ size int64 }

func (broken brokenReadAt) Stat() (os.FileInfo, error) { return fakeInfo{size: broken.size}, nil }
func (brokenReadAt) ReadAt([]byte, int64) (int, error) { return 0, errors.New("leitura indisponível") }

func TestEndsWithoutNewlinePropagatesTheStatFailure(t *testing.T) {
	if _, err := endsWithoutNewline(brokenStat{}); err == nil {
		t.Fatal("stat indisponível tem de virar erro, não \"sem quebra de linha\"")
	}
}

func TestEndsWithoutNewlinePropagatesTheReadAtFailure(t *testing.T) {
	if _, err := endsWithoutNewline(brokenReadAt{size: 3}); err == nil {
		t.Fatal("leitura indisponível tem de virar erro, não \"sem quebra de linha\"")
	}
}

func equalStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
