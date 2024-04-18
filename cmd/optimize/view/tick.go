package view

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tickCmdWithDuration(time.Second)
}

func tickCmdWithDuration(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
