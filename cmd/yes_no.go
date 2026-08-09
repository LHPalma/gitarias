package cmd

func yesNo(value bool) string {
	if value {
		return "sim"
	}

	return "não"
}
