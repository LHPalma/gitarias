package ui

import "time"

// DescribeWeekday traduz o dia da semana para o nome em português, em caixa
// baixa. É vocabulário de exibição: o domínio devolve time.Weekday, e a
// tradução mora aqui, alcançável pelos dois fronts.
func DescribeWeekday(weekday time.Weekday) string {
	switch weekday {
	case time.Monday:
		return "segunda"
	case time.Tuesday:
		return "terça"
	case time.Wednesday:
		return "quarta"
	case time.Thursday:
		return "quinta"
	case time.Friday:
		return "sexta"
	case time.Saturday:
		return "sábado"
	default:
		return "domingo"
	}
}
