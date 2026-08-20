package aitrailers

import "fmt"

// Signature devolve o trailer canônico que este pacote associa ao nome de
// ferramenta dado — o mesmo Co-Authored-By que tool() reconheceria de
// volta, para Blame e List/Strip nunca divergirem sobre o que "é" a
// assinatura de cada uma. "claude" e "copilot" são os únicos nomes
// aceitos, os mesmos dois que detect.go já sabe reconhecer.
//
// O ID numérico da conta bot do Copilot é um valor plausível, não
// verificado contra a conta real do GitHub — a detecção de Copilot em
// coAuthorTool não depende dele, só do nome "copilot" e do domínio
// @users.noreply.github.com, então o número em si não muda se bate ou não.
func Signature(name string) (Trailer, error) {
	switch name {
	case "claude":
		return Trailer{Key: "Co-Authored-By", Value: "Claude <" + claudeEmail + ">", Tool: "Claude Code"}, nil
	case "copilot":
		return Trailer{Key: "Co-Authored-By", Value: "Copilot <198982749+Copilot@users.noreply.github.com>", Tool: "GitHub Copilot"}, nil
	default:
		return Trailer{}, fmt.Errorf("ferramenta %q desconhecida; use claude ou copilot", name)
	}
}
