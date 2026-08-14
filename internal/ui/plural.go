package ui

// Plural devolve singular quando count é 1, e plural em qualquer outro caso,
// inclusive zero e valores negativos.
func Plural(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}
