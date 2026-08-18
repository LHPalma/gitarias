package webtest

import (
	"context"
	"fmt"
)

// Client é o web.Client falso, roteirizado por ordem de chamada — como o
// exectest.Runner, e pela mesma razão: uma URL só, sem variação de
// argumento que valha a pena chavear.
type Client struct {
	Responses []Response
	Calls     []string
}

func NewClient(responses ...Response) *Client {
	return &Client{Responses: responses}
}

func (client *Client) Get(ctx context.Context, url string) (string, error) {
	if cancelled := ctx.Err(); cancelled != nil {
		return "", cancelled
	}

	client.Calls = append(client.Calls, url)

	if len(client.Calls) > len(client.Responses) {
		return "", fmt.Errorf("webtest: chamada %d sem resposta roteirizada", len(client.Calls))
	}

	response := client.Responses[len(client.Calls)-1]

	return response.Output, response.Err
}
