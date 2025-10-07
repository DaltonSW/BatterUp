package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"go.dalton.dog/batterup/internal/mlb"
)

// GameModel manages live game state and polling.
type GameModel struct {
	client  *mlb.Client
	context context.Context

	width  int
	height int

	gameID  int
	feed    *mlb.GameFeed
	err     error
	loading bool
	active  bool

	requestID int

	playViews       []playView
	playLines       []playLine
	playLineOffsets []int
	playsOffset     int
	playsHeight     int
	selectedPlay    int
	selectedAtBat   int
}

type gameLoadedMsg struct {
	id     int
	gameID int
	feed   *mlb.GameFeed
}

type gameFailedMsg struct {
	id     int
	gameID int
	err    error
}

type gamePollMsg struct{}

func NewGameModel(client *mlb.Client, ctx context.Context) GameModel {
	return GameModel{
		client:  client,
		context: ctx,
	}
}

func (g *GameModel) SetSize(width, height int) {
	if height < 1 {
		height = 1
	}
	g.width = width
	g.height = height - 2

	g.playsHeight = max(height-10, 5)
	g.scrollToSelected()
}

func (g GameModel) Init() tea.Cmd {
	return nil
}

func (g *GameModel) SetActive(active bool) {
	g.active = active
	if !active {
		g.loading = false
	}
}

func (g GameModel) Update(msg tea.Msg) (GameModel, tea.Cmd) {
	switch msg := msg.(type) {
	case openGameMsg:
		if !g.active {
			return g, nil
		}
		g.gameID = msg.GameID
		g.feed = nil
		g.err = nil
		g.resetPlayState()
		g.loading = msg.GameID != 0
		if g.gameID == 0 {
			return g, nil
		}
		g.requestID++
		return g, g.fetch()
	case tea.KeyMsg:
		if !g.active {
			return g, nil
		}
		switch msg.String() {
		case "j", "down":
			g.moveSelection(1)
		case "k", "up":
			g.moveSelection(-1)
		case "pgdown":
			g.moveSelection(g.pageDelta())
		case "pgup":
			g.moveSelection(-g.pageDelta())
		}
	case tea.WindowSizeMsg:
		if g.active {
			g.SetSize(msg.Width, msg.Height)
		}
	case gameLoadedMsg:
		if msg.id != g.requestID || msg.gameID != g.gameID {
			return g, nil
		}
		if !g.active {
			return g, nil
		}
		g.loading = false
		g.err = nil
		g.feed = msg.feed
		g.refreshViewport()
		wait := msg.feed.MetaData.Wait
		if wait == 0 {
			wait = 10
		}
		return g, tea.Tick(time.Duration(wait)*time.Second, func(time.Time) tea.Msg { return gamePollMsg{} })
	case gameFailedMsg:
		if msg.id != g.requestID || msg.gameID != g.gameID {
			return g, nil
		}
		if !g.active {
			return g, nil
		}
		g.loading = false
		g.err = msg.err
	case gamePollMsg:
		if !g.active || g.gameID == 0 {
			return g, nil
		}
		g.requestID++
		g.loading = true
		return g, g.fetch()
	}
	return g, nil
}

func (g GameModel) fetch() tea.Cmd {
	if g.gameID == 0 || g.client == nil {
		return nil
	}

	ctx := g.context
	if ctx == nil {
		ctx = context.Background()
	}

	requestID := g.requestID
	gameID := g.gameID
	client := g.client

	return func() tea.Msg {
		feed, err := client.FetchGame(ctx, gameID)
		if err != nil {
			return gameFailedMsg{id: requestID, gameID: gameID, err: err}
		}
		return gameLoadedMsg{id: requestID, gameID: gameID, feed: feed}
	}
}

func (g GameModel) View() string {
	if g.gameID == 0 {
		return "Select a game to view"
	}
	if g.loading && g.feed == nil {
		return "Loading game…"
	}
	if g.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Render("Error loading game: " + g.err.Error())
	}
	if g.feed == nil {
		return ""
	}

	switch g.feed.GameData.Status.AbstractGameCode {
	case "P":
		return g.renderPreview()
	default:
		return g.renderLive()
	}
}
