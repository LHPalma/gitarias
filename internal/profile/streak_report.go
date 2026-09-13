package profile

import "time"

// StreakReport reúne a sequência em curso, a maior de todo o histórico e o
// dia do commit mais recente. Sem nenhum commit de quem foi pedido, as duas
// sequências vêm vazias e Last sem valor.
type StreakReport struct {
	Current Streak
	Longest Streak
	Last    time.Time
}
