package ui

import (
	"testing"
	"time"
)

// TestDescribeWeekday cobre os sete ramos direto, um a um: switch de rótulo
// sem teste por caso deixa passar troca entre dois deles, que nenhuma outra
// asserção do projeto pegaria.
func TestDescribeWeekday(t *testing.T) {
	tests := []struct {
		weekday time.Weekday
		wanted  string
	}{
		{time.Monday, "segunda"},
		{time.Tuesday, "terça"},
		{time.Wednesday, "quarta"},
		{time.Thursday, "quinta"},
		{time.Friday, "sexta"},
		{time.Saturday, "sábado"},
		{time.Sunday, "domingo"},
	}

	for _, test := range tests {
		t.Run(test.wanted, func(t *testing.T) {
			if described := DescribeWeekday(test.weekday); described != test.wanted {
				t.Errorf("DescribeWeekday(%v) = %q, queria %q", test.weekday, described, test.wanted)
			}
		})
	}
}
