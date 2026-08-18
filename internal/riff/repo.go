package riff

import (
	"context"
	"fmt"
	"strings"
)

// source é a API pública do whatthecommit.com: devolve, em texto puro, uma
// linha com uma mensagem de commit aleatória — sem chave, sem autenticação.
const source = "https://whatthecommit.com/index.txt"

// Client é o que este pacote precisa da internet: um GET. internal/web tem a
// implementação de verdade.
type Client interface {
	Get(ctx context.Context, url string) (string, error)
}

type Repo struct {
	client Client
}

func NewRepo(client Client) *Repo {
	return &Repo{client: client}
}

// Random devolve uma mensagem de commit aleatória do whatthecommit.com.
func (repo *Repo) Random(ctx context.Context) (string, error) {
	body, err := repo.client.Get(ctx, source)
	if err != nil {
		return "", err
	}

	message := strings.TrimSpace(body)
	if message == "" {
		return "", fmt.Errorf("whatthecommit.com devolveu uma mensagem vazia")
	}

	return message, nil
}
