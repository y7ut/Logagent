package table

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 获取一个map的全部的key
func MapKeys[Key comparable, T any](m map[Key]T) []Key {
	s := make([]Key, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	return s
}

// 表格的搜索query like
type TableSearchQuery struct {
	Search string
	Value  string
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#6EAF23"))

type HermersTable struct {
	keyMap keyMap
	table  table.Model
	help   help.Model
	board  string
}

func (m HermersTable) Init() tea.Cmd { return nil }

func (m HermersTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the help menu it can gracefully truncate
		// its view as needed.
		m.help.Width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Focus):
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, cmd
		case key.Matches(msg, m.keyMap.Enter):
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
				// TODO： 刷新表格内部数据和标题
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

type keyMap struct {
	base  table.KeyMap
	Enter key.Binding
	Quit  key.Binding
	Help  key.Binding
	Focus key.Binding
}

func DefaultTableKeyMap() keyMap {
	keyMapDefault := keyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "checkout item"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("h", "ctrl+h"),
			key.WithHelp("h", "help"),
		),
		Focus: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "focus"),
			// 让表格锁定住不能动
		),
		base: table.DefaultKeyMap(),
	}
	return keyMapDefault
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.base.LineUp, k.base.LineDown, k.Enter, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.base.LineUp, k.base.LineDown},    // first column
		{k.base.GotoTop, k.base.GotoBottom}, // second column
		{k.base.PageUp, k.base.PageDown},
		{k.Quit, k.Focus},
	}
}

func (m HermersTable) View() string {
	tableHelp := m.help.View(m.keyMap)
	return baseStyle.Render(m.table.View()) + "\n" + m.board + "\n" + tableHelp + "\n"
}

func newBaseTable(columns []table.Column, rows []table.Row) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(8),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	return t
}

func Create(data Grid, query ...TableSearchQuery) HermersTable {
	columns, rows := data.Render()
	var board string = fmt.Sprintf("当前收集日志文件%d个", len(rows))
	if len(query) > 0 && query[0].Search != "" && query[0].Value != "" {

		for k, v := range columns {
			if v.Title == query[0].Search {

				searchedRows := make([]table.Row, 0, len(rows))
				for _, v := range rows {
					if v[k] == query[0].Value {
						searchedRows = append(searchedRows, v)
					}
				}
				board += fmt.Sprintf("正在搜索标题是%s并且值是%s的记录", query[0].Search, query[0].Value)
				rows = searchedRows

			}
		}

	}

	m := HermersTable{DefaultTableKeyMap(), newBaseTable(columns, rows), help.New(), board}

	return m
}
