package profile

import "time"

// Streak é uma sequência de dias consecutivos com pelo menos um commit,
// contando os dois extremos. Days zero é a sequência vazia, e nela Start e
// End não têm valor.
type Streak struct {
	Days  int
	Start time.Time
	End   time.Time
}
