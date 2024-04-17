package view

// A simple example that shows how to render an animated progress bar. In this
// example we bump the progress by 25% every two seconds, animating our
// progress bar to its new target state.
//
// It's also possible to render a progress bar in a more static fashion without
// transitions. For details on that approach see the progress-static example.

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

const (
	padding  = 2
	maxWidth = 80
)

type ProgressBar struct {
	updateChan    chan float64
	title         string
	progressValue float64
	progress      progress.Model
}

func NewProgressBar(title string) ProgressBar {
	return ProgressBar{
		updateChan: make(chan float64, 1000),
		title:      title,
		progress:   progress.New(progress.WithDefaultGradient()),
	}
}

func (m ProgressBar) UpdateProgressBar(v float64) {
	m.updateChan <- v
}

func (m ProgressBar) Init() tea.Cmd {
	return tickCmd()
}

func (m ProgressBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:
		if m.progress.Percent() == 1.0 {
			return m, tea.Quit
		}

		cmd := tickCmd()
		for {
			shouldBreak := false
			select {
			case u := <-m.updateChan:
				m.progressValue = u
				cmd = tea.Batch(tickCmd(), m.progress.SetPercent(u))
			default:
				shouldBreak = true
			}
			if shouldBreak {
				break
			}
		}
		return m, tea.Batch(tickCmd(), cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m ProgressBar) View() string {
	if m.progressValue == 1 {
		return ""
	}
	pad := strings.Repeat(" ", padding)

	return "\n" +
		pad + m.title + "\n\n" +
		pad + m.progress.View()
}
