package aitrailers

import "strings"

// Strip devolve a mensagem sem os trailers de autoria de IA, e se algo foi
// de fato removido. rawMessage é o corpo inteiro do commit (%B); rawBlock é
// o bloco de trailers exatamente como aparece no fim de rawMessage —
// %(trailers:only), sem unfold, porque só assim ele é garantidamente sufixo
// literal de rawMessage, byte a byte (medido contra o git antes de assumir
// isso). unfoldedBlock é o mesmo bloco com %(trailers:only,unfold), uma
// linha por trailer, o único jeito confiável de casar cada linha contra as
// assinaturas conhecidas.
//
// Um trailer humano cujo valor original quebrasse em mais de uma linha
// sairia reformatado numa linha só, porque a reconstrução parte do bloco
// desdobrado — perderia a quebra de linha, não o conteúdo. Nenhum dos
// trailers que este pacote reconhece (Co-Authored-By, Claude-Session) quebra
// assim na prática; é o limite conhecido, registrado e não escondido.
func Strip(rawMessage string, rawBlock string, unfoldedBlock string) (string, bool) {
	if rawBlock == "" {
		return rawMessage, false
	}

	var kept []string
	var removed bool
	for _, line := range strings.Split(unfoldedBlock, "\n") {
		if line == "" {
			continue
		}

		key, value, found := strings.Cut(line, ": ")
		if !found {
			kept = append(kept, line)
			continue
		}

		if _, matched := tool(key, value); matched {
			removed = true
			continue
		}

		kept = append(kept, line)
	}

	if !removed {
		return rawMessage, false
	}

	head := strings.TrimRight(strings.TrimSuffix(rawMessage, rawBlock), "\n")

	if len(kept) == 0 {
		return head, true
	}

	return head + "\n\n" + strings.Join(kept, "\n"), true
}
