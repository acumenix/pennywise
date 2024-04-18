package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type OptimizationItem struct {
	Instance            types.Instance
	Region              string
	OptimizationLoading bool
	TargetInstanceType  string
	TotalSaving         float64
}

type Ec2InstanceOptimizations struct {
	itemsChan chan OptimizationItem
	loading   bool
	debugMsg  string

	table        table.Model
	items        []OptimizationItem
	selectedItem *OptimizationItem
}

func NewEC2InstanceOptimizations() *Ec2InstanceOptimizations {
	columns := []table.Column{
		{Title: "InstanceId", Width: 30},
		{Title: "InstanceType", Width: 20},
		{Title: "Region", Width: 10},
		{Title: "PlatformDetails", Width: 15},
		{Title: "TargetInstanceType", Width: 20},
		{Title: "TotalSaving", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(nil),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &Ec2InstanceOptimizations{
		itemsChan:    make(chan OptimizationItem, 1000),
		loading:      false,
		table:        t,
		items:        nil,
		selectedItem: nil,
	}
}

func (m *Ec2InstanceOptimizations) Init() tea.Cmd { return tickCmdWithDuration(time.Millisecond * 50) }

func (m *Ec2InstanceOptimizations) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tickMsg:
		for {
			nothingToAdd := false
			select {
			case newItem := <-m.itemsChan:
				updated := false
				for idx, i := range m.items {
					if *newItem.Instance.InstanceId == *i.Instance.InstanceId {
						m.items[idx] = newItem
						updated = true
						break
					}
				}
				if !updated {
					m.items = append(m.items, newItem)
				}

				var rows []table.Row
				for _, i := range m.items {
					row := table.Row{
						*i.Instance.InstanceId,
						string(i.Instance.InstanceType),
						i.Region,
						*i.Instance.PlatformDetails,
						i.TargetInstanceType,
						fmt.Sprintf("$%v", i.TotalSaving),
					}
					if i.OptimizationLoading {
						row[4] = "..."
						row[5] = "..."
					}
					rows = append(rows, row)
				}
				m.table.SetRows(rows)
			default:
				nothingToAdd = true
			}
			if nothingToAdd {
				break
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "enter":
			if len(m.table.SelectedRow()) == 0 {
				break
			}

			selectedInstanceID := m.table.SelectedRow()[0]
			for _, i := range m.items {
				if selectedInstanceID == *i.Instance.InstanceId {
					m.selectedItem = &i
				}
			}
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, tea.Batch(cmd, tickCmdWithDuration(time.Millisecond*50))
}

func (m *Ec2InstanceOptimizations) View() string {
	return m.debugMsg + "\n" + baseStyle.Render(m.table.View()) + "\n"
}

func (m *Ec2InstanceOptimizations) SendItem(item OptimizationItem) {
	m.itemsChan <- item
}

func (m *Ec2InstanceOptimizations) Finished() {
	m.loading = false
}
