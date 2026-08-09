package worktree

import (
	"strconv"
	"strings"
)

func unquote(value string) string {
	if !strings.HasPrefix(value, `"`) {
		return value
	}

	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return value
	}

	return unquoted
}
