package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/spinner"

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
	viewStandings
)

// Model orchestrates the entire Bubble Tea program.
type Model struct {
	ctx    context.Context
	cancel context.CancelFunc

	client *mlb.Client

	curModel ModelIndex
	schedule ScheduleModel
	// game      GameModel
	// standings StandingsModel

	spinner   spinner.Model
	helpModel help.Model

	width  int
	height int
}

// New constructs the Bubble Tea model.
func NewAppModel(client *mlb.Client) Model {
	ctx, cancel := context.WithCancel(context.Background())

	spinner := spinner.New()

	m := Model{
		ctx:      ctx,
		cancel:   cancel,
		client:   client,
		curModel: viewSchedule,

		spinner:   spinner,
		helpModel: help.New(),

		schedule: NewScheduleModel(client, ctx),
		// game:            NewGameModel(client),
		// standings:       NewStandingsModel(client).Year()),
	}
	return m
}

// Init boots the initial commands for the program.
func (m Model) Init() tea.Cmd {
	return m.schedule.Init()
}

// Update reacts to incoming messages and user input.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.schedule.SetSize(msg.Width, msg.Height-2) // Account for header and footer
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			return m, tea.Quit
		}
	}

	// Update sub-models (inactive models ignore key events)
	m.schedule, cmd = m.schedule.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if m.isLoading() {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) isLoading() bool {
	return m.schedule.loading
}

// View renders the entire screen for the current state.
func (m Model) View() string {
	header := styles.AppHeaderStyle.Width(m.width).Render("Batter Up!")
	footer := styles.AppHeaderStyle.Width(m.width).Render("https://github.com/daltonsw/batterup")

	mainHeight := m.height
	if mainHeight > 0 {
		mainHeight--
	}
	var content string
	switch m.curModel {
	case viewSchedule:
		content = m.schedule.View()
		// case viewGame:
		// 	content = m.game.View()
		// case viewStandings:
		// 	content = m.standings.View()
	}

	if m.height <= 0 {
		return content
	}

	return lipgloss.JoinVertical(lipgloss.Center, header, content, footer)
}

func (m *Model) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
}

// openGameMsg instructs the root model to enter the game view.
// type openGameMsg struct {
// 	GameID int
// }
