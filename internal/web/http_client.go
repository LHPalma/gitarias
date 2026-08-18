package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient é o Client de verdade: um GET, sem cabeçalho especial e sem
// retry — quem chama decide o prazo, pelo context.
type HTTPClient struct{}

func (HTTPClient) Get(ctx context.Context, url string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s devolveu %s", url, response.Status)
	}

	return string(body), nil
}
