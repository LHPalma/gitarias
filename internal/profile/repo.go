package profile

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LHPalma/gitarias/internal/git"
)

type Repo struct {
	runner git.Runner
}

func NewRepo(runner git.Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// Identity devolve o e-mail configurado como autor neste repositório — git
// config user.email —, ou o nome se só ele estiver configurado. Vazio quando
// nenhum dos dois está.
func (repo *Repo) Identity(ctx context.Context) (string, error) {
	email, err := repo.setting(ctx, "user.email")
	if err != nil {
		return "", err
	}
	if email != "" {
		return email, nil
	}

	return repo.setting(ctx, "user.name")
}

// setting devolve o valor configurado, vazio quando a chave não existe — o
// código de saída 1 do próprio git config --get, não qualquer erro: falha de
// verdade (cancelamento incluído) tem de subir, não virar "não configurado".
func (repo *Repo) setting(ctx context.Context, key string) (string, error) {
	output, err := repo.runner.Run(ctx, "config", "--get", key)
	if err == nil {
		return strings.TrimSpace(output), nil
	}

	var exitError *git.ExitError
	if errors.As(err, &exitError) && exitError.Code == 1 {
		return "", nil
	}

	return "", err
}

// CommitCount conta quantos commits de identity caem entre since e until —
// os dois em AAAA-MM-DD, os dois dias inteiros incluídos — no HEAD atual.
// Repositório sem nenhum commit ainda devolve zero, não erro.
//
// since/until viram meia-noite e 23:59:59 explícitas antes de chegar ao git,
// nunca a data nua: --since=2026-08-15 sozinho não vale meia-noite daquele
// dia, vale a hora corrente de agora, nesse dia — medido contra o git antes
// de decidir por isso. Sem a hora explícita, --since e --until iguais (um
// dia só) davam zero, mesmo com commit dentro do dia.
func (repo *Repo) CommitCount(ctx context.Context, identity string, since string, until string) (int, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return 0, err
	}
	if empty {
		return 0, nil
	}

	output, err := repo.runner.Run(ctx, "rev-list", "--count", "--author="+identity,
		"--since="+since+" 00:00:00", "--until="+until+" 23:59:59", "HEAD")
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, fmt.Errorf("rev-list devolveu uma contagem ilegível: %q", output)
	}

	return count, nil
}

// Unpushed conta quantos commits do HEAD atual ainda não chegaram ao
// upstream configurado — @{u}. hasUpstream vem falso, sem erro, quando a
// branch atual não tem upstream: é o estado normal de uma branch nova, e
// quem chama precisa distinguir isso de "não dá para saber".
func (repo *Repo) Unpushed(ctx context.Context) (int, bool, error) {
	output, err := repo.runner.Run(ctx, "rev-list", "--count", "@{u}..HEAD")
	if err != nil {
		var exitError *git.ExitError
		if errors.As(err, &exitError) && strings.Contains(exitError.Message, "no upstream") {
			return 0, false, nil
		}

		return 0, false, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, false, fmt.Errorf("rev-list devolveu uma contagem ilegível: %q", output)
	}

	return count, true, nil
}

func (repo *Repo) empty(ctx context.Context) (bool, error) {
	_, err := repo.runner.Run(ctx, "rev-parse", "--verify", "--quiet", "HEAD")
	if err == nil {
		return false, nil
	}

	var exitError *git.ExitError
	if errors.As(err, &exitError) && exitError.Code == 1 {
		return true, nil
	}

	return false, err
}

const dayLayout = "2006-01-02"

// Streaks apura as sequências de dias com commit de author no HEAD atual: a
// que está em curso e a maior de todo o histórico. author casa por
// substring, como o --author do próprio git log. today é o dia de
// referência da sequência em curso e vem de quem chama, porque o domínio não
// lê o relógio — só a data dele conta, a hora é descartada.
//
// A sequência em curso não quebra no dia que ainda está correndo: sem commit
// hoje, ela termina ontem e continua valendo; quebra quando ontem também não
// teve. Repositório sem nenhum commit, ou autor sem nenhum, devolve as duas
// sequências vazias e Last sem valor, não erro.
//
// O dia é o da autoria (%ad) no fuso de quem roda (--date=short-local): é o
// dia que a pessoa viveu, e é o que sobrevive a um rebase, que preserva a
// data de autoria e reescreve a de commit.
func (repo *Repo) Streaks(ctx context.Context, author string, today time.Time) (StreakReport, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return StreakReport{}, err
	}
	if empty {
		return StreakReport{}, nil
	}

	output, err := repo.runner.Run(ctx, "log", "--format=%ad", "--date=short-local", "--author="+author)
	if err != nil {
		return StreakReport{}, err
	}

	days, err := parseDays(output)
	if err != nil {
		return StreakReport{}, err
	}
	if len(days) == 0 {
		return StreakReport{}, nil
	}

	return StreakReport{Current: current(days, civil(today)), Longest: longest(days), Last: days[0]}, nil
}

// parseDays lê a saída do log em dias distintos, do mais novo para o mais
// velho. A ordenação não é redundante: o git log ordena por data de commit, e
// a de autoria pode vir fora de ordem — rebase, cherry-pick e amend deslocam
// uma sem a outra.
func parseDays(output string) ([]time.Time, error) {
	seen := map[string]bool{}
	days := []time.Time{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true

		day, err := time.Parse(dayLayout, line)
		if err != nil {
			return nil, fmt.Errorf("git log devolveu uma data ilegível: %q", line)
		}

		days = append(days, day)
	}

	sort.Slice(days, func(first int, second int) bool { return days[first].After(days[second]) })

	return days, nil
}

// civil reduz um instante ao dia dele, representado à meia-noite UTC. Toda a
// aritmética de dia acontece nessa representação para que somar ou subtrair
// um dia não caia num horário que não existe: onde o horário de verão entra à
// meia-noite, o dia anterior de uma meia-noite local não é meia-noite.
func civil(moment time.Time) time.Time {
	year, month, day := moment.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// current devolve a sequência que chega a today ou a ontem, e vazia quando
// nenhum dos dois teve commit. days vem sem repetição, do mais novo para o
// mais velho. Dia datado no futuro é pulado em vez de abrir sequência:
// relógio adiantado de quem commitou não inventa constância.
func current(days []time.Time, today time.Time) Streak {
	yesterday := today.AddDate(0, 0, -1)

	for index, day := range days {
		if day.After(today) {
			continue
		}
		if day.Before(yesterday) {
			return Streak{}
		}

		return runEndingAt(days, index)
	}

	return Streak{}
}

// longest devolve a maior sequência do histórico, a mais recente em caso de
// empate. Índice que cai no meio de uma sequência já contada é pulado, então
// cada dia é visitado no máximo duas vezes.
func longest(days []time.Time) Streak {
	var best Streak

	for index, day := range days {
		if index > 0 && days[index-1].Equal(day.AddDate(0, 0, 1)) {
			continue
		}

		if run := runEndingAt(days, index); run.Days > best.Days {
			best = run
		}
	}

	return best
}

// runEndingAt estende para trás a sequência que termina em days[index],
// enquanto os dias forem consecutivos.
func runEndingAt(days []time.Time, index int) Streak {
	streak := Streak{Days: 1, Start: days[index], End: days[index]}

	for next := index + 1; next < len(days); next++ {
		if !days[next].Equal(streak.Start.AddDate(0, 0, -1)) {
			break
		}

		streak.Start = days[next]
		streak.Days++
	}

	return streak
}

// weekdayOrder é a ordem em que a semana sai: começa na segunda e termina no
// domingo, porque a pergunta é sobre hábito de trabalho e o fim de semana diz
// mais no fim da tabela do que partido entre as pontas. time.Weekday começa
// no domingo, então a ordem é explícita e não a do enum.
var weekdayOrder = []time.Weekday{
	time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
	time.Friday, time.Saturday, time.Sunday,
}

const hoursInDay = 24

// CommitCountByHour quebra por hora do dia a mesma contagem do CommitCount —
// mesma identidade, mesmo período, mesmos limites explícitos. Devolve as 24
// horas sempre, inclusive as de contagem zero: a forma da distribuição é a
// resposta, e hora ausente da tabela deixaria quem lê contando linha.
func (repo *Repo) CommitCountByHour(ctx context.Context, identity string, since string, until string) ([]HourCount, error) {
	moments, err := repo.moments(ctx, identity, since, until)
	if err != nil {
		return nil, err
	}

	counts := make([]HourCount, hoursInDay)
	for hour := range counts {
		counts[hour] = HourCount{Hour: hour}
	}
	for _, moment := range moments {
		counts[moment.Hour()].Commits++
	}

	return counts, nil
}

// CommitCountByWeekday quebra a mesma contagem por dia da semana, da segunda
// ao domingo. Como o CommitCountByHour, devolve os sete dias sempre.
func (repo *Repo) CommitCountByWeekday(ctx context.Context, identity string, since string, until string) ([]WeekdayCount, error) {
	moments, err := repo.moments(ctx, identity, since, until)
	if err != nil {
		return nil, err
	}

	commits := map[time.Weekday]int{}
	for _, moment := range moments {
		commits[moment.Weekday()]++
	}

	counts := make([]WeekdayCount, 0, len(weekdayOrder))
	for _, weekday := range weekdayOrder {
		counts = append(counts, WeekdayCount{Weekday: weekday, Commits: commits[weekday]})
	}

	return counts, nil
}

// moments devolve o instante de autoria de cada commit da identidade no
// período, já convertido para o fuso de quem roda pelo próprio git. O
// --date=iso-strict-local é RFC 3339 exato — medido contra o git antes de
// escolher —, então a stdlib parseia com a constante dela e o deslocamento
// vem embutido: a hora e o dia da semana lidos daqui já são os locais, sem
// segunda conversão.
//
// Repositório sem nenhum commit devolve lista vazia, não erro, como o
// CommitCount devolve zero.
func (repo *Repo) moments(ctx context.Context, identity string, since string, until string) ([]time.Time, error) {
	empty, err := repo.empty(ctx)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}

	output, err := repo.runner.Run(ctx, "log", "--format=%ad", "--date=iso-strict-local",
		"--author="+identity, "--since="+since+" 00:00:00", "--until="+until+" 23:59:59")
	if err != nil {
		return nil, err
	}

	moments := []time.Time{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		moment, err := time.Parse(time.RFC3339, line)
		if err != nil {
			return nil, fmt.Errorf("git log devolveu um instante ilegível: %q", line)
		}

		moments = append(moments, moment)
	}

	return moments, nil
}
