package doctor

import (
	"strconv"
	"strings"
)

// minimumGit é a versão mais antiga do git que o gtr aceita. O critério é o
// branch --show-current, comando mais novo entre os que o gtr chama e do qual
// a resolução de base depende, disponível a partir do git 2.22.
var minimumGit = release{major: 2, minor: 22}

// release é uma versão de git reduzida ao que serve de critério: o número
// maior e o menor.
type release struct {
	major int
	minor int
}

func (version release) String() string {
	return strconv.Itoa(version.major) + "." + strconv.Itoa(version.minor)
}

func (version release) before(other release) bool {
	if version.major != other.major {
		return version.major < other.major
	}

	return version.minor < other.minor
}

// parseRelease extrai o número maior e o menor de uma versão e informa se
// conseguiu. Só esses dois entram na comparação com a mínima, porque o resto
// varia demais para servir de critério: o git do macOS anexa a build da Apple
// e o do Windows anexa o sufixo da distribuição.
func parseRelease(text string) (release, bool) {
	fields := strings.SplitN(text, ".", 3)
	if len(fields) < 2 {
		return release{}, false
	}

	major, err := strconv.Atoi(fields[0])
	if err != nil {
		return release{}, false
	}

	minor, err := strconv.Atoi(fields[1])
	if err != nil {
		return release{}, false
	}

	return release{major: major, minor: minor}, true
}

// version extrai a versão de uma saída de --version, no formato do git e no do
// gh. Procura o campo seguinte à palavra "version" e, não a encontrando, cai no
// último campo da primeira linha.
func version(output string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")

	fields := strings.Fields(line)
	for index, field := range fields {
		if field == "version" && index+1 < len(fields) {
			return fields[index+1]
		}
	}

	if len(fields) == 0 {
		return ""
	}

	return fields[len(fields)-1]
}
