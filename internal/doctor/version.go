package doctor

import (
	"strconv"
	"strings"
)

// minimumGit é a versão mais antiga do git aceita pelo diagnóstico: a primeira
// em que existe o subcomando "branch --show-current", do qual a resolução de
// base depende.
var minimumGit = release{major: 2, minor: 22}

// release é uma versão de git reduzida aos números maior e menor.
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

// parseRelease extrai os números maior e menor do início de text e informa se
// ambos foram lidos. O que vier depois do segundo número é ignorado, como o
// patch e os sufixos que as distribuições anexam.
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

// version extrai o número de versão da saída de um comando "--version".
// Devolve o campo seguinte à palavra "version" na primeira linha; na ausência
// dela, o último campo dessa linha; e a string vazia se não houver campo algum.
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
