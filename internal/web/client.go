package web

import "context"

// Client busca o corpo de uma URL por HTTP puro, sem passar pelo gh — a
// única porta do gtr que fala direto com a internet.
type Client interface {
	Get(ctx context.Context, url string) (string, error)
}
