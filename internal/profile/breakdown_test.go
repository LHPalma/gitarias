package profile

import (
	"strings"
	"testing"
	"time"

	"github.com/LHPalma/gitarias/internal/git"
	"github.com/LHPalma/gitarias/internal/git/gittest"
)

func momentsCall(identity string, since string, until string) string {
	return "log --format=%ad --date=iso-strict-local --author=" + identity +
		" --since=" + since + " 00:00:00 --until=" + until + " 23:59:59"
}

func broken(t *testing.T, log string) *Repo {
	t.Helper()

	return NewRepo(gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		momentsCall("real@real.com", "2026-09-01", "2026-09-13"): {Output: log},
	}))
}

func countedByHour(t *testing.T, log string) map[int]int {
	t.Helper()

	hours, err := broken(t, log).CommitCountByHour(t.Context(), "real@real.com", "2026-09-01", "2026-09-13")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}
	if len(hours) != 24 {
		t.Fatalf("horas = %d, a distribuição tem de trazer as 24 sempre", len(hours))
	}

	counted := map[int]int{}
	for index, hour := range hours {
		if hour.Hour != index {
			t.Fatalf("hora na posição %d = %d, a tabela tem de sair em ordem", index, hour.Hour)
		}
		if hour.Commits != 0 {
			counted[hour.Hour] = hour.Commits
		}
	}

	return counted
}

func TestCommitCountByHour(t *testing.T) {
	counted := countedByHour(t, strings.Join([]string{
		"2026-09-13T22:41:07+00:00",
		"2026-09-13T22:58:00+00:00",
		"2026-09-12T09:15:00+00:00",
	}, "\n"))

	if counted[22] != 2 || counted[9] != 1 || len(counted) != 2 {
		t.Errorf("contagem = %v, queria duas commits nas 22h e uma nas 9h", counted)
	}
}

func TestCommitCountByHourReadsTheOffsetThatCameWithTheInstant(t *testing.T) {
	counted := countedByHour(t, "2026-09-13T01:30:00-03:00\n")

	if counted[1] != 1 {
		t.Errorf("contagem = %v; a hora é a do deslocamento que veio do git, não a convertida de novo", counted)
	}
}

func TestCommitCountByHourWithoutAnyCommit(t *testing.T) {
	if counted := countedByHour(t, ""); len(counted) != 0 {
		t.Errorf("contagem = %v, queria as 24 horas zeradas", counted)
	}
}

func TestCommitCountByWeekday(t *testing.T) {
	// 2026-09-13 é domingo; 2026-09-07, segunda.
	weekdays, err := broken(t, "2026-09-13T22:41:07+00:00\n2026-09-07T10:00:00+00:00\n2026-09-07T11:00:00+00:00\n").
		CommitCountByWeekday(t.Context(), "real@real.com", "2026-09-01", "2026-09-13")
	if err != nil {
		t.Fatalf("não esperava erro, veio %v", err)
	}

	wanted := []struct {
		weekday time.Weekday
		commits int
	}{
		{time.Monday, 2}, {time.Tuesday, 0}, {time.Wednesday, 0}, {time.Thursday, 0},
		{time.Friday, 0}, {time.Saturday, 0}, {time.Sunday, 1},
	}

	if len(weekdays) != len(wanted) {
		t.Fatalf("dias = %d, a semana inteira tem de sair sempre", len(weekdays))
	}
	for index, expected := range wanted {
		if weekdays[index].Weekday != expected.weekday || weekdays[index].Commits != expected.commits {
			t.Errorf("posição %d = %v com %d commits, queria %v com %d — a semana começa na segunda",
				index, weekdays[index].Weekday, weekdays[index].Commits, expected.weekday, expected.commits)
		}
	}
}

func TestBreakdownsSkipTheLogInAnEmptyRepository(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Err: &git.ExitError{Code: 1, Message: ""}},
	})

	hours, err := NewRepo(runner).CommitCountByHour(t.Context(), "real@real.com", "2026-09-01", "2026-09-13")
	if err != nil {
		t.Fatalf("repositório sem commit não é erro, veio %v", err)
	}
	for _, hour := range hours {
		if hour.Commits != 0 {
			t.Fatalf("hora %d com %d commits num repositório vazio", hour.Hour, hour.Commits)
		}
	}
	if len(runner.Calls) != 1 {
		t.Errorf("chamadas = %v, sem HEAD não há log a pedir", runner.Calls)
	}
}

func TestBreakdownsPropagateTheEmptyCheckFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{verifyHead: {Err: errNotARepository}})

	if _, err := NewRepo(runner).CommitCountByWeekday(t.Context(), "real@real.com", "2026-09-01", "2026-09-13"); err == nil {
		t.Fatal("falha real na checagem de repositório vazio tem de virar erro")
	}
}

func TestBreakdownsPropagateTheLogFailure(t *testing.T) {
	runner := gittest.NewRunner(map[string]gittest.Response{
		verifyHead: {Output: "abc123"},
		momentsCall("real@real.com", "2026-09-01", "2026-09-13"): {Err: errNotARepository},
	})

	if _, err := NewRepo(runner).CommitCountByHour(t.Context(), "real@real.com", "2026-09-01", "2026-09-13"); err == nil {
		t.Fatal("falha do log tem de virar erro")
	}
}

func TestBreakdownsRejectAnUnreadableInstant(t *testing.T) {
	_, err := broken(t, "2026-09-13T22:41:07+00:00\nontem de noite\n").
		CommitCountByWeekday(t.Context(), "real@real.com", "2026-09-01", "2026-09-13")

	if err == nil {
		t.Fatal("instante ilegível tem de virar erro, não hora inventada")
	}
	if !strings.Contains(err.Error(), "ontem de noite") {
		t.Errorf("erro = %v, queria o valor ilegível nomeado", err)
	}
}
