package view

//
//import (
//	"fmt"
//	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
//	tea "github.com/charmbracelet/bubbletea"
//	"os"
//	"strings"
//	"time"
//)
//
//type Item struct {
//	Instance types.Instance
//	Region   string
//}
//
//type EC2InstanceList struct {
//	itemsChan chan Item
//	loading   bool
//
//	items    []Item
//	cursor   int
//	selected int
//}
//
//func NewEC2InstanceList() *EC2InstanceList {
//	return &EC2InstanceList{
//		itemsChan: make(chan Item, 1000),
//		loading:   true,
//		selected:  -1,
//	}
//}
//
//func (m *EC2InstanceList) Init() tea.Cmd {
//	return tickCmdWithDuration(time.Millisecond * 50)
//}
//
//func (m *EC2InstanceList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//	switch msg := msg.(type) {
//	case tickMsg:
//		for {
//			nothingToAdd := false
//			select {
//			case item := <-m.itemsChan:
//				m.items = append(m.items, item)
//			default:
//				nothingToAdd = true
//			}
//			if nothingToAdd {
//				break
//			}
//		}
//
//	case tea.KeyMsg:
//		switch msg.String() {
//		case "ctrl+c", "q":
//			os.Exit(0)
//		case "up", "k":
//			if m.cursor > 0 {
//				m.cursor--
//			}
//		case "down", "j":
//			if m.cursor < len(m.items)-1 {
//				m.cursor++
//			}
//		case "enter", " ":
//			if len(m.items) > 0 {
//				m.selected = m.cursor
//				return m, tea.Quit
//			}
//		}
//	}
//	return m, tickCmdWithDuration(time.Millisecond * 50)
//}
//
//func (m *EC2InstanceList) View() string {
//	if m.selected != -1 {
//		return ""
//	}
//
//	var b strings.Builder
//	pad := strings.Repeat(" ", padding)
//	b.WriteString("\n")
//	if m.loading {
//		loadingStr := ""
//		switch (time.Now().UnixMilli() / 100) % 3 {
//		case 0:
//			loadingStr = "-"
//		case 1:
//			loadingStr = "\\"
//		case 2:
//			loadingStr = "/"
//		}
//		b.WriteString(fmt.Sprintf(pad+loadingStr+" Found %d EC2 Instances.\n\n", len(m.items)))
//	} else {
//		b.WriteString(pad + "Finished loading all EC2 Instances.\n\n")
//	}
//
//	if len(m.items) > 0 {
//		b.WriteString(pad + "Which EC2 Instance you want to optimize?\n\n")
//
//		for i, choice := range m.items {
//			cursor := " "
//			if m.cursor == i {
//				cursor = ">"
//			}
//			text := fmt.Sprintf("%s%s %s - %s - %s - %s", pad, cursor, *choice.Instance.InstanceId, choice.Instance.InstanceType, *choice.Instance.PlatformDetails, choice.Region)
//			if m.cursor == i {
//				text = selectedStyle(text)
//			}
//			text += "\n"
//			b.WriteString(text)
//		}
//	}
//	b.WriteString("\n" + pad + "Press q to quit.\n")
//	return b.String()
//}
//
//func (m *EC2InstanceList) SendItem(item Item) {
//	m.itemsChan <- item
//}
//
//func (m *EC2InstanceList) Finished() {
//	m.loading = false
//}
//
//func (m *EC2InstanceList) SelectedItem() *Item {
//	if m.selected == -1 {
//		return nil
//	}
//	return &m.items[m.selected]
//}
