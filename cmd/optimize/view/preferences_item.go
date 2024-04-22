package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	preferences2 "github.com/kaytu-io/pennywise/cmd/optimize/preferences"
	"strconv"
	"strings"
)

type PreferenceItem struct {
	pref        preferences2.PreferenceItem
	input       textinput.Model
	valueIdx    int
	hidden      bool
	hideService bool
}

func NewPreferenceItem(pref preferences2.PreferenceItem) *PreferenceItem {
	in := textinput.New()
	in.CharLimit = 30
	in.Width = 30
	i := PreferenceItem{
		input:    in,
		pref:     pref,
		valueIdx: 0,
	}
	i.ReconfigureInput()
	return &i
}

func (m *PreferenceItem) ReconfigureInput() {
	if m.pref.Pinned {
		m.input.Placeholder = "Pinned to current EC2 Instance"
		m.input.Validate = pinnedValidator
		m.input.SetValue("")
		m.pref.Value = nil
	} else {
		m.input.Placeholder = "Any"
		m.input.Validate = nil
		if m.pref.Value != nil {
			m.input.SetValue(*m.pref.Value)
		}
		m.input.ShowSuggestions = true
		m.input.SetSuggestions(m.pref.PossibleValues)
		if len(m.pref.PossibleValues) > 0 {
			m.input.Placeholder = "Any"
			m.input.SetValue(m.pref.PossibleValues[m.valueIdx])
			m.pref.Value = aws.String(m.pref.PossibleValues[m.valueIdx])
			m.input.Validate = pinnedValidator
		}
		if m.pref.IsNumber {
			m.input.Validate = numberValidator
		}
	}
}

func (m *PreferenceItem) Init() tea.Cmd { return textinput.Blink }

func (m *PreferenceItem) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.hidden {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			if m.pref.PreventPinning {
				break
			}
			m.pref.Pinned = !m.pref.Pinned
			m.valueIdx = 0
			m.ReconfigureInput()

		case tea.KeyRight:
			if l := len(m.pref.PossibleValues); l > 0 {
				m.valueIdx = (m.valueIdx + 1) % l
				m.input.CursorEnd()
				m.ReconfigureInput()
			}
			if m.pref.IsNumber {
				curr, _ := strconv.ParseInt(m.input.Value(), 10, 64)
				curr++
				newVal := fmt.Sprintf("%d", curr)
				m.input.SetValue(newVal)
				m.pref.Value = aws.String(newVal)
				m.input.CursorEnd()
			}

		case tea.KeyLeft:
			if l := len(m.pref.PossibleValues); l > 0 {
				m.valueIdx--
				if m.valueIdx < 0 {
					m.valueIdx = l - 1
				}
				m.input.CursorEnd()
				m.ReconfigureInput()
			}
			if m.pref.IsNumber {
				curr, _ := strconv.ParseInt(m.input.Value(), 10, 64)
				curr--
				newVal := fmt.Sprintf("%d", curr)
				m.input.SetValue(newVal)
				m.pref.Value = aws.String(newVal)
				m.input.CursorEnd()
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *PreferenceItem) View() string {
	if m.hidden {
		return ""
	}
	builder := strings.Builder{}

	key := m.pref.Key
	if !m.hideService {
		key = fmt.Sprintf("%s: %s", m.pref.Service, key)
	}
	if len(m.pref.Unit) > 0 {
		key = fmt.Sprintf("%s (%s)", key, m.pref.Unit)
	}
	builder.WriteString(" ")
	builder.WriteString(inputStyle.Width(45).Render(key))
	builder.WriteString(" ")
	builder.WriteString(m.input.View())
	if len(m.pref.PossibleValues) > 1 && m.input.Focused() && !m.pref.Pinned {
		builder.WriteString(continueStyle.Render(" ←/→ to change value"))
	}
	builder.WriteString("\n")

	return builder.String()
}

func (m *PreferenceItem) Blur() {
	m.input.Blur()
}

func (m *PreferenceItem) Focus() {
	m.input.Focus()
}
