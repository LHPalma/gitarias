package profile

import "time"

// WeekdayCount é um dia da semana e quantos commits caíram nele, no fuso de
// quem roda.
type WeekdayCount struct {
	Weekday time.Weekday
	Commits int
}
