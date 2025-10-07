package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"go.dalton.dog/batterup/internal/mlb"
	"go.dalton.dog/batterup/internal/styles"
)

// View enumerates the high-level screens in the TUI.
type ModelIndex int

const (
	viewSchedule ModelIndex = iota
	viewGame
)

// Model orchestrates the entire Bubble Tea program.
type Model struct {
	ctx    context.Context
	cancel context.CancelFunc

	curModel ModelIndex
	schedule ScheduleModel
	game     GameModel

	width  int
	height int
}

// NewAppModel constructs the Bubble Tea model.
func NewAppModel(client *mlb.Client) Model {
	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		ctx:      ctx,
		cancel:   cancel,
		curModel: viewSchedule,

		schedule: NewScheduleModel(client, ctx),
		game:     NewGameModel(client, ctx),
	}

	return m
}

// Init boots the initial commands for the program.
func (m Model) Init() tea.Cmd {
	return m.schedule.Init()
}

// Update reacts to incoming messages and user input.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds           []tea.Cmd
		gameCmd        tea.Cmd
		handledGameMsg bool
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.schedule.SetSize(msg.Width, msg.Height-2) // Account for header and footer
		m.game.SetSize(msg.Width, msg.Height-2)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			return m, tea.Quit
		case "esc", "q":
			if m.curModel == viewGame {
				m.curModel = viewSchedule
				m.game.SetActive(false)
				m.schedule.SetActive(true)
				return m, nil
			}
		}

	case openGameMsg:
		m.curModel = viewGame
		m.schedule.SetActive(false)
		m.game.SetActive(true)
		if m.width > 0 && m.height > 0 {
			m.game.SetSize(m.width, m.height-2)
		}
		m.game, gameCmd = m.game.Update(msg)
		handledGameMsg = true
	}

	if m.curModel != viewSchedule {
		m.schedule.SetActive(false)
	} else {
		m.schedule.SetActive(true)
	}

	if m.curModel == viewSchedule {
		var cmd tea.Cmd
		var outModel tea.Model
		outModel, cmd = m.schedule.Update(msg)
		m.schedule = outModel.(ScheduleModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	}

	if m.curModel == viewGame {
		if !handledGameMsg {
			m.game, gameCmd = m.game.Update(msg)
		}
		if gameCmd != nil {
			cmds = append(cmds, gameCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the entire screen for the current state.
func (m Model) View() string {
	header := styles.AppHeaderStyle.Width(m.width).Render("Batter Up!")
	footer := styles.AppHeaderStyle.Width(m.width).Render("https://github.com/daltonsw/batterup")

	var content string
	switch m.curModel {
	case viewSchedule:
		content = m.schedule.View()
	case viewGame:
		content = m.game.View()
	}

	if m.height <= 0 {
		return content
	}

	content = styles.MainContentWrapperStyle.Height(m.height - lipgloss.Height(header) - lipgloss.Height(footer)).Render(content)

	return lipgloss.JoinVertical(lipgloss.Center, header, content, footer)
}

func (m *Model) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
}

// openGameMsg instructs the root model to enter the game view.
type openGameMsg struct {
	GameID int
}
