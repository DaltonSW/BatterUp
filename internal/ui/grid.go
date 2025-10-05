package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"go.dalton.dog/batterup/internal/styles"
)

type GridItem string

// GridModel is essentially a List but in a grid instead of just a column.
// It doessn't (puresently) handle overflowing very well, though hopefully
// that shouldn't be much eof an issue...? There aren't *that* many games per day.
type GridModel struct {
	items  []GridItem
	cursor int

	width  int
	height int

	itemWidth    int
	itemHeight   int
	itemsPerRow  int
	itemsPerPage int
}

func NewGridModel() GridModel {
	model := GridModel{}

	return model
}

func (m GridModel) GetIndex() int {
	return m.cursor
}

func (m *GridModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.calculateLayout()
}

func (m *GridModel) SetItems(items []GridItem) {
	m.items = items

	m.itemWidth = 0
	m.itemHeight = 0
	for _, item := range m.items {
		itemStr := string(item)
		if w := lipgloss.Width(itemStr); w > m.itemWidth {
			m.itemWidth = w
		}
		if h := lipgloss.Height(itemStr); h > m.itemHeight {
			m.itemHeight = h
		}
	}

	if len(m.items) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	m.calculateLayout()
}

func (m *GridModel) SetCursor(pos int) {
	if len(m.items) == 0 {
		m.cursor = 0
		return
	}

	if pos < 0 {
		pos = 0
	}
	if pos >= len(m.items) {
		pos = len(m.items) - 1
	}

	m.cursor = pos
}

func (m GridModel) Init() tea.Cmd {
	return nil
}

func (m GridModel) Update(msg tea.Msg) (GridModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.calculateLayout()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "k", "up":
			if m.cursor >= m.itemsPerRow {
				m.cursor -= m.itemsPerRow
			}

		case "j", "down":
			if m.cursor+m.itemsPerRow < len(m.items) {
				m.cursor += m.itemsPerRow
			}

		case "h", "left":
			if m.cursor > 0 {
				m.cursor--
			}

		case "l", "right":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

func (m GridModel) View() string {
	var b strings.Builder

	selectedStyle := styles.ScheduleListCurr.
		Width(m.itemWidth).
		Height(m.itemHeight).Margin(0, 1, 0)

	normalStyle := styles.ScheduleListItem.
		Width(m.itemWidth).
		Height(m.itemHeight).Margin(0, 1, 1)

	rows := (len(m.items) + m.itemsPerRow - 1) / m.itemsPerRow

	for row := range rows {
		var rowItems []string

		for col := 0; col < m.itemsPerRow; col++ {
			idx := row*m.itemsPerRow + col

			if idx >= len(m.items) {
				// Empty placeholder
				emptyStyle := lipgloss.NewStyle().
					Width(m.itemWidth + 2).
					Height(m.itemHeight + 2)
				rowItems = append(rowItems, emptyStyle.Render(""))
				continue
			}

			var rendered string
			if idx == m.cursor {
				rendered = selectedStyle.Render(string(m.items[idx]))
			} else {
				rendered = normalStyle.Render(string(m.items[idx]))
			}

			rowItems = append(rowItems, rendered)
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, rowItems...) + "\n")
	}

	b.WriteString(styles.HelpTextStyle.Render("hjkl / ←↓↑→ to navigate • q to quit"))

	return b.String()
}

func (m *GridModel) calculateLayout() {
	// Calculate how many items fit per row with spacing
	// Each item takes itemWidth + 2 spaces (1 on each side)
	totalItemWidth := m.itemWidth + 2
	m.itemsPerRow = max(1, m.width/totalItemWidth)

	// Calculate items per page
	totalItemHeight := m.itemHeight + 2
	rows := max(1, (m.height-2)/totalItemHeight)
	m.itemsPerPage = m.itemsPerRow * rows
}
