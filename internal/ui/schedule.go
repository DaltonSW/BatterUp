package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"

	"go.dalton.dog/batterup/internal/mlb"
)

// ScheduleModel is a model that displays a given day
// of the MLB schedule. It allows for navigation around
// the games for that day, if any.
type ScheduleModel struct {
	client  *mlb.Client
	context context.Context

	date     time.Time
	games    []mlb.ScheduleGame
	loading  bool
	err      error
	selected int

	grid GridModel

	width  int
	height int

	initialized bool
	active      bool
}

type scheduleLoadedMsg struct {
	date  time.Time
	games []mlb.ScheduleGame
}

type scheduleFailedMsg struct {
	date time.Time
	err  error
}

type scheduleAutoRefreshMsg struct{}

func NewScheduleModel(client *mlb.Client, ctx context.Context) ScheduleModel {
	return ScheduleModel{
		client:  client,
		date:    time.Now(),
		active:  true,
		loading: true,
		context: ctx,

		grid: NewGridModel(),
	}
}

func (s ScheduleModel) Init() tea.Cmd {
	return s.load()
}

func (s *ScheduleModel) SetActive(active bool) {
	s.active = active
}

func (s *ScheduleModel) SetSize(width, height int) {
	s.width = width
	s.height = height

	s.grid.SetSize(width, height-1)
}

func (s ScheduleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !s.active {
			return s, nil
		}

		switch msg.String() {
		case "enter":
			if s.loading || len(s.games) == 0 {
				return s, nil
			}
			idx := s.grid.GetIndex()
			if idx < 0 || idx >= len(s.games) {
				return s, nil
			}
			s.selected = idx
			gameID := s.games[idx].GamePk
			return s, func() tea.Msg { return openGameMsg{GameID: gameID} }
		case "p", "P":
			s.date = s.date.AddDate(0, 0, -1)
			s.selected = 0
			s.grid.SetCursor(0)
			s.loading = true
			s.err = nil
			return s, s.load()
		case "n", "N":
			s.date = s.date.AddDate(0, 0, 1)
			s.selected = 0
			s.grid.SetCursor(0)
			s.loading = true
			s.err = nil
			return s, s.load()
		case "t", "T":
			today := time.Now()
			s.date = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
			s.selected = 0
			s.grid.SetCursor(0)
			s.loading = true
			s.err = nil
			return s, s.load()

		}
	case scheduleLoadedMsg:
		if !sameDay(msg.date, s.date) {
			return s, nil
		}
		s.initialized = true
		s.loading = false
		s.err = nil
		s.games = msg.games
		if s.selected >= len(s.games) {
			s.selected = max(len(s.games)-1, 0)
		}
		items := make([]GridItem, len(s.games))
		for idx, game := range s.games {
			items[idx] = GridItem(s.renderGame(game))
		}
		s.grid.SetItems(items)
		s.grid.SetCursor(s.selected)
		return s, tea.Tick(30*time.Second, func(time.Time) tea.Msg { return scheduleAutoRefreshMsg{} })
	case scheduleFailedMsg:
		if !sameDay(msg.date, s.date) {
			return s, nil
		}
		s.loading = false
		s.err = msg.err
	case scheduleAutoRefreshMsg:
		if s.viewingToday() {
			s.loading = true
			s.err = nil
			return s, s.load()
		}
		return s, nil
	}

	var cmd tea.Cmd
	s.grid, cmd = s.grid.Update(msg)
	s.selected = s.grid.GetIndex()
	return s, cmd
}

func (s *ScheduleModel) viewingToday() bool {
	return time.Now().Format("2006-01-02") == s.date.Format("2006-01-02")
}

func (s ScheduleModel) load() tea.Cmd {
	date := s.date
	return func() tea.Msg {
		resp, err := s.client.FetchSchedule(s.context, date)
		if err != nil {
			return scheduleFailedMsg{date: date, err: err}
		}
		games := []mlb.ScheduleGame{}
		if len(resp.Dates) > 0 {
			games = resp.Dates[0].Games
		}
		return scheduleLoadedMsg{date: date, games: games}
	}
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func (s ScheduleModel) View() string {
	var builder strings.Builder
	builder.WriteString(lipgloss.NewStyle().Bold(true).AlignHorizontal(lipgloss.Center).PaddingTop(1).Render(s.date.Format("Monday, January 2, 2006") + "\n<< [P]rev | [T]oday | [N]ext >>"))
	builder.WriteString("\n\n")

	switch {
	case s.loading && len(s.games) == 0:
		builder.WriteString("Loading schedule…")
	case s.err != nil:
		builder.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Render("Error loading schedule: " + s.err.Error()))
	case len(s.games) == 0:
		builder.WriteString("No games scheduled")
	default:
		builder.WriteString(s.grid.View())
	}

	return builder.String()
}

func (s ScheduleModel) renderGame(game mlb.ScheduleGame) string {
	linescore := game.Linescore
	awayRuns, awayHits, awayErrors := "-", "-", "-"
	homeRuns, homeHits, homeErrors := "-", "-", "-"
	if linescore != nil {
		awayRuns = fmt.Sprintf("%d", linescore.Teams.Away.Runs)
		awayHits = fmt.Sprintf("%d", linescore.Teams.Away.Hits)
		awayErrors = fmt.Sprintf("%d", linescore.Teams.Away.Errors)
		homeRuns = fmt.Sprintf("%d", linescore.Teams.Home.Runs)
		homeHits = fmt.Sprintf("%d", linescore.Teams.Home.Hits)
		homeErrors = fmt.Sprintf("%d", linescore.Teams.Home.Errors)
	}

	gameTable := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).BorderBottom(false).
		Headers("Teams                ", "R ", "H ", "E ").
		Row(
			"  "+formatScheduleTeam(game.Teams.Away),
			awayRuns,
			awayHits,
			awayErrors,
		).
		Row(
			"@ "+formatScheduleTeam(game.Teams.Home),
			homeRuns,
			homeHits,
			homeErrors,
		)

	status := lipgloss.NewStyle().Reverse(true).Render(describeGameStatus(game))

	table := gameTable.Render()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		status,
		table,
	)
}

func describeGameStatus(game mlb.ScheduleGame) string {
	switch game.Status.AbstractGameCode {
	case "P":
		if game.DoubleHeader == "Y" && game.GameNumber > 1 {
			return fmt.Sprintf("Game %d", game.GameNumber)
		}
		if game.Status.StartTimeTBD {
			return "Start time TBD"
		}
		return game.GameDate.Local().Format("Starts @ 3:04 PM MST")
	case "L":
		if game.Linescore != nil {
			state := strings.TrimSpace(game.Linescore.InningState + " " + game.Linescore.CurrentInningOrdinal)
			if state == "" {
				state = game.Status.DetailedState
			}
			if state == "" {
				state = "In Progress"
			}
			return state
		}
		return game.Status.DetailedState
	case "F":
		if game.Status.Reason != "" {
			return fmt.Sprintf("%s | %s", game.Status.DetailedState, game.Status.Reason)
		}
		return game.Status.DetailedState
	default:
		return game.Status.DetailedState
	}
}

func formatScheduleTeam(team mlb.ScheduleTeam) string {
	return fmt.Sprintf("%s (%d-%d)", team.Team.TeamName, team.LeagueRecord.Wins, team.LeagueRecord.Losses)
}
