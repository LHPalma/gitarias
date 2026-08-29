package overdub

import (
	"fmt"
	"os"
	"strings"
)

// SequenceStepCommand é o subcomando oculto do gtr que Overdub usa como
// GIT_SEQUENCE_EDITOR: troca "pick" por "edit" só na linha do commit-alvo
// no todo da rebase interativa. Não depende de sed — ausente por padrão no
// Windows — e não interpola o sha do usuário numa expressão regular de
// shell: o sha chega como argumento comum de linha de comando, já validado
// como hex puro por Overdub antes de virar parte do valor de
// GIT_SEQUENCE_EDITOR. Precisa estar no PATH, mesma exigência que o gh e o
// StripStepCommand já têm.
const SequenceStepCommand = "overdub-sequence-step"

// MarkForEdit troca "pick" por "edit" na linha de path (o arquivo de todo
// da rebase) que começa com "pick " seguido de sha. Erro se nenhuma linha
// bater: sha fora do intervalo reescrito não pode virar uma rebase que
// termina sem nunca ter parado em lugar nenhum.
func MarkForEdit(sha string, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	matched := false
	for index, line := range lines {
		rest, ok := pickRest(line, sha)
		if !ok {
			continue
		}

		lines[index] = "edit " + sha + rest
		matched = true
	}

	if !matched {
		return fmt.Errorf("%s não apareceu como pick no todo da rebase", sha)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

// pickRest devolve o que sobra de line depois de "pick "+sha, só quando sha
// termina ali numa fronteira de palavra — nunca quando é prefixo de um sha
// mais comprido. Sem essa fronteira, um sha mais curto bateria por acidente
// contra outro que só começa igual.
func pickRest(line string, sha string) (string, bool) {
	prefix := "pick " + sha
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}

	rest := line[len(prefix):]
	if rest != "" && rest[0] != ' ' {
		return "", false
	}

	return rest, true
}
