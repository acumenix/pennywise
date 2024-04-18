package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	preferences2 "github.com/kaytu-io/pennywise/cmd/optimize/preferences"
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
	CurrentCost         float64
	TargetCost          float64

	AvgCPUUsage string
	TargetCores string

	AvgNetworkBandwidth       string
	TargetNetworkPerformance  string
	CurrentNetworkPerformance string

	CurrentMemory string
	TargetMemory  string

	Preferences []preferences2.PreferenceItem
}

type Ec2InstanceOptimizations struct {
	itemsChan chan OptimizationItem

	table table.Model
	items []OptimizationItem

	detailsPage *Ec2InstanceDetail
	prefConf    *PreferencesConfiguration

	clearScreen  bool
	instanceChan chan OptimizationItem
}

func NewEC2InstanceOptimizations(instanceChan chan OptimizationItem) *Ec2InstanceOptimizations {
	columns := []table.Column{
		{Title: "Instance Id", Width: 23},
		{Title: "Instance Type", Width: 15},
		{Title: "Region", Width: 15},
		{Title: "Platform", Width: 15},
		{Title: "Optimized Instance Type", Width: 25},
		{Title: "Total Saving (Monthly)", Width: 25},
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
		table:        t,
		items:        nil,
		instanceChan: instanceChan,
	}
}

func (m *Ec2InstanceOptimizations) Init() tea.Cmd { return tickCmdWithDuration(time.Millisecond * 50) }

func (m *Ec2InstanceOptimizations) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.detailsPage != nil {
		_, cmd := m.detailsPage.Update(msg)
		return m, tea.Batch(cmd, tickCmdWithDuration(time.Millisecond*50))
	}
	if m.prefConf != nil {
		_, cmd := m.prefConf.Update(msg)
		return m, tea.Batch(cmd, tickCmdWithDuration(time.Millisecond*50))
	}

	var cmd, initCmd tea.Cmd
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
		case "q":
			return m, tea.Quit
		case "p":
			if len(m.table.SelectedRow()) == 0 {
				break
			}
			selectedInstanceID := m.table.SelectedRow()[0]
			for _, i := range m.items {
				if selectedInstanceID == *i.Instance.InstanceId {
					m.prefConf = NewPreferencesConfiguration(i.Preferences, func(items []preferences2.PreferenceItem) {
						i.Preferences = items
						i.OptimizationLoading = true
						m.itemsChan <- i
						m.prefConf = nil
						m.clearScreen = true
						// re-evaluate
						m.instanceChan <- i
					})
					initCmd = m.prefConf.Init()
					break
				}
			}
		case "enter":
			if len(m.table.SelectedRow()) == 0 {
				break
			}

			selectedInstanceID := m.table.SelectedRow()[0]
			for _, i := range m.items {
				if selectedInstanceID == *i.Instance.InstanceId {
					m.detailsPage = NewEc2InstanceDetail(i, func() {
						m.detailsPage = nil
					})
					initCmd = m.detailsPage.Init()
				}
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	cmd = tea.Batch(cmd, tickCmdWithDuration(time.Millisecond*50))
	if initCmd != nil {
		cmd = tea.Batch(cmd, initCmd)
	}
	return m, cmd
}

func (m *Ec2InstanceOptimizations) View() string {
	if m.clearScreen {
		m.clearScreen = false
		return ""
	}
	if m.detailsPage != nil {
		return m.detailsPage.View()
	}
	if m.prefConf != nil {
		return m.prefConf.View()
	}
	return baseStyle.Render(m.table.View()) + "\n\n" +
		"  ↑/↓: move\n" +
		"  enter: see details\n" +
		"  p: change preferences for one item\n" +
		"  q/ctrl+c: exit\n"
}

func (m *Ec2InstanceOptimizations) SendItem(item OptimizationItem) {
	m.itemsChan <- item
}
