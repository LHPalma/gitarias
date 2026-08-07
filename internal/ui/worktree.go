package ui

import (
	"strings"

	"github.com/LHPalma/gitarias/internal/worktree"
)

func DescribeCheckout(entry worktree.Worktree) string {
	switch {
	case entry.Bare:
		return "(bare)"
	case entry.Detached:
		return "(HEAD destacado em " + shortHead(entry.Head) + ")"
	case entry.Branch == "":
		return "(sem branch)"
	default:
		return entry.Branch
	}
}

func DescribeState(entry worktree.Worktree) string {
	var states []string
	if entry.Locked {
		states = append(states, withReason("trancado", entry.LockedReason))
	}
	if entry.Prunable {
		states = append(states, withReason("podável", entry.PrunableReason))
	}
	return strings.Join(states, ", ")
}

func withReason(state string, reason string) string {
	if reason == "" {
		return state
	}
	return state + ": " + reason
}

func shortHead(head string) string {
	if len(head) > 7 {
		return head[:7]
	}
	return head
}
