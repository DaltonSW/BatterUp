package ui

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"go.dalton.dog/batterup/internal/mlb"
	"go.dalton.dog/batterup/internal/styles"
)

type GameModel struct {
	client *mlb.Client
	logger *log.Logger

	width  int
	height int

	gameID  int
	feed    *mlb.GameFeed
	err     error
	loading bool
	active  bool

	requestID int

	playsLines  []string
	playsOffset int
	playsHeight int
	playsWidth  int
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

func newGameModel(client *mlb.Client, logger *log.Logger) GameModel {
	return GameModel{
		client: client,
		logger: logger,
	}
}

func (g *GameModel) SetSize(width, height int) {
	if height < 1 {
		height = 1
	}
	g.width = width
	g.height = height

	availWidth := max(width-4, 20)
	availHeight := max(height-10, 5)

	g.playsWidth = availWidth
	g.playsHeight = availHeight
	g.enforcePlayOffset()
}

func (g *GameModel) Start(gameID int) {
	g.gameID = gameID
	g.feed = nil
	g.err = nil
	g.loading = true
	g.active = true
	g.requestID++
	g.playsLines = nil
	g.playsOffset = 0
}

func (g *GameModel) Init(ctx context.Context) tea.Cmd {
	if g.gameID == 0 {
		return nil
	}
	return g.fetch(ctx)
}

func (g *GameModel) SetActive(active bool) {
	g.active = active
	if !active {
		g.loading = false
	}
}

func (g *GameModel) Update(ctx context.Context, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !g.active {
			return nil
		}
		switch msg.String() {
		case "j", "down":
			g.scrollPlays(1)
		case "k", "up":
			g.scrollPlays(-1)
		case "pgdown":
			g.scrollPlays(g.playsHeight)
		case "pgup":
			g.scrollPlays(-g.playsHeight)
		}
	case tea.WindowSizeMsg:
		if g.active {
			g.SetSize(msg.Width, msg.Height)
		}
	case gameLoadedMsg:
		if msg.id != g.requestID || msg.gameID != g.gameID {
			return nil
		}
		if !g.active {
			return nil
		}
		g.loading = false
		g.err = nil
		g.feed = msg.feed
		g.refreshViewport()
		wait := msg.feed.MetaData.Wait
		if wait == 0 {
			wait = 10
		}
		return tea.Tick(time.Duration(wait)*time.Second, func(time.Time) tea.Msg { return gamePollMsg{} })
	case gameFailedMsg:
		if msg.id != g.requestID || msg.gameID != g.gameID {
			return nil
		}
		if !g.active {
			return nil
		}
		g.loading = false
		g.err = msg.err
	case gamePollMsg:
		if !g.active {
			return nil
		}
		return g.fetch(ctx)
	}
	return nil
}

func (g *GameModel) fetch(ctx context.Context) tea.Cmd {
	g.loading = true
	currentID := g.requestID
	gameID := g.gameID
	return func() tea.Msg {
		feed, err := g.client.FetchGame(ctx, gameID)
		if err != nil {
			return gameFailedMsg{id: currentID, gameID: gameID, err: err}
		}
		return gameLoadedMsg{id: currentID, gameID: gameID, feed: feed}
	}
}

func (g *GameModel) View() string {
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
	case "L":
		return g.renderLive()
	case "F":
		return g.renderFinished()
	default:
		return g.renderLive()
	}
}

func (g *GameModel) refreshViewport() {
	if g.feed == nil {
		g.playsLines = nil
		g.playsOffset = 0
		return
	}
	plays := renderAllPlays(g.feed.LiveData.Plays.AllPlays)
	if plays == "" {
		g.playsLines = nil
		g.playsOffset = 0
		return
	}
	g.playsLines = strings.Split(plays, "\n")
	g.enforcePlayOffset()
}

func (g *GameModel) renderPlaysView() string {
	if len(g.playsLines) == 0 {
		return ""
	}
	if g.playsHeight <= 0 {
		return strings.Join(g.playsLines, "\n")
	}

	start := max(g.playsOffset, 0)

	maxPlays := g.maxPlaysOffset()
	if start > maxPlays {
		start = maxPlays
	}

	end := min(start+g.playsHeight, len(g.playsLines))
	return strings.Join(g.playsLines[start:end], "\n")
}

func (g *GameModel) scrollPlays(delta int) {
	if len(g.playsLines) == 0 {
		return
	}
	if g.playsHeight <= 0 {
		return
	}
	g.playsOffset += delta
	g.enforcePlayOffset()
}

func (g *GameModel) enforcePlayOffset() {
	if g.playsHeight <= 0 {
		g.playsOffset = 0
		return
	}
	max := g.maxPlaysOffset()
	if max < 0 {
		max = 0
	}
	if g.playsOffset > max {
		g.playsOffset = max
	}
	if g.playsOffset < 0 {
		g.playsOffset = 0
	}
}

func (g *GameModel) maxPlaysOffset() int {
	if g.playsHeight <= 0 {
		return 0
	}
	if len(g.playsLines) <= g.playsHeight {
		return 0
	}
	return len(g.playsLines) - g.playsHeight
}

func (g *GameModel) renderPreview() string {
	teams := g.feed.GameData.Teams
	venue := g.feed.GameData.Venue
	probables := g.feed.GameData.ProbablePitchers
	box := g.feed.LiveData.Boxscore

	awayLines := previewTeamLines(teams.Away, probables.Away, box.Teams.Away.Players)
	homeLines := previewTeamLines(teams.Home, probables.Home, box.Teams.Home.Players)

	startTime := "Start time TBD"
	if !g.feed.GameData.Status.StartTimeTBD {
		if t, err := time.Parse(time.RFC3339, g.feed.GameData.Datetime.DateTime); err == nil {
			startTime = t.Local().Format("Monday, January 2, 2006 3:04 PM")
		}
	}

	middle := fmt.Sprintf("%s\n%s\n%s, %s",
		startTime,
		venue.Name,
		venue.Location.City,
		venue.Location.StateAbbrev,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		columnStyle.Render(strings.Join(awayLines, "\n")),
		columnStyle.Render(middle),
		columnStyle.Render(strings.Join(homeLines, "\n")),
	)
}

func previewTeamLines(team mlb.GameTeam, probable *mlb.PersonRef, roster map[string]mlb.BoxscorePlayer) []string {
	lines := []string{fmt.Sprintf("%s", team.TeamName), fmt.Sprintf("(%d-%d)", team.Record.Wins, team.Record.Losses)}
	if probable != nil {
		key := fmt.Sprintf("ID%d", probable.ID)
		if player, ok := roster[key]; ok {
			lines = append(lines, "", player.Person.FullName)
			lines = append(lines, fmt.Sprintf("#%s", player.JerseyNumber))
			lines = append(lines, fmt.Sprintf("%d-%d", player.SeasonStats.Pitching.Wins, player.SeasonStats.Pitching.Losses))
			lines = append(lines, fmt.Sprintf("%s ERA %d K", player.SeasonStats.Pitching.ERA, player.SeasonStats.Pitching.StrikeOuts))
		}
	} else {
		lines = append(lines, "", "Probable: TBD")
	}
	return lines
}

func (g *GameModel) renderLive() string {
	linescore := g.feed.LiveData.Linescore
	teams := g.feed.GameData.Teams

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		renderInning(linescore),
		countStyle.Render(renderCount(linescore)),
		basesStyle.Render(renderBases(linescore)),
		lipgloss.NewStyle().PaddingLeft(2).Render(renderLineScoreTable(linescore, teams)),
	)

	matchup := renderMatchup(g.feed.LiveData, teams)
	atBat := renderAtBat(g.feed.LiveData.Plays.CurrentPlay)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		lipgloss.NewStyle().Margin(1, 0).Render(matchup),
		atBat,
		"",
		lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Width(g.playsWidth).Render(g.renderPlaysView()),
	)

	return content
}

func (g *GameModel) renderFinished() string {
	linescore := g.feed.LiveData.Linescore
	teams := g.feed.GameData.Teams
	decisions := g.feed.LiveData.Decisions
	box := g.feed.LiveData.Boxscore

	score := fmt.Sprintf("%s %d - %s %d",
		teams.Away.Abbreviation, linescore.Teams.Away.Runs,
		teams.Home.Abbreviation, linescore.Teams.Home.Runs,
	)

	dec := renderDecisions(decisions, box, g.feed.GameData.Teams)

	table := renderLineScoreTable(linescore, teams)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(score),
		table,
		"",
		dec,
	)
}

func renderLineScoreTable(linescore mlb.LiveLineScore, teams mlb.GameTeams) string {
	totalInnings := max(len(linescore.Innings), 9)

	header := []string{"  "}
	for i := 1; i <= totalInnings; i++ {
		header = append(header, fmt.Sprintf("%2d", i))
	}
	header = append(header, " R", " H", " E")

	awayLine := []string{teams.Away.Abbreviation}
	homeLine := []string{teams.Home.Abbreviation}

	for i := range totalInnings {
		var awayRuns, homeRuns string
		if i < len(linescore.Innings) {
			if linescore.Innings[i].Away.Runs != nil {
				awayRuns = fmt.Sprintf("%2d", *linescore.Innings[i].Away.Runs)
			} else if linescore.CurrentInning > i+1 {
				awayRuns = " X"
			} else {
				awayRuns = "  "
			}
			if linescore.Innings[i].Home.Runs != nil {
				homeRuns = fmt.Sprintf("%2d", *linescore.Innings[i].Home.Runs)
			} else if linescore.CurrentInning > i+1 {
				homeRuns = " X"
			} else {
				homeRuns = "  "
			}
		} else {
			awayRuns = "  "
			homeRuns = "  "
		}
		awayLine = append(awayLine, awayRuns)
		homeLine = append(homeLine, homeRuns)
	}

	awayLine = append(awayLine,
		fmt.Sprintf("%2d", linescore.Teams.Away.Runs),
		fmt.Sprintf("%2d", linescore.Teams.Away.Hits),
		fmt.Sprintf("%2d", linescore.Teams.Away.Errors),
	)
	homeLine = append(homeLine,
		fmt.Sprintf("%2d", linescore.Teams.Home.Runs),
		fmt.Sprintf("%2d", linescore.Teams.Home.Hits),
		fmt.Sprintf("%2d", linescore.Teams.Home.Errors),
	)

	table := lipgloss.JoinVertical(lipgloss.Left,
		strings.Join(header, " "),
		strings.Join(awayLine, " "),
		strings.Join(homeLine, " "),
	)

	return tableStyle.Render(table)
}

func renderMatchup(live mlb.LiveData, teams mlb.GameTeams) string {
	current := live.Plays.CurrentPlay
	box := live.Boxscore

	pitcher, pitchTeam := lookupPlayer(current.Matchup.Pitcher.ID, box, teams)
	batter, batTeam := lookupPlayer(current.Matchup.Batter.ID, box, teams)

	pitchTeam = safeTeam(pitchTeam)
	batTeam = safeTeam(batTeam)

	pitchLine := fmt.Sprintf("%s Pitching: %s %s IP, %d P, %s ERA",
		pitchTeam, safeName(pitcher.Person.FullName), pitcher.Stats.Pitching.InningsPitched,
		pitcher.Stats.Pitching.PitchesThrown, pitcher.SeasonStats.Pitching.ERA)

	batLine := fmt.Sprintf("%s At Bat: %s %d-%d, %s AVG, %d HR",
		batTeam, safeName(batter.Person.FullName), batter.Stats.Batting.Hits, batter.Stats.Batting.AtBats,
		batter.SeasonStats.Batting.AVG, batter.SeasonStats.Batting.HomeRuns)

	return lipgloss.JoinVertical(lipgloss.Left, pitchLine, batLine)
}

func lookupPlayer(id int, box mlb.Boxscore, teams mlb.GameTeams) (mlb.BoxscorePlayer, string) {
	key := fmt.Sprintf("ID%d", id)
	if player, ok := box.Teams.Home.Players[key]; ok {
		return player, teams.Home.Abbreviation
	}
	if player, ok := box.Teams.Away.Players[key]; ok {
		return player, teams.Away.Abbreviation
	}
	return mlb.BoxscorePlayer{}, ""
}

func renderAtBat(play mlb.Play) string {
	lines := []string{}
	if play.Result.Description != "" {
		lines = append(lines, play.Result.Description)
	}
	for i := len(play.PlayEvents) - 1; i >= 0; i-- {
		event := play.PlayEvents[i]
		line := renderEventLine(event)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func renderEventLine(event mlb.PlayEvent) string {
	if event.Details.Description == "" {
		return ""
	}
	line := event.Details.Description
	if event.IsPitch && event.PitchData != nil && event.PitchData.StartSpeed > 0 {
		line = fmt.Sprintf("[%0.0f MPH] %s", event.PitchData.StartSpeed, line)
	}
	if !event.IsPitch {
		if event.Details.Event != "" {
			line = fmt.Sprintf("[%s] %s", event.Details.Event, line)
		}
	}
	return line
}

func renderCount(linescore mlb.LiveLineScore) string {
	ball := bullet(linescore.Balls, 4, styles.BallColor)
	strike := bullet(linescore.Strikes, 3, styles.StrikeColor)
	out := bullet(linescore.Outs, 3, styles.OutColor)
	return strings.Join([]string{
		fmt.Sprintf("B: %s", ball),
		fmt.Sprintf("S: %s", strike),
		fmt.Sprintf("O: %s", out),
	}, "\n")
}

func renderBases(linescore mlb.LiveLineScore) string {
	on := styles.OnBaseColor
	diamond := func(active bool) string {
		if active {
			return lipgloss.NewStyle().Foreground(on).Render("◆")
		}
		return "◇"
	}
	return strings.Join([]string{
		"  " + diamond(linescore.Offense.Second != nil),
		fmt.Sprintf("%s   %s", diamond(linescore.Offense.Third != nil), diamond(linescore.Offense.First != nil)),
	}, "\n")
}

func bullet(count, total int, color color.Color) string {
	active := lipgloss.NewStyle().Foreground(color).Render("●")
	inactive := "○"
	parts := make([]string, 0, total)
	for i := range total {
		if i < count {
			parts = append(parts, active)
		} else {
			parts = append(parts, inactive)
		}
	}
	return strings.Join(parts, " ")
}

func renderAllPlays(plays []mlb.Play) string {
	if len(plays) == 0 {
		return ""
	}
	var lines []string
	for i := len(plays) - 1; i >= 0; i-- {
		play := plays[i]
		inning := fmt.Sprintf("[%s %d]", strings.ToUpper(play.About.HalfInning), play.About.Inning)
		result := colorForPlay(play)
		line := fmt.Sprintf("%s %s", inning, result)
		lines = append(lines, line)
		for j := len(play.PlayEvents) - 1; j >= 0; j-- {
			event := play.PlayEvents[j]
			if event.Details.Description == "" {
				continue
			}
			lines = append(lines, "  "+event.Details.Description)
		}
	}
	return strings.Join(lines, "\n")
}

func colorForPlay(play mlb.Play) string {
	color := styles.OtherEventColor
	last := lastPlayEvent(play)
	if last != nil {
		if last.Details.IsBall {
			color = styles.WalkColor
		} else if last.Details.IsStrike {
			color = styles.StrikeOutColor
		} else if last.Details.IsInPlay {
			if play.About.HasOut {
				color = styles.InPlayOutColor
			} else {
				color = styles.InPlayNoOutColor
			}
		}
	}
	style := lipgloss.NewStyle().Foreground(color).Bold(play.About.IsScoringPlay)
	text := play.Result.Event
	if text == "" {
		text = play.Result.Description
	}
	return style.Render(text)
}

func lastPlayEvent(play mlb.Play) *mlb.PlayEvent {
	if len(play.PlayEvents) == 0 {
		return nil
	}
	return &play.PlayEvents[len(play.PlayEvents)-1]
}

func renderDecisions(decisions *mlb.Decisions, box mlb.Boxscore, teams mlb.GameTeams) string {
	if decisions == nil {
		return ""
	}
	var lines []string
	if decisions.Winner != nil {
		player, team := lookupPlayer(decisions.Winner.ID, box, teams)
		lines = append(lines, fmt.Sprintf("Win: %s %s (%d-%d)", safeTeam(team), safeName(player.Person.FullName), player.SeasonStats.Pitching.Wins, player.SeasonStats.Pitching.Losses))
	}
	if decisions.Loser != nil {
		player, team := lookupPlayer(decisions.Loser.ID, box, teams)
		lines = append(lines, fmt.Sprintf("Loss: %s %s (%d-%d)", safeTeam(team), safeName(player.Person.FullName), player.SeasonStats.Pitching.Wins, player.SeasonStats.Pitching.Losses))
	}
	if decisions.Save != nil {
		player, team := lookupPlayer(decisions.Save.ID, box, teams)
		lines = append(lines, fmt.Sprintf("Save: %s %s (%d)", safeTeam(team), safeName(player.Person.FullName), player.SeasonStats.Pitching.Saves))
	}
	return strings.Join(lines, "\n")
}

func renderInning(linescore mlb.LiveLineScore) string {
	symbol := "▼"
	if linescore.IsTopInning {
		symbol = "▲"
	}
	return inningStyle.Render(fmt.Sprintf("%s\n%d", symbol, linescore.CurrentInning))
}

var (
	columnStyle = lipgloss.NewStyle().Width(30).Align(lipgloss.Center).Padding(0, 2)
	tableStyle  = lipgloss.NewStyle().Padding(0, 1)
	inningStyle = lipgloss.NewStyle().Align(lipgloss.Right)
	countStyle  = lipgloss.NewStyle().MarginLeft(2)
	basesStyle  = lipgloss.NewStyle().MarginLeft(2)
)

func safeName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Unknown"
	}
	return trimmed
}

func safeTeam(abbrev string) string {
	trimmed := strings.TrimSpace(abbrev)
	if trimmed == "" {
		return "UNK"
	}
	return trimmed
}
