package cmd

import (
	"strings"
	"testing"
)

func TestLicensesCommandShowsOnlyTheSummaryByDefault(t *testing.T) {
	result := execute(t, nil, "", "licenses")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "resumo") {
		t.Errorf("saída = %q, queria o resumo", result.stdout)
	}
	if strings.Contains(result.stdout, "texto completo") {
		t.Errorf("saída = %q, sem --full o texto integral fica de fora", result.stdout)
	}
	if len(result.calls) != 0 {
		t.Errorf("chamadas = %v, a licença não depende de repositório nenhum", result.calls)
	}
}

func TestLicensesCommandShowsEverythingWithFull(t *testing.T) {
	result := execute(t, nil, "", "licenses", "--full")

	if result.err != nil {
		t.Fatalf("não esperava erro, veio %v", result.err)
	}
	if !strings.Contains(result.stdout, "texto completo") {
		t.Errorf("saída = %q, queria o texto integral", result.stdout)
	}
}

func TestLicensesCommandWorksOutsideARepository(t *testing.T) {
	result := execute(t, nil, "", "licenses")

	if result.err != nil {
		t.Fatalf("a licença tem de sair mesmo fora de um repositório git, veio %v", result.err)
	}
}

func TestNoticeWithoutASeparator(t *testing.T) {
	if got := notice("licença sem separador\n", false); !strings.Contains(got, "licença sem separador") {
		t.Errorf("resumo = %q, sem separador o texto inteiro é o resumo", got)
	}
}

func TestNoticeCutsTheSeparatorEvenOnCRLF(t *testing.T) {
	crlf := strings.ReplaceAll(noNotices, "\n", "\r\n")

	got := notice(crlf, false)

	if strings.Contains(got, "texto completo") {
		t.Errorf("resumo = %q, um checkout Windows com core.autocrlf grava CRLF no arquivo embutido; o separador tem de casar mesmo assim", got)
	}
	if !strings.Contains(got, "resumo") {
		t.Errorf("resumo = %q, o texto antes do separador continua saindo", got)
	}
}

func TestLicensesCommandNeverEndsALineWithSpace(t *testing.T) {
	result := execute(t, nil, "", "licenses", "--full")

	for number, line := range strings.Split(result.stdout, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("RN-10: a linha %d termina em espaço: %q", number+1, line)
		}
	}
}
