package view

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Ec2InstanceDetail struct {
	item  OptimizationItem
	close func()
	table table.Model
}

func NewEc2InstanceDetail(item OptimizationItem, close func()) *Ec2InstanceDetail {
	columns := []table.Column{
		{Title: "Properties", Width: 50},
		{Title: "Current", Width: 50},
		{Title: "After", Width: 50},
	}
	rows := []table.Row{
		{
			"Instance ID",
			*item.Instance.InstanceId,
			*item.Instance.InstanceId,
		},
		{
			"Instance Type",
			string(item.Instance.InstanceType),
			item.TargetInstanceType,
		},
		{
			"vCPU",
			fmt.Sprintf("%v", *item.Instance.CpuOptions.CoreCount**item.Instance.CpuOptions.ThreadsPerCore),
			item.TargetCores,
		},
		{
			"Memory",
			item.CurrentMemory,
			item.TargetMemory,
		},
		{
			"Bandwidth",
			item.CurrentNetworkPerformance,
			item.TargetNetworkPerformance,
		},
		{
			"Region",
			item.Region,
			item.Region,
		},
		{
			"Total Cost (Monthly)",
			fmt.Sprintf("$%v", item.CurrentCost),
			fmt.Sprintf("$%v", item.TargetCost),
		},
		{
			"Total Saving (Monthly)",
			"$0",
			fmt.Sprintf("$%v", item.TotalSaving),
		},
		{
			"Average Network Bandwidth",
			item.AvgNetworkBandwidth,
			"",
		},
		{
			"Average CPU Usage",
			item.AvgCPUUsage,
			"",
		},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
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

	return &Ec2InstanceDetail{
		item:  item,
		table: t,
		close: close,
	}
}

func (m *Ec2InstanceDetail) Init() tea.Cmd { return nil }

func (m *Ec2InstanceDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "esc":
			m.close()
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Ec2InstanceDetail) View() string {
	return baseStyle.Render(m.table.View()) + "\n\n" +
		"  ↑/↓: move\n" +
		"  esc: back to ec2 instance list\n" +
		"  q/ctrl+c: exit\n"
}
