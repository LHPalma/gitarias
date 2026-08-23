package forge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/LHPalma/gitarias/internal/exec"
)

// fields são os campos pedidos ao gh, e a ordem não importa para ele. É
// formato contratual: o gh recusa nome de campo que não conhece, então um erro
// de digitação aqui falha alto em vez de devolver silêncio.
var fields = []string{"number", "title", "headRefName", "baseRefName", "author", "state", "isDraft", "url"}

var (
	// ErrUnavailable é o gh ausente ou impedido de rodar. O front distingue
	// essa recusa da falha de rede: uma se resolve instalando, a outra não.
	ErrUnavailable = errors.New("o gh não está disponível")

	// ErrUnauthenticated é não haver credencial nenhuma. O gh distingue esse
	// caso dos demais pelo código de saída, e é o que torna a checagem de
	// conexão possível sem interpretar prosa.
	ErrUnauthenticated = errors.New("não há credencial configurada para o GitHub")
)

// unauthenticated é o código com que o gh sai quando não encontra credencial
// alguma. Erro de servidor sai com 1, então os dois não se confundem.
const unauthenticated = 4

// CLI busca os pull requests embrulhando o gh, que já resolve autenticação,
// host de GitHub Enterprise e qual repositório é o do diretório atual — e por
// isso o gtr nunca vê o token.
type CLI struct {
	commands exec.Runner
}

func NewCLI(commands exec.Runner) *CLI {
	return &CLI{commands: commands}
}

func (source *CLI) PullRequests(ctx context.Context, limit int) ([]PullRequest, error) {
	arguments := []string{
		"pr", "list",
		"--json", strings.Join(fields, ","),
		"--limit", strconv.Itoa(limit),
	}

	result, err := source.commands.Run(ctx, "", "gh", arguments...)
	if err != nil {
		return nil, ErrUnavailable
	}
	if !result.Passed() {
		return nil, fmt.Errorf("o gh não conseguiu listar os pull requests: %s", strings.TrimSpace(result.Output))
	}

	return parse(result.Output)
}

// listing espelha o json do gh. O author vem como objeto, e é o único campo
// que não é escalar.
type listing struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Head   string `json:"headRefName"`
	Base   string `json:"baseRefName"`
	State  string `json:"state"`
	Draft  bool   `json:"isDraft"`
	URL    string `json:"url"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

func parse(output string) ([]PullRequest, error) {
	var listed []listing
	if err := json.Unmarshal([]byte(output), &listed); err != nil {
		return nil, fmt.Errorf("não entendi a resposta do gh: %w", err)
	}

	requests := make([]PullRequest, 0, len(listed))
	for _, entry := range listed {
		requests = append(requests, PullRequest{
			Number: entry.Number,
			Title:  entry.Title,
			Head:   entry.Head,
			Base:   entry.Base,
			Author: entry.Author.Login,
			State:  strings.ToLower(entry.State),
			Draft:  entry.Draft,
			URL:    entry.URL,
		})
	}

	return requests, nil
}

// Viewer devolve quem o GitHub diz que somos. É a checagem de conexão: uma
// resposta afirma que existe credencial e que o servidor a aceitou, nada mais
// forte que isso.
func (source *CLI) Viewer(ctx context.Context) (string, error) {
	result, err := source.commands.Run(ctx, "", "gh", "api", "user")
	if err != nil {
		return "", ErrUnavailable
	}

	if result.Code == unauthenticated {
		return "", ErrUnauthenticated
	}
	if !result.Passed() {
		return "", fmt.Errorf("o GitHub recusou: %s", strings.TrimSpace(result.Output))
	}

	var identity struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(result.Output), &identity); err != nil {
		return "", fmt.Errorf("não entendi a resposta do gh: %w", err)
	}

	return identity.Login, nil
}

// Scopes devolve as permissões que o token do gh carrega. Existe separado do
// Viewer porque são duas perguntas independentes — quem o GitHub diz que
// somos, e o que o token autoriza — e nem todo chamador precisa das duas.
func (source *CLI) Scopes(ctx context.Context) ([]string, error) {
	result, err := source.commands.Run(ctx, "", "gh", "api", "user", "-i")
	if err != nil {
		return nil, ErrUnavailable
	}

	if result.Code == unauthenticated {
		return nil, ErrUnauthenticated
	}
	if !result.Passed() {
		return nil, fmt.Errorf("o GitHub recusou: %s", strings.TrimSpace(result.Output))
	}

	return scopesFrom(result.Output), nil
}

// scopesFrom lê o cabeçalho X-Oauth-Scopes da resposta crua que o gh api -i
// imprime antes do corpo. O nome do cabeçalho chega com a caixa que o
// servidor escolheu, não a que o padrão HTTP recomendaria, e por isso a busca
// é insensível a caixa. Cabeçalho ausente ou vazio — token sem escopo nenhum
// — devolve nil, nunca erro: a resposta chegou e foi entendida.
func scopesFrom(output string) []string {
	const header = "x-oauth-scopes:"

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < len(header) || !strings.EqualFold(line[:len(header)], header) {
			continue
		}

		value := strings.TrimSpace(line[len(header):])
		if value == "" {
			return nil
		}

		scopes := strings.Split(value, ",")
		for index, scope := range scopes {
			scopes[index] = strings.TrimSpace(scope)
		}

		return scopes
	}

	return nil
}
