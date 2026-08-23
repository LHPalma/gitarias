package forge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

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

// contributionsQuery pede só o total de commits contribuídos — nada de nome
// de repositório ou detalhe por fonte. É a mesma conta que a atividade do
// GitHub mostra à própria conta, incluindo o que é privado quando o token
// carrega o escopo read:user; sem esse escopo, o GitHub omite o privado da
// soma, calado, sem erro para distinguir os dois casos.
const contributionsQuery = `query($from: DateTime!, $to: DateTime!) {
  viewer {
    contributionsCollection(from: $from, to: $to) {
      totalCommitContributions
    }
  }
}`

// yearWindow é o teto que o contributionsCollection do GitHub aceita numa
// única consulta.
const yearWindow = 365 * 24 * time.Hour

// dateOnly nomeia uma janela num erro sem o ruído da hora, que quem lê não
// pediu e não ajuda a encontrar o período mais curto que resolveria.
const dateOnly = "2006-01-02"

// window é um pedaço do período pedido, pequeno o bastante para uma única
// consulta ao contributionsCollection aceitar.
type window struct {
	from  time.Time
	until time.Time
}

// windows quebra [since, until] em pedaços de até yearWindow, o teto que a
// API aceita por consulta — since e until inclusos nas pontas.
func windows(since time.Time, until time.Time) []window {
	var pieces []window

	for start := since; !start.After(until); start = start.Add(yearWindow) {
		end := start.Add(yearWindow)
		if end.After(until) {
			end = until
		}

		pieces = append(pieces, window{from: start, until: end})
	}

	return pieces
}

// AccountCommitCount conta os commits que a conta autenticada contribuiu, em
// todos os repositórios que ela alcança, entre since e until — os dois
// inclusos. Quem resolve "conta autenticada" é o próprio gh, nunca um login
// escolhido aqui, mesma disciplina do Viewer.
//
// O período pode passar de um ano; a API não aceita isso numa consulta só, e
// por isso é quebrado em janelas de até um ano e somado aqui.
func (source *CLI) AccountCommitCount(ctx context.Context, since time.Time, until time.Time) (int, error) {
	total := 0

	for _, piece := range windows(since, until) {
		count, err := source.commitContributions(ctx, piece.from, piece.until)
		if err != nil {
			return 0, err
		}

		total += count
	}

	return total, nil
}

func (source *CLI) commitContributions(ctx context.Context, from time.Time, to time.Time) (int, error) {
	result, err := source.commands.Run(ctx, "", "gh", "api", "graphql",
		"-f", "query="+contributionsQuery,
		"-f", "from="+from.Format(time.RFC3339),
		"-f", "to="+to.Format(time.RFC3339),
	)
	if err != nil {
		return 0, ErrUnavailable
	}

	if result.Code == unauthenticated {
		return 0, ErrUnauthenticated
	}
	if !result.Passed() {
		return 0, fmt.Errorf("o gh não conseguiu contar as contribuições: %s", strings.TrimSpace(result.Output))
	}

	var response struct {
		Data struct {
			Viewer struct {
				ContributionsCollection struct {
					TotalCommitContributions int `json:"totalCommitContributions"`
				} `json:"contributionsCollection"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(result.Output), &response); err != nil {
		return 0, fmt.Errorf("não entendi a resposta do gh: %w", err)
	}

	return response.Data.Viewer.ContributionsCollection.TotalCommitContributions, nil
}

// maxRepositoriesPerQuery é o teto que o commitContributionsByRepository do
// GitHub aceita — pedir mais que isso é erro da API, não corte silencioso.
const maxRepositoriesPerQuery = 100

const repositoryContributionsQuery = `query($from: DateTime!, $to: DateTime!) {
  viewer {
    contributionsCollection(from: $from, to: $to) {
      totalRepositoriesWithContributedCommits
      commitContributionsByRepository(maxRepositories: 100) {
        repository {
          nameWithOwner
          isPrivate
        }
        contributions {
          totalCount
        }
      }
    }
  }
}`

// minSplit é o menor pedaço que a bisecção tenta antes de desistir e nomear
// o corte. Cem repositórios ativos dentro de uma janela mais estreita que
// isso é um padrão de uso que o gtr não tenta mais adivinhar — a essa altura
// é sinal de algo automatizado, não de uma pessoa commitando.
const minSplit = time.Hour

// AccountCommitCountByRepository quebra AccountCommitCount por repositório,
// entre since e until — os dois inclusos. Igual a AccountCommitCount, um
// período maior que um ano é quebrado em janelas — mas aqui isso não basta:
// o mesmo repositório pode aparecer em janelas diferentes, e quem chama quer
// uma linha por repositório, não uma por repositório-e-janela. Por isso as
// janelas se somam por nameWithOwner aqui dentro, antes de devolver.
func (source *CLI) AccountCommitCountByRepository(ctx context.Context, since time.Time, until time.Time) ([]RepositoryCommitCount, error) {
	totals := make(map[string]RepositoryCommitCount)

	for _, piece := range windows(since, until) {
		if err := source.mergeRepositoryContributions(ctx, piece.from, piece.until, totals); err != nil {
			return nil, err
		}
	}

	counts := make([]RepositoryCommitCount, 0, len(totals))
	for _, count := range totals {
		counts = append(counts, count)
	}

	sort.Slice(counts, func(i int, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}

		return counts[i].Repository < counts[j].Repository
	})

	return counts, nil
}

// mergeRepositoryContributions busca uma janela e soma o que achou em
// totals. Quando a janela vem cortada — mais repositórios ativos do que o
// teto por consulta —, a resposta não é confiável sozinha: GitHub ordena por
// contribuição e trunca o resto calado. Em vez de aceitar isso ou desistir
// na hora, a janela é bisseccionada e cada metade tenta de novo, recursivo,
// até caber ou até minSplit — só aí o corte vira erro nomeado, porque nem
// dividindo mais coube.
func (source *CLI) mergeRepositoryContributions(ctx context.Context, from time.Time, until time.Time, totals map[string]RepositoryCommitCount) error {
	page, err := source.repositoryContributionsPage(ctx, from, until)
	if err != nil {
		return err
	}

	if !page.truncated {
		for _, count := range page.counts {
			merged := totals[count.Repository]
			merged.Repository = count.Repository
			merged.Private = count.Private
			merged.Count += count.Count
			totals[count.Repository] = merged
		}

		return nil
	}

	if until.Sub(from) <= minSplit {
		return fmt.Errorf(
			"a conta contribuiu em %d repositórios entre %s e %s, e --by-repo só traz até %d por consulta; nem dividindo a janela por hora isso coube",
			page.total, from.Format(dateOnly), until.Format(dateOnly), maxRepositoriesPerQuery)
	}

	middle := from.Add(until.Sub(from) / 2)

	if err := source.mergeRepositoryContributions(ctx, from, middle, totals); err != nil {
		return err
	}

	return source.mergeRepositoryContributions(ctx, middle.Add(time.Second), until, totals)
}

// repositoryPage é o que uma única consulta ao commitContributionsByRepository
// devolve — truncated diz se o teto de maxRepositoriesPerQuery cortou a
// resposta, e total é quantos repositórios havia de verdade, só para nomear
// o corte se ele sobreviver até desistir.
type repositoryPage struct {
	counts    []RepositoryCommitCount
	truncated bool
	total     int
}

func (source *CLI) repositoryContributionsPage(ctx context.Context, from time.Time, until time.Time) (repositoryPage, error) {
	result, err := source.commands.Run(ctx, "", "gh", "api", "graphql",
		"-f", "query="+repositoryContributionsQuery,
		"-f", "from="+from.Format(time.RFC3339),
		"-f", "to="+until.Format(time.RFC3339),
	)
	if err != nil {
		return repositoryPage{}, ErrUnavailable
	}

	if result.Code == unauthenticated {
		return repositoryPage{}, ErrUnauthenticated
	}
	if !result.Passed() {
		return repositoryPage{}, fmt.Errorf("o gh não conseguiu contar as contribuições: %s", strings.TrimSpace(result.Output))
	}

	var response struct {
		Data struct {
			Viewer struct {
				ContributionsCollection struct {
					TotalRepositoriesWithContributedCommits int `json:"totalRepositoriesWithContributedCommits"`
					CommitContributionsByRepository         []struct {
						Repository struct {
							NameWithOwner string `json:"nameWithOwner"`
							IsPrivate     bool   `json:"isPrivate"`
						} `json:"repository"`
						Contributions struct {
							TotalCount int `json:"totalCount"`
						} `json:"contributions"`
					} `json:"commitContributionsByRepository"`
				} `json:"contributionsCollection"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(result.Output), &response); err != nil {
		return repositoryPage{}, fmt.Errorf("não entendi a resposta do gh: %w", err)
	}

	collection := response.Data.Viewer.ContributionsCollection
	listed := collection.CommitContributionsByRepository

	counts := make([]RepositoryCommitCount, 0, len(listed))
	for _, entry := range listed {
		counts = append(counts, RepositoryCommitCount{
			Repository: entry.Repository.NameWithOwner,
			Private:    entry.Repository.IsPrivate,
			Count:      entry.Contributions.TotalCount,
		})
	}

	return repositoryPage{
		counts:    counts,
		truncated: len(listed) < collection.TotalRepositoriesWithContributedCommits,
		total:     collection.TotalRepositoriesWithContributedCommits,
	}, nil
}
