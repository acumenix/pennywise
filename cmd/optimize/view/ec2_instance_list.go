package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"time"
)

type Item struct {
	Instance types.Instance
	Region   string
}

type EC2InstanceList struct {
	itemsChan chan Item
	loading   bool

	items    []Item
	cursor   int
	selected int
}

func NewEC2InstanceList() *EC2InstanceList {
	return &EC2InstanceList{
		itemsChan: make(chan Item, 1000),
		loading:   true,
		selected:  -1,
	}
}

func (m *EC2InstanceList) Init() tea.Cmd {
	return tickCmd()
}

func (m *EC2InstanceList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		for {
			nothingToAdd := false
			select {
			case item := <-m.itemsChan:
				m.items = append(m.items, item)
			default:
				nothingToAdd = true
			}
			if nothingToAdd {
				break
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor
			return m, tea.Quit
		}
	}
	return m, tickCmd()
}

func (m *EC2InstanceList) View() string {
	var b strings.Builder

	if m.loading {
		b.WriteString("Loading all EC2 Instances ")
		for i := int64(0); i < 5; i++ {
			selected := time.Now().Unix() % 5
			if i == selected {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•"))
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•"))
			}
		}
		b.WriteString("\n\n")
	} else {
		b.WriteString("Finished loading all EC2 Instances.\n\n")
	}
	b.WriteString("Which EC2 Instance you want to optimize?\n\n")

	for i, choice := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		name := *choice.Instance.InstanceId
		if choice.Instance.KeyName != nil {
			name = *choice.Instance.KeyName
		}
		b.WriteString(fmt.Sprintf("%s %s - %s - %s - %s\n", cursor, name, choice.Instance.InstanceType, *choice.Instance.PlatformDetails, choice.Region))
	}
	b.WriteString("\nPress q to quit.\n")
	return b.String()
}

func (m *EC2InstanceList) SendItem(item Item) {
	m.itemsChan <- item
}

func (m *EC2InstanceList) Finished() {
	m.loading = false
}

func (m *EC2InstanceList) SelectedItem() *Item {
	if m.selected == -1 {
		return nil
	}
	return &m.items[m.selected]
}
