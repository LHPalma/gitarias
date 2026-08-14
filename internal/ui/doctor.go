package ui

import "github.com/LHPalma/gitarias/internal/doctor"

func DescribeCheck(check doctor.Check) string {
	switch check.State {
	case doctor.Warning:
		return "aviso"
	case doctor.Failure:
		return "falta"
	default:
		return "ok"
	}
}
