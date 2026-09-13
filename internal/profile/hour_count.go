package profile

// HourCount é uma hora do dia, de 0 a 23, e quantos commits caíram nela. A
// hora é a do fuso de quem roda, não a gravada pelo autor.
type HourCount struct {
	Hour    int
	Commits int
}
