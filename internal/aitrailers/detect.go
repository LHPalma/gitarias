package aitrailers

import (
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`<([^>]+)>`)

// tool devolve a ferramenta por trás do trailer key: value, e se ele bate em
// alguma assinatura conhecida. A lista é o que já foi visto na prática —
// neste próprio repositório e no ecossistema — não é exaustiva, e cresce por
// medição, não por suposição: Claude Code deixa Co-Authored-By e
// Claude-Session; GitHub Copilot deixa Co-authored-by com a conta bot.
func tool(key string, value string) (string, bool) {
	if strings.EqualFold(key, "Claude-Session") {
		return "Claude Code", true
	}

	if strings.EqualFold(key, "Co-Authored-By") {
		return coAuthorTool(value)
	}

	return "", false
}

func coAuthorTool(value string) (string, bool) {
	email := strings.ToLower(emailOf(value))
	if email == "" {
		return "", false
	}

	switch {
	case email == "noreply@anthropic.com":
		return "Claude Code", true
	case strings.Contains(strings.ToLower(value), "copilot") && strings.HasSuffix(email, "@users.noreply.github.com"):
		return "GitHub Copilot", true
	default:
		return "", false
	}
}

func emailOf(value string) string {
	match := emailPattern.FindStringSubmatch(value)
	if match == nil {
		return ""
	}

	return match[1]
}
