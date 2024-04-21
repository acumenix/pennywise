package view

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	preferences2 "github.com/kaytu-io/pennywise/cmd/optimize/preferences"
	"github.com/kaytu-io/pennywise/pkg/api/wastage"
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
	Volumes             []types.Volume
	Region              string
	OptimizationLoading bool

	Preferences               []preferences2.PreferenceItem
	RightSizingRecommendation wastage.RightSizingRecommendation
}

type Ec2InstanceOptimizations struct {
	itemsChan chan OptimizationItem

	table table.Model
	items []OptimizationItem
	help  HelpView

	detailsPage *Ec2InstanceDetail
	prefConf    *PreferencesConfiguration

	clearScreen  bool
	instanceChan chan OptimizationItem

	Width  int
	height int

	tableHeight int
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
		itemsChan: make(chan OptimizationItem, 1000),
		table:     t,
		items:     nil,
		help: HelpView{
			lines: []string{
				"↑/↓: move",
				"enter: see details",
				"p: change preferences for one item",
				"P: change preferences for all items",
				"q/ctrl+c: exit",
			},
		},
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
	case tea.WindowSizeMsg:
		m.UpdateResponsive()

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
					platform := ""
					if i.Instance.PlatformDetails != nil {
						platform = *i.Instance.PlatformDetails
					}
					row := table.Row{
						*i.Instance.InstanceId,
						string(i.Instance.InstanceType),
						i.Region,
						platform,
						i.RightSizingRecommendation.TargetInstanceType,
						fmt.Sprintf("$%.2f", i.RightSizingRecommendation.Saving),
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
		case "P":
			if len(m.table.SelectedRow()) == 0 {
				break
			}

			m.prefConf = NewPreferencesConfiguration(preferences2.DefaultPreferences(), func(items []preferences2.PreferenceItem) {
				for _, i := range m.items {
					i.Preferences = items
					i.OptimizationLoading = true
					m.itemsChan <- i
					m.instanceChan <- i
				}
				m.prefConf = nil
				m.clearScreen = true
			})
			initCmd = m.prefConf.Init()
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
	return baseStyle.Render(m.table.View()) + "\n" +
		m.help.String()
}

func (m *Ec2InstanceOptimizations) SendItem(item OptimizationItem) {
	m.itemsChan <- item
}

func (m *Ec2InstanceOptimizations) UpdateResponsive() {
	defer func() {
		m.table.SetHeight(m.tableHeight - 4)
	}()
	m.tableHeight = 5
	m.help.SetHeight(m.help.MinHeight())

	checkResponsive := func() bool {
		return m.height >= m.help.height+m.tableHeight && m.help.IsResponsive() && m.tableHeight >= 5
	}

	if !checkResponsive() {
		return // nothing to do
	}

	for m.tableHeight < 9 {
		m.tableHeight++
		if !checkResponsive() {
			m.tableHeight--
			return
		}
	}

	for m.help.height < m.help.MaxHeight() {
		m.help.SetHeight(m.help.height + 1)
		if !checkResponsive() {
			m.help.SetHeight(m.help.height - 1)
			return
		}
	}

	for m.tableHeight < 30 {
		m.tableHeight++
		if !checkResponsive() {
			m.tableHeight--
			return
		}
	}
}

func (m *Ec2InstanceOptimizations) SetHeight(height int) {
	m.height = height
}

func (m *Ec2InstanceOptimizations) MinHeight() int {
	return m.help.MinHeight() + 5
}

func (m *Ec2InstanceOptimizations) PreferredMinHeight() int {
	return 10
}

func (m *Ec2InstanceOptimizations) MaxHeight() int {
	return m.help.MaxHeight() + 30
}

func (m *Ec2InstanceOptimizations) IsResponsive() bool {
	return m.height >= m.MinHeight()
}
