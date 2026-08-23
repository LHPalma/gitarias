package forge

import (
	"context"
	"time"
)

// Source é de onde os pull requests vêm. Existe como contrato porque há dois
// caminhos previstos: embrulhar o gh, que já resolve autenticação e host, e
// falar HTTP com o token do ambiente, para onde o gh não está instalado.
type Source interface {
	PullRequests(ctx context.Context, limit int) ([]PullRequest, error)
	Viewer(ctx context.Context) (string, error)
	Scopes(ctx context.Context) ([]string, error)
	AccountCommitCount(ctx context.Context, since time.Time, until time.Time) (int, error)
	AccountCommitCountByRepository(ctx context.Context, since time.Time, until time.Time) ([]RepositoryCommitCount, error)
}
