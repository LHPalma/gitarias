package ui

// DescribeResidence diz se o caminho ainda está na árvore. O rótulo do que
// saiu é o achado do comando: o arquivo sumiu da árvore e continua sendo
// baixado em todo clone.
func DescribeResidence(inTree bool) string {
	if inTree {
		return "na árvore"
	}

	return "só no histórico"
}
