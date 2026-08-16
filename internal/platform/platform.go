package platform

import "strings"

// Homepage é onde cada ferramenta se instala quando não há gerenciador de
// pacotes reconhecido, ou quando ela não está no repositório dele.
var Homepage = map[string]string{
	"git": "https://git-scm.com",
	"gh":  "https://cli.github.com",
}

// Platform é a máquina de quem roda, reduzida ao que decide um comando de
// instalação: o sistema, a distribuição quando há, e o gerenciador achado.
type Platform struct {
	OS     string
	Distro string

	found    manager
	elevate  bool
	detected bool
}

// Detect procura, na ordem da tabela, o primeiro gerenciador de pacotes que
// esteja no PATH.
func Detect(finder Finder) Platform {
	found := Platform{OS: finder.OS(), Distro: distro(finder.Release())}

	for _, candidate := range managers[found.OS] {
		if !finder.Look(candidate.name) {
			continue
		}

		found.found = candidate
		found.detected = true
		found.elevate = candidate.elevates && !finder.Root()

		break
	}

	return found
}

// Manager é o nome do gerenciador achado, ou string vazia quando não houve.
func (platform Platform) Manager() string {
	return platform.found.name
}

// Describe nomeia a máquina para quem lê — a distribuição quando ela existe,
// senão o sistema operacional.
func (platform Platform) Describe() string {
	if platform.Distro != "" {
		return platform.Distro
	}

	return platform.OS
}

// Install devolve os comandos que instalam uma ferramenta nesta máquina, na
// ordem em que devem ser digitados. Devolve nada quando não há gerenciador
// reconhecido ou quando a ferramenta é desconhecida — nesses casos o chamador
// cai no Homepage, porque um comando inventado é pior que nenhum.
func (platform Platform) Install(tool string) []string {
	name, known := platform.found.packages[tool]
	if !platform.detected || !known {
		return nil
	}

	var commands []string
	if len(platform.found.refresh) > 0 {
		commands = append(commands, platform.command(platform.found.refresh))
	}

	return append(commands, platform.command(append(platform.found.install, name)))
}

func (platform Platform) command(parts []string) string {
	line := strings.Join(parts, " ")
	if platform.elevate {
		return "sudo " + line
	}

	return line
}

// distro lê o ID e o VERSION_ID do /etc/os-release, que é o formato que as
// distribuições de Linux publicam. Valor entre aspas é desembrulhado.
func distro(release string) string {
	fields := map[string]string{}
	for _, line := range strings.Split(release, "\n") {
		key, value, found := strings.Cut(line, "=")
		if found {
			fields[key] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}

	name := fields["ID"]
	if name == "" {
		return ""
	}

	if version := fields["VERSION_ID"]; version != "" {
		return name + " " + version
	}

	return name
}
