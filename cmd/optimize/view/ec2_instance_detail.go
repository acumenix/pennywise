package view

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Ec2InstanceDetail struct {
	item        OptimizationItem
	close       func()
	deviceTable table.Model
	detailTable table.Model
	width       int
	height      int
}

func NewEc2InstanceDetail(item OptimizationItem, close func()) *Ec2InstanceDetail {
	deviceColumns := []table.Column{
		{Title: "DeviceID", Width: 30},
		{Title: "ResourceType", Width: 20},
		{Title: "Cost", Width: 10},
		{Title: "Saving", Width: 10},
	}
	deviceRows := []table.Row{
		{
			*item.Instance.InstanceId,
			"EC2 Instance",
			fmt.Sprintf("%.2f", item.CurrentCost),
			fmt.Sprintf("%.2f", item.CurrentCost-item.TargetCost),
		},
	}
	for _, v := range item.Instance.BlockDeviceMappings {
		deviceRows = append(deviceRows, table.Row{
			*v.Ebs.VolumeId,
			"EBS Volume",
			"",
			"",
		})
	}

	detailColumns := []table.Column{
		{Title: "Properties", Width: 30},
		{Title: "Provisioned", Width: 20},
		{Title: "Utilization", Width: 20},
		{Title: "Suggested", Width: 20},
	}
	detailRows := []table.Row{
		{
			"Instance ID",
			*item.Instance.InstanceId,
			"",
			"",
		},
		{
			"Region",
			item.Region,
			"",
			"",
		},
		{
			"Instance Type",
			string(item.Instance.InstanceType),
			"",
			item.TargetInstanceType,
		},
		{
			"vCPU",
			fmt.Sprintf("%v", *item.Instance.CpuOptions.CoreCount**item.Instance.CpuOptions.ThreadsPerCore),
			item.AvgCPUUsage,
			item.TargetCores,
		},
		{
			"Memory",
			item.CurrentMemory,
			item.AvgMemoryUsage,
			item.TargetMemory,
		},
		{
			"Bandwidth",
			item.CurrentNetworkPerformance,
			item.AvgNetworkBandwidth,
			item.TargetNetworkPerformance,
		},
		{
			"Total Cost (Monthly)",
			fmt.Sprintf("$%v", item.CurrentCost),
			"",
			fmt.Sprintf("$%v", item.TargetCost),
		},
		{
			"Total Saving (Monthly)",
			"$0",
			"",
			fmt.Sprintf("$%v", item.TotalSaving),
		},
	}

	model := Ec2InstanceDetail{
		item:  item,
		close: close,
		detailTable: table.New(
			table.WithColumns(detailColumns),
			table.WithRows(detailRows),
			table.WithFocused(false),
			table.WithHeight(8),
		),
		deviceTable: table.New(
			table.WithColumns(deviceColumns),
			table.WithRows(deviceRows),
			table.WithFocused(true),
			table.WithHeight(12),
		),
	}

	detailStyle := table.DefaultStyles()
	detailStyle.Header = detailStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	detailStyle.Selected = lipgloss.NewStyle()

	deviceStyle := table.DefaultStyles()
	deviceStyle.Header = deviceStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	deviceStyle.Selected = deviceStyle.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	model.detailTable.SetStyles(detailStyle)
	model.deviceTable.SetStyles(deviceStyle)
	return &model
}

func (m *Ec2InstanceDetail) Init() tea.Cmd { return nil }

func (m *Ec2InstanceDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd, detailCMD tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.deviceTable.SetWidth(m.width)
		m.detailTable.SetWidth(m.width)
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "esc":
			m.close()
		}
	}
	m.deviceTable, cmd = m.deviceTable.Update(msg)
	//m.detailTable, detailCMD = m.detailTable.Update(msg)
	return m, tea.Batch(detailCMD, cmd)
}

func (m *Ec2InstanceDetail) View() string {
	return baseStyle.Render(m.deviceTable.View()) + "\n" +
		baseStyle.Render(m.detailTable.View()) +
		helpStyle.Render(`
↑/↓: move
esc: back to ec2 instance list
q/ctrl+c: exit
`)
}
