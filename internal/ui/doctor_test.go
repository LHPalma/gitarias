package ui

import (
	"testing"

	"github.com/LHPalma/gitarias/internal/doctor"
)

func TestDescribeCheck(t *testing.T) {
	tests := []struct {
		name  string
		state doctor.State
		want  string
	}{
		{name: "ok", state: doctor.Ok, want: "ok"},
		{name: "aviso", state: doctor.Warning, want: "aviso"},
		{name: "falha", state: doctor.Failure, want: "falta"},
		{name: "estado desconhecido nao alarma", state: doctor.State(99), want: "ok"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DescribeCheck(doctor.Check{State: test.state}); got != test.want {
				t.Errorf("rotulo = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDescribeCheckStaysApartFromTheToken(t *testing.T) {
	failed := doctor.Check{State: doctor.Failure}

	if DescribeCheck(failed) == failed.State.String() {
		t.Error("o rotulo de tela e o token de maquina sao vocabularios diferentes; o roadmap 9 traduz um e nao o outro")
	}
}

func TestDescribeCheckMarksTheSkipped(t *testing.T) {
	if got := DescribeCheck(doctor.Check{State: doctor.Skipped}); got != "--" {
		t.Errorf("rotulo = %q; pulado nao e ok nem falha, e nao pode se disfarcar de nenhum dos dois", got)
	}
}
