package view

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	preferences2 "github.com/kaytu-io/pennywise/cmd/optimize/preferences"
	"strconv"
	"strings"
)

type (
	errMsg error
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

type PreferencesConfiguration struct {
	inputs     []textinput.Model
	focused    int
	valueFocus int
	err        error

	pref  []preferences2.PreferenceItem
	close func([]preferences2.PreferenceItem)
}

func NewPreferencesConfiguration(preferences []preferences2.PreferenceItem, close func([]preferences2.PreferenceItem)) *PreferencesConfiguration {
	var inputs []textinput.Model

	for idx, pref := range preferences {
		in := textinput.New()
		if idx == 0 {
			in.Focus()
		}
		in.CharLimit = pref.MaxCharacters
		in.Width = pref.MaxCharacters + 5
		if pref.Pinned {
			in.Placeholder = "Pinned to current EC2 Instance"
			in.Validate = pinnedValidator
		} else {
			in.Placeholder = "Any"
			if pref.Value != nil {
				in.SetValue(*pref.Value)
			}
		}
		if pref.IsNumber {
			in.Validate = numberValidator
		}
		inputs = append(inputs, in)
	}
	return &PreferencesConfiguration{
		inputs: inputs,
		pref:   preferences,
		close:  close,
	}
}

func (m *PreferencesConfiguration) Init() tea.Cmd { return textinput.Blink }

func (m *PreferencesConfiguration) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m.close(m.pref)
			return m, nil
		case tea.KeyTab:
			pref := m.pref[m.focused]
			in := m.inputs[m.focused]
			pref.Pinned = !pref.Pinned
			if pref.Pinned {
				in.Placeholder = "Pinned to current EC2 Instance"
				in.Validate = pinnedValidator
				in.SetValue("")
				pref.Value = nil
			} else {
				in.Placeholder = "Any"
				in.Validate = nil
				if len(pref.PossibleValues) > 0 {
					in.SetValue(pref.PossibleValues[0])
					in.CursorStart()
				}
			}
			m.pref[m.focused] = pref
			m.inputs[m.focused] = in
		case tea.KeyRight:
			l := len(m.pref[m.focused].PossibleValues)
			if l > 0 {
				m.valueFocus = (m.valueFocus + 1) % l
				m.inputs[m.focused].SetValue(m.pref[m.focused].PossibleValues[m.valueFocus])
				m.inputs[m.focused].CursorStart()
			}

			if m.pref[m.focused].IsNumber {
				curr, _ := strconv.ParseInt(m.inputs[m.focused].Value(), 10, 64)
				curr++
				m.inputs[m.focused].SetValue(fmt.Sprintf("%d", curr))
				m.inputs[m.focused].CursorEnd()
			}
		case tea.KeyLeft:
			l := len(m.pref[m.focused].PossibleValues)
			if l > 0 {
				m.valueFocus = (m.valueFocus - 1) % l
				m.inputs[m.focused].SetValue(m.pref[m.focused].PossibleValues[m.valueFocus])
				m.inputs[m.focused].CursorStart()
			}

			if m.pref[m.focused].IsNumber {
				curr, _ := strconv.ParseInt(m.inputs[m.focused].Value(), 10, 64)
				curr--
				m.inputs[m.focused].SetValue(fmt.Sprintf("%d", curr))
				m.inputs[m.focused].CursorEnd()
			}
		case tea.KeyEnter:
			m.nextInput()
			m.valueFocus = 0
		case tea.KeyUp:
			m.prevInput()
			m.valueFocus = 0
		case tea.KeyDown:
			m.nextInput()
			m.valueFocus = 0
		}
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focused].Focus()

	case errMsg:
		m.err = msg
		return m, nil
	}

	for i := range m.inputs {
		if !m.pref[i].Pinned && len(m.pref[i].PossibleValues) == 0 {
			m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
		}

		val := m.inputs[i].Value()
		if len(val) > 0 {
			m.pref[i].Value = &val
		} else {
			m.pref[i].Value = nil
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *PreferencesConfiguration) View() string {
	builder := strings.Builder{}

	builder.WriteString("Configure your preferences:\n")
	for idx, pref := range m.pref {
		builder.WriteString("  ")
		builder.WriteString(inputStyle.Width(30).Render(pref.Key))
		builder.WriteString("    ")
		builder.WriteString(m.inputs[idx].View())
		builder.WriteString("\n")
	}
	builder.WriteString(helpStyle.Render(`
↑/↓: move
enter: next field
←/→: prev/next value (for fields with specific values)
esc: apply and exit
tab: pin/unpin value to current ec2 instance
ctrl+c: exit
`))
	return builder.String()
}

func pinnedValidator(s string) error {
	if s == "" {
		return nil
	}
	return errors.New("pinned")
}

func numberValidator(s string) error {
	_, err := strconv.ParseInt(s, 10, 64)
	return err
}

func (m *PreferencesConfiguration) nextInput() {
	m.focused = (m.focused + 1) % len(m.inputs)
}

func (m *PreferencesConfiguration) prevInput() {
	m.focused--
	// Wrap around
	if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}
}
