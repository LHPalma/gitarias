package changelog

// Type é o tipo do commit segundo o Conventional Commits. Miscellaneous não é
// um dos onze do padrão: é onde cai o que a assinatura não reconheceu, para
// nenhum commit sumir do changelog.
type Type string

const (
	Feat          Type = "feat"
	Fix           Type = "fix"
	Perf          Type = "perf"
	Refactor      Type = "refactor"
	Docs          Type = "docs"
	Test          Type = "test"
	Build         Type = "build"
	CI            Type = "ci"
	Chore         Type = "chore"
	Style         Type = "style"
	Revert        Type = "revert"
	Miscellaneous Type = "misc"
)

// Types lista as seções na ordem em que aparecem no changelog, a mais
// relevante para quem lê release notes primeiro. Miscellaneous vem por
// último, sempre.
var Types = []Type{Feat, Fix, Perf, Refactor, Docs, Test, Build, CI, Chore, Style, Revert, Miscellaneous}

// known diz se candidate é um dos onze tipos do Conventional Commits.
// Miscellaneous fica de fora de propósito: não é uma assinatura que um
// commit escreve, é o que sobra de quem não escreveu nenhuma.
func known(candidate string) (Type, bool) {
	kind := Type(candidate)
	for _, valid := range Types {
		if valid != Miscellaneous && valid == kind {
			return kind, true
		}
	}

	return "", false
}
