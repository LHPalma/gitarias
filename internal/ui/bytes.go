package ui

import "strconv"

const step = 1024

// DescribeBytes escreve um tamanho para leitura humana, com uma casa decimal a
// partir de KB. Só a tela usa isto: nos formatos delimitados o número sai cru,
// porque quem lê é uma planilha ou um script.
func DescribeBytes(size int64) string {
	if size < step {
		return strconv.FormatInt(size, 10) + " B"
	}

	value := float64(size)
	for _, unit := range []string{"KB", "MB", "GB", "TB"} {
		value /= step
		if value < step {
			return strconv.FormatFloat(value, 'f', 1, 64) + " " + unit
		}
	}

	return strconv.FormatFloat(value/step, 'f', 1, 64) + " PB"
}
