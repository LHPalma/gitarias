package profile

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func streakCall(author string) string {
	return "log --format=%ad --date=short-local --author=" + author
}

// describe reduz uma sequência a uma linha comparável. A data sai formatada
// de propósito: comparar time.Time com == arrastaria fuso e monotônico para
// dentro da asserção, que é sobre o dia e nada mais.
func describe(streak Streak) string {
	if streak.Days == 0 {
		return "vazia"
	}

	return fmt.Sprintf("%d: %s..%s", streak.Days, streak.Start.Format("2006-01-02"), streak.End.Format("2006-01-02"))
}

func describeDay(day time.Time) string {
	if day.IsZero() {
		return "nenhum"
	}

	return day.Format("2006-01-02")
}

func streakOf(t *testing.T, log string, today string) StreakReport {
	t.Helper()

	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                  {Output: "abc123"},
		streakCall("real@real.com"): {Output: log},
	})

	reference, err := time.Parse("2006-01-02", today)
	if err != nil {
		t.Fatalf("data de referência ilegível no próprio teste: %q", today)
	}

	report, err := NewRepo(runner).Streaks(t.Context(), "real@real.com", reference)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	return report
}

func TestStreaks(t *testing.T) {
	tests := []struct {
		name    string
		log     string
		today   string
		current string
		longest string
		last    string
	}{
		{
			name:    "sequencia em curso termina hoje",
			log:     "2026-09-13\n2026-09-12\n2026-09-11\n2026-09-08\n",
			today:   "2026-09-13",
			current: "3: 2026-09-11..2026-09-13",
			longest: "3: 2026-09-11..2026-09-13",
			last:    "2026-09-13",
		},
		{
			name:    "dia de hoje ainda sem commit nao quebra a sequencia",
			log:     "2026-09-12\n2026-09-11\n",
			today:   "2026-09-13",
			current: "2: 2026-09-11..2026-09-12",
			longest: "2: 2026-09-11..2026-09-12",
			last:    "2026-09-12",
		},
		{
			name:    "dois dias sem commit quebram a sequencia",
			log:     "2026-09-11\n2026-09-10\n",
			today:   "2026-09-13",
			current: "vazia",
			longest: "2: 2026-09-10..2026-09-11",
			last:    "2026-09-11",
		},
		{
			name:    "commits repetidos no mesmo dia contam um dia so",
			log:     "2026-09-13\n2026-09-13\n2026-09-13\n2026-09-12\n",
			today:   "2026-09-13",
			current: "2: 2026-09-12..2026-09-13",
			longest: "2: 2026-09-12..2026-09-13",
			last:    "2026-09-13",
		},
		{
			name:    "data de autoria fora de ordem entra no lugar dela",
			log:     "2026-09-11\n2026-09-13\n2026-09-12\n",
			today:   "2026-09-13",
			current: "3: 2026-09-11..2026-09-13",
			longest: "3: 2026-09-11..2026-09-13",
			last:    "2026-09-13",
		},
		{
			name:    "maior sequencia empatada fica com a mais recente",
			log:     "2026-08-02\n2026-08-01\n2026-07-02\n2026-07-01\n",
			today:   "2026-09-13",
			current: "vazia",
			longest: "2: 2026-08-01..2026-08-02",
			last:    "2026-08-02",
		},
		{
			name:    "sequencia atravessa a virada do ano",
			log:     "2027-01-01\n2026-12-31\n2026-12-30\n",
			today:   "2027-01-01",
			current: "3: 2026-12-30..2027-01-01",
			longest: "3: 2026-12-30..2027-01-01",
			last:    "2027-01-01",
		},
		{
			name:    "commit datado no futuro nao abre sequencia",
			log:     "2026-09-20\n2026-09-11\n",
			today:   "2026-09-13",
			current: "vazia",
			longest: "1: 2026-09-20..2026-09-20",
			last:    "2026-09-20",
		},
		{
			name:    "so commit datado no futuro deixa a atual vazia",
			log:     "2026-09-20\n",
			today:   "2026-09-13",
			current: "vazia",
			longest: "1: 2026-09-20..2026-09-20",
			last:    "2026-09-20",
		},
		{
			name:    "commit datado no futuro nao interrompe a de hoje",
			log:     "2026-09-20\n2026-09-13\n2026-09-12\n",
			today:   "2026-09-13",
			current: "2: 2026-09-12..2026-09-13",
			longest: "2: 2026-09-12..2026-09-13",
			last:    "2026-09-20",
		},
		{
			name:    "um dia so",
			log:     "2026-09-13\n",
			today:   "2026-09-13",
			current: "1: 2026-09-13..2026-09-13",
			longest: "1: 2026-09-13..2026-09-13",
			last:    "2026-09-13",
		},
		{
			name:    "autor sem nenhum commit",
			log:     "",
			today:   "2026-09-13",
			current: "vazia",
			longest: "vazia",
			last:    "nenhum",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := streakOf(t, test.log, test.today)

			if describe(report.Current) != test.current {
				t.Errorf("sequência em curso = %s, queria %s", describe(report.Current), test.current)
			}
			if describe(report.Longest) != test.longest {
				t.Errorf("maior sequência = %s, queria %s", describe(report.Longest), test.longest)
			}
			if describeDay(report.Last) != test.last {
				t.Errorf("último dia = %s, queria %s", describeDay(report.Last), test.last)
			}
		})
	}
}

func TestStreaksIgnoresTheHourOfTheReference(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                  {Output: "abc123"},
		streakCall("real@real.com"): {Output: "2026-09-13\n2026-09-12\n"},
	})

	lateAtNight := time.Date(2026, 9, 13, 23, 59, 59, 0, time.Local)

	report, err := NewRepo(runner).Streaks(t.Context(), "real@real.com", lateAtNight)
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if describe(report.Current) != "2: 2026-09-12..2026-09-13" {
		t.Errorf("sequência em curso = %s; a hora da referência não pode mudar o dia dela", describe(report.Current))
	}
}

func TestStreaksFiltersByTheAuthorAsked(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                      {Output: "abc123"},
		streakCall("natalia@teste.com"): {Output: "2026-09-13\n"},
	})

	if _, err := NewRepo(runner).Streaks(t.Context(), "natalia@teste.com", time.Now()); err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	if runner.Calls[1] != streakCall("natalia@teste.com") {
		t.Errorf("chamada = %q, o autor pedido tem de chegar ao git log", runner.Calls[1])
	}
}

func TestStreaksSkipsTheLogInAnEmptyRepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	report, err := NewRepo(runner).Streaks(t.Context(), "real@real.com", time.Now())
	if err != nil {
		t.Fatalf("repositório sem commit é lista vazia, não erro; veio %v", err)
	}
	if describe(report.Longest) != "vazia" || !report.Last.IsZero() {
		t.Errorf("relatório = %+v, queria vazio", report)
	}
	if len(runner.Calls) != 1 {
		t.Errorf("chamadas = %v, sem HEAD não há log a pedir", runner.Calls)
	}
}

func TestStreaksPropagatesTheEmptyCheckFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Streaks(t.Context(), "real@real.com", time.Now()); err == nil {
		t.Fatal("falha real (não código 1 de repositório vazio) tem de virar erro")
	}
}

func TestStreaksPropagatesTheLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                  {Output: "abc123"},
		streakCall("real@real.com"): {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).Streaks(t.Context(), "real@real.com", time.Now()); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestStreaksRejectsAnUnreadableDate(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead:                  {Output: "abc123"},
		streakCall("real@real.com"): {Output: "2026-09-13\nontem\n"},
	})

	_, err := NewRepo(runner).Streaks(t.Context(), "real@real.com", time.Now())
	if err == nil {
		t.Fatal("data ilegível tem de virar erro, não sequência inventada")
	}
	if !strings.Contains(err.Error(), "ontem") {
		t.Errorf("erro = %v, queria o valor ilegível nomeado", err)
	}
}

// TestStreaksAcrossEveryMonthBoundary varre sequências de 1 a 6 dias
// terminando em cada dia de quatro anos, bissexto incluído, e fixa que o
// tamanho do mês não entra na conta: 31 de agosto e 1º de setembro são dias
// consecutivos como quaisquer outros, e a aritmética de dia não conhece mês.
func TestStreaksAcrossEveryMonthBoundary(t *testing.T) {
	first := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

	for offset := range 4 * 365 {
		end := first.AddDate(0, 0, offset)

		for length := 1; length <= 6; length++ {
			lines := make([]string, 0, length)
			for back := range length {
				lines = append(lines, end.AddDate(0, 0, -back).Format("2006-01-02"))
			}

			runner := gittest.NewRunner(map[string]gittest.Response{
				verifyHead:                  {Output: "abc123"},
				streakCall("real@real.com"): {Output: strings.Join(lines, "\n")},
			})

			report, err := NewRepo(runner).Streaks(t.Context(), "real@real.com", end)
			if err != nil {
				t.Fatalf("%s, %d dias: não esperava erro, veio %v", lines[0], length, err)
			}
			if report.Current.Days != length || report.Longest.Days != length {
				t.Fatalf("terminando em %s: em curso = %d e maior = %d, montei %d dias seguidos (%v)",
					lines[0], report.Current.Days, report.Longest.Days, length, lines)
			}
		}
	}
}
