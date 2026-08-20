package changelog

import "regexp"

// Entry é um commit já classificado pelo Conventional Commits: tipo, escopo
// opcional, se veio marcado como mudança incompatível e o assunto sem o
// prefixo. Um commit que não bate na assinatura vira Miscellaneous com o
// Subject cru — o assunto inteiro, prefixo e tudo — nunca é descartado.
type Entry struct {
	Hash     string
	Type     Type
	Scope    string
	Breaking bool
	Subject  string
}

// subjectPattern segue a gramática do Conventional Commits 1.0.0: tipo,
// escopo opcional entre parênteses, "!" opcional antes dos dois-pontos para
// anunciar mudança incompatível, e o assunto. É a forma abreviada de marcar
// breaking change; a outra forma que a spec permite, o rodapé "BREAKING
// CHANGE:" no corpo do commit, esta versão não lê — exigiria o corpo inteiro
// do commit, não só o assunto (%s), e não há nenhum commit real no
// histórico do gitarias para medir o parser contra ele.
var subjectPattern = regexp.MustCompile(`^([a-zA-Z]+)(\(([^()]+)\))?(!)?: (.+)$`)

func parse(hash string, subject string) Entry {
	match := subjectPattern.FindStringSubmatch(subject)
	if match == nil {
		return Entry{Hash: hash, Type: Miscellaneous, Subject: subject}
	}

	kind, recognized := known(match[1])
	if !recognized {
		return Entry{Hash: hash, Type: Miscellaneous, Subject: subject}
	}

	return Entry{Hash: hash, Type: kind, Scope: match[3], Breaking: match[4] == "!", Subject: match[5]}
}
