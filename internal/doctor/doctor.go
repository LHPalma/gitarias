package doctor

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/git"
)

const (
	gitHomepage = "https://git-scm.com"
	ghHomepage  = "https://cli.github.com"
)

type Doctor struct {
	runner   git.Runner
	commands exec.Runner
}

func New(runner git.Runner, commands exec.Runner) *Doctor {
	return &Doctor{runner: runner, commands: commands}
}

func (doctor *Doctor) Diagnose(ctx context.Context) []Check {
	repository := doctor.repository(ctx)

	return []Check{
		doctor.git(ctx),
		doctor.scratch(),
		repository,
		doctor.tree(ctx, repository),
		doctor.identity(ctx),
		doctor.base(ctx, repository),
		doctor.gh(ctx),
	}
}

// scratch relata se o diretório de temporários aceita escrita. A verificação é
// criar um diretório lá e apagá-lo em seguida, e não inspecionar permissões:
// montagem somente leitura, disco cheio e ACL negam a escrita sem que o modo do
// arquivo mude.
func (doctor *Doctor) scratch() Check {
	workspace, err := os.MkdirTemp("", "gtr-doctor-")
	if err != nil {
		return Check{
			Name:   "temporário",
			State:  Warning,
			Detail: os.TempDir() + ": " + reason(err),
			Hint:   "só o gtr commits check precisa dele, para materializar a árvore de cada commit; aponte a variável TMPDIR para um diretório gravável e com espaço",
		}
	}
	defer os.RemoveAll(workspace)

	return Check{Name: "temporário", State: Ok}
}

// reason devolve o motivo que o sistema operacional deu, sem o caminho que a
// stdlib prefixa: o caminho já é dito à parte, e o sufixo aleatório do nome do
// diretório só atrapalha a leitura.
func reason(err error) string {
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		return pathError.Err.Error()
	}

	return err.Error()
}

func (doctor *Doctor) git(ctx context.Context) Check {
	result, err := doctor.commands.Run(ctx, "", "git", "--version")
	if err != nil {
		return Check{
			Name:   "git",
			State:  Failure,
			Detail: "não encontrado no PATH",
			Hint:   "o gtr orquestra o git da máquina e não funciona sem ele; instale em " + gitHomepage,
		}
	}

	if !result.Passed() {
		return Check{
			Name:   "git",
			State:  Failure,
			Detail: "está no PATH mas não roda",
			Hint:   strings.TrimSpace(result.Output),
		}
	}

	reported := version(result.Output)

	running, readable := parseRelease(reported)
	switch {
	case !readable:
		return Check{
			Name:   "git",
			State:  Warning,
			Detail: "versão ilegível",
			Hint:   "o gtr precisa do git " + minimumGit.String() + " ou mais novo, e daqui não dá para conferir qual está instalado",
		}
	case running.before(minimumGit):
		return Check{
			Name:   "git",
			State:  Failure,
			Detail: reported + ", abaixo da mínima " + minimumGit.String(),
			Hint:   "o gtr usa git branch --show-current, que só existe a partir do " + minimumGit.String() + "; atualize em " + gitHomepage,
		}
	}

	return Check{Name: "git", State: Ok, Detail: reported}
}

func (doctor *Doctor) repository(ctx context.Context) Check {
	if err := git.EnsureRepo(ctx, doctor.runner); err != nil {
		return Check{Name: "repositório", State: Skipped, Detail: "o diretório atual não é um repositório git"}
	}

	return Check{Name: "repositório", State: Ok}
}

// tree relata se há uma operação de git no meio do caminho. A leitura é pelas
// refs que o git mantém enquanto a operação dura, e não pela saída de git
// status, que é prosa e muda entre versões. Devolve pulada fora de um
// repositório.
func (doctor *Doctor) tree(ctx context.Context, repository Check) Check {
	if !repository.Passed() {
		return Check{Name: "árvore", State: Skipped, Detail: "depende de estar num repositório"}
	}

	for _, pending := range operations {
		if _, err := doctor.runner.Run(ctx, "rev-parse", "--verify", "--quiet", pending.ref); err == nil {
			return interrupted(pending.name)
		}
	}

	if doctor.applying(ctx) {
		return interrupted("am")
	}

	return Check{Name: "árvore", State: Ok, Detail: "sem operação em curso"}
}

// applying relata o git am, a operação que não deixa ref: ela aplica texto de
// patch, e não um commit a que uma ref pudesse apontar. O que fica é o
// diretório rebase-apply com um arquivo applying dentro.
//
// O diretório sozinho não serve de sinal, porque o backend --apply do rebase
// usa o mesmo e deixa lá um arquivo onto. O applying separa os dois, e o rebase
// já foi reconhecido pela ref antes de chegar aqui.
//
// O caminho vem do próprio git em vez de ser montado como .git/rebase-apply,
// que estaria errado dentro de um worktree e sob GIT_DIR. O git o devolve
// relativo ao diretório corrente, que é de onde o os.Stat também parte.
func (doctor *Doctor) applying(ctx context.Context) bool {
	directory, err := doctor.runner.Run(ctx, "rev-parse", "--git-path", "rebase-apply")
	if err != nil {
		return false
	}

	_, err = os.Stat(filepath.Join(directory, "applying"))

	return err == nil
}

// interrupted monta o aviso de uma operação parada. As quatro do git que têm
// ref e o am aceitam as mesmas duas saídas, --continue e --abort.
func interrupted(name string) Check {
	return Check{
		Name:   "árvore",
		State:  Warning,
		Detail: name + " em andamento",
		Hint: "conclua com git " + name + " --continue ou desfaça com git " + name +
			" --abort; no meio de uma operação o HEAD não é o que parece, e o gtr branches decide o que deletar a partir dele",
	}
}

// identity relata a identidade de git em vigor, sem julgá-la: a precedência
// entre as configurações local e global é resolvida pelo próprio git. Devolve
// aviso quando user.name ou user.email está ausente ou vazio, nomeando as
// chaves que faltam.
func (doctor *Doctor) identity(ctx context.Context) Check {
	name := doctor.setting(ctx, "user.name")
	email := doctor.setting(ctx, "user.email")

	var missing []string
	if name == "" {
		missing = append(missing, "user.name")
	}
	if email == "" {
		missing = append(missing, "user.email")
	}

	if len(missing) > 0 {
		return Check{
			Name:   "identidade",
			State:  Warning,
			Detail: "falta " + strings.Join(missing, " e "),
			Hint:   "o gtr branches monta um commit de sondagem para achar branch squashada, e o git recusa montá-lo sem autor; configure com git config user.name e git config user.email",
		}
	}

	return Check{Name: "identidade", State: Ok, Detail: name + " <" + email + ">"}
}

func (doctor *Doctor) setting(ctx context.Context, key string) string {
	value, err := doctor.runner.Run(ctx, "config", "--get", key)
	if err != nil {
		return ""
	}

	return value
}

// base relata a branch base e a origem dela. Devolve aviso quando o nome foi
// obtido por tentativa entre main e master, e não declarado por
// refs/remotes/origin/HEAD: só nesse caso o resultado pode estar errado sem
// que a resolução tenha falhado. Devolve pulada fora de um repositório.
func (doctor *Doctor) base(ctx context.Context, repository Check) Check {
	if !repository.Passed() {
		return Check{Name: "base", State: Skipped, Detail: "depende de estar num repositório"}
	}

	resolved, err := branch.NewRepo(doctor.runner).ResolveBase(ctx, "")
	if err != nil {
		return Check{
			Name:   "base",
			State:  Warning,
			Detail: "não determinável aqui",
			Hint:   "só o gtr branches precisa dela; informe com --base <branch> ou crie main ou master",
		}
	}

	if resolved.Source == branch.BaseFromLocal {
		return Check{
			Name:   "base",
			State:  Warning,
			Detail: resolved.Name + ", adivinhada pelo nome",
			Hint:   "o remoto não disse qual é a padrão dele, então o gtr chutou entre main e master; traga o origin/HEAD com git fetch origin, ou informe a base com --base <branch>",
		}
	}

	return Check{Name: "base", State: Ok, Detail: resolved.Name + ", declarada pelo remoto"}
}

func (doctor *Doctor) gh(ctx context.Context) Check {
	result, err := doctor.commands.Run(ctx, "", "gh", "--version")
	if err != nil {
		return Check{
			Name:   "gh",
			State:  Warning,
			Detail: "não encontrado no PATH",
			Hint:   "só é preciso para os comandos de PR; o resto do gtr funciona sem ele. Instale em " + ghHomepage,
		}
	}

	if !result.Passed() {
		return Check{
			Name:   "gh",
			State:  Warning,
			Detail: "está no PATH mas não roda",
			Hint:   strings.TrimSpace(result.Output),
		}
	}

	return Check{Name: "gh", State: Ok, Detail: version(result.Output)}
}
