package tui

import "github.com/charmbracelet/lipgloss"

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	checkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
