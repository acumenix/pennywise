package view

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kaytu-io/pennywise/pkg/hash"
)

type Ec2InstanceDetail struct {
	item             OptimizationItem
	close            func()
	deviceTable      table.Model
	detailTable      table.Model
	deviceProperties map[string][]table.Row
	width            int
	height           int
	selectedDevice   string
	help             HelpView
}

func ExtractProperties(item OptimizationItem) map[string][]table.Row {
	res := map[string][]table.Row{
		*item.Instance.InstanceId: {
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
				item.RightSizingRecommendation.TargetInstanceType,
			},
			{
				"vCPU",
				fmt.Sprintf("%v", *item.Instance.CpuOptions.CoreCount**item.Instance.CpuOptions.ThreadsPerCore),
				item.RightSizingRecommendation.AvgCPUUsage,
				item.RightSizingRecommendation.TargetCores,
			},
			{
				"Memory",
				item.RightSizingRecommendation.CurrentMemory,
				item.RightSizingRecommendation.MaxMemoryUsagePercentage,
				item.RightSizingRecommendation.TargetMemory,
			},
			{
				"Network Bandwidth",
				item.RightSizingRecommendation.CurrentNetworkPerformance,
				item.RightSizingRecommendation.AvgNetworkBandwidth,
				item.RightSizingRecommendation.TargetNetworkPerformance,
			},
			{
				"EBS Bandwidth",
				item.RightSizingRecommendation.CurrentEBSBandwidth,
				item.RightSizingRecommendation.AvgEBSBandwidth,
				item.RightSizingRecommendation.TargetEBSBandwidth,
			},
			{
				"Total Cost (Monthly)",
				fmt.Sprintf("$%.2f", item.RightSizingRecommendation.CurrentCost),
				"",
				fmt.Sprintf("$%.2f", item.RightSizingRecommendation.TargetCost),
			},
			{
				"Total Saving (Monthly)",
				"",
				"",
				fmt.Sprintf("$%.2f", item.RightSizingRecommendation.Saving),
			},
		},
	}

	for _, v := range item.Volumes {
		vid := hash.HashString(*v.VolumeId)
		volumeSize := ""
		volumeThroughput := ""
		targetThroughput := "Not applicable"
		volumeIops := ""
		targetIops := "Not applicable"
		if v.Size != nil {
			volumeSize = fmt.Sprintf("%d GB", *v.Size)
		}
		if v.Throughput != nil {
			volumeThroughput = fmt.Sprintf("%d MB/s", *v.Throughput/8.0)
		}
		if vt := item.RightSizingRecommendation.VolumesTargetTypes[vid]; vt == "gp3" {
			targetThroughput = fmt.Sprintf("%.2f MB/s", item.RightSizingRecommendation.VolumesTargetThroughput[vid]/8.0)
		}

		if v.Iops != nil {
			volumeIops = fmt.Sprintf("%d", *v.Iops)
		}
		if vt := item.RightSizingRecommendation.VolumesTargetTypes[vid]; vt == "io1" || vt == "io2" || vt == "gp3" {
			targetIops = fmt.Sprintf("%d", item.RightSizingRecommendation.VolumesTargetIOPS[vid])
		}
		res[*v.VolumeId] = []table.Row{
			{
				"Volume Type",
				string(v.VolumeType),
				"",
				string(item.RightSizingRecommendation.VolumesTargetTypes[vid]),
			},
			{
				"Size",
				volumeSize,
				"",
				fmt.Sprintf("%d GB", item.RightSizingRecommendation.VolumesTargetSizes[vid]),
			},
			{
				"IOPS",
				volumeIops,
				fmt.Sprintf("Avg: %.2f", item.RightSizingRecommendation.VolumesIOPSUtilization[vid]),
				targetIops,
			},
			{
				"Throughput",
				volumeThroughput,
				fmt.Sprintf("Avg: %.2f MB/s", item.RightSizingRecommendation.VolumesThroughputUtilization[vid]/8.0),
				targetThroughput,
			},
			{
				"Total Cost (Monthly)",
				fmt.Sprintf("$%v", item.RightSizingRecommendation.VolumesCurrentCosts[vid]),
				"",
				fmt.Sprintf("$%v", item.RightSizingRecommendation.VolumesTargetCosts[vid]),
			},
			{
				"Total Saving (Monthly)",
				"",
				"",
				fmt.Sprintf("$%v", item.RightSizingRecommendation.VolumesSaving[vid]),
			},
		}
	}

	return res
}

func NewEc2InstanceDetail(item OptimizationItem, close func()) *Ec2InstanceDetail {
	deviceColumns := []table.Column{
		{Title: "DeviceID", Width: 30},
		{Title: "ResourceType", Width: 20},
		{Title: "Cost", Width: 10},
		{Title: "Potential Savings", Width: 20},
	}
	deviceRows := []table.Row{
		{
			*item.Instance.InstanceId,
			"EC2 Instance",
			fmt.Sprintf("$%.2f", item.RightSizingRecommendation.CurrentCost),
			fmt.Sprintf("$%.2f", item.RightSizingRecommendation.CurrentCost-item.RightSizingRecommendation.TargetCost),
		},
	}
	for _, v := range item.Instance.BlockDeviceMappings {
		saving := item.RightSizingRecommendation.VolumesCurrentCosts[hash.HashString(*v.Ebs.VolumeId)] - item.RightSizingRecommendation.VolumesTargetCosts[hash.HashString(*v.Ebs.VolumeId)]
		deviceRows = append(deviceRows, table.Row{
			*v.Ebs.VolumeId,
			"EBS Volume",
			fmt.Sprintf("$%.2f", item.RightSizingRecommendation.VolumesCurrentCosts[hash.HashString(*v.Ebs.VolumeId)]),
			fmt.Sprintf("$%.2f", saving),
		})
	}

	detailColumns := []table.Column{
		{Title: "Properties", Width: 30},
		{Title: "Current", Width: 20},
		{Title: "Utilization", Width: 20},
		{Title: "Suggested", Width: 20},
	}

	model := Ec2InstanceDetail{
		item:  item,
		close: close,
		detailTable: table.New(
			table.WithColumns(detailColumns),
			table.WithFocused(false),
			table.WithHeight(1),
		),
		deviceTable: table.New(
			table.WithColumns(deviceColumns),
			table.WithRows(deviceRows),
			table.WithFocused(true),
			table.WithHeight(len(deviceRows)),
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
	model.deviceProperties = ExtractProperties(item)
	model.help = HelpView{
		lines: []string{
			"↑/↓: move",
			"esc: back to ec2 instance list",
			"q/ctrl+c: exit",
		},
		height: 0,
	}
	return &model
}

func (m *Ec2InstanceDetail) Init() tea.Cmd { return nil }

func (m *Ec2InstanceDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd, detailCMD tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.deviceTable.SetWidth(m.width)
		m.detailTable.SetWidth(m.width)
		m.SetHeight(m.height)
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "esc":
			m.close()
		}
	}
	m.deviceTable, cmd = m.deviceTable.Update(msg)
	if m.deviceTable.SelectedRow() != nil {
		if m.selectedDevice != m.deviceTable.SelectedRow()[0] {
			m.selectedDevice = m.deviceTable.SelectedRow()[0]
			m.detailTable.SetRows(m.deviceProperties[m.selectedDevice])
			m.detailTable.SetHeight(len(m.deviceProperties[m.selectedDevice]))
			m.SetHeight(m.height)
		}
	}
	//m.detailTable, detailCMD = m.detailTable.Update(msg)
	return m, tea.Batch(detailCMD, cmd)
}

func (m *Ec2InstanceDetail) View() string {
	return baseStyle.Render(m.deviceTable.View()) + "\n" +
		baseStyle.Render(m.detailTable.View()) + "\n" +
		m.help.String()
}

func (m *Ec2InstanceDetail) IsResponsive() bool {
	return m.height >= m.MinHeight()
}

func (m *Ec2InstanceDetail) SetHeight(height int) {
	m.height = height
	m.help.SetHeight(m.height - (len(m.detailTable.Rows()) + 4 + len(m.deviceTable.Rows()) + 4))
}

func (m *Ec2InstanceDetail) MinHeight() int {
	return len(m.detailTable.Rows()) + 4 + len(m.deviceTable.Rows()) + 4 + m.help.MinHeight()
}
