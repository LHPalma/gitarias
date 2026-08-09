package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDescribeCancellation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "cancelamento", err: context.Canceled, want: "interrompido"},
		{name: "prazo esgotado", err: context.DeadlineExceeded, want: "tempo esgotado"},
		{name: "erro qualquer fica como veio", err: errors.New("disco cheio"), want: "disco cheio"},
		{name: "cancelamento embrulhado", err: errWrapped, want: "interrompido"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := describeCancellation(test.err).Error(); got != test.want {
				t.Errorf("mensagem = %q, queria %q", got, test.want)
			}
		})
	}
}

var errWrapped = errors.Join(errors.New("verificando"), context.Canceled)

func TestDescribeCancellationSpeaksPortuguese(t *testing.T) {
	message := describeCancellation(context.Canceled).Error()

	if strings.Contains(message, "context") {
		t.Errorf("mensagem = %q; RN-07: o erro que se le e o da ferramenta, nao o da stdlib", message)
	}
}
