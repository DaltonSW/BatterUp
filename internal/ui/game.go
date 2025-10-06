package ui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"

	"go.dalton.dog/batterup/internal/mlb"
	"go.dalton.dog/batterup/internal/styles"
)

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
	playsWidth      int
	selectedPlay    int
	selectedAtBat   int
}

type playView struct {
	play      mlb.Play
	snapshot  playSnapshot
	lines     []string
	lineCount int
}

type playLine struct {
	text      string
	playIndex int
	isHeader  bool
}

type playSnapshot struct {
	linescore mlb.LiveLineScore
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
	g.height = height

	availWidth := max(width-4, 20)
	availHeight := max(height-10, 5)

	g.playsWidth = availWidth
	g.playsHeight = availHeight
	g.scrollToSelected()
}

func (g GameModel) Start(gameID int) (GameModel, tea.Cmd) {
	g.gameID = gameID
	g.feed = nil
	g.err = nil
	g.active = true
	g.playViews = nil
	g.playLines = nil
	g.playLineOffsets = nil
	g.playsOffset = 0
	g.selectedPlay = 0
	g.selectedAtBat = -1
	g.loading = gameID != 0

	if gameID == 0 {
		return g, nil
	}

	g.requestID++
	return g, g.fetch()
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
	// case "L":
	// 	return g.renderLive()
	// case "F":
	// 	return g.renderFinished()
	default:
		return g.renderLive()
	}
}

func (g *GameModel) refreshViewport() {
	if g.feed == nil {
		g.playViews = nil
		g.playLines = nil
		g.playLineOffsets = nil
		g.playsOffset = 0
		g.selectedPlay = 0
		g.selectedAtBat = -1
		return
	}
	plays := g.feed.LiveData.Plays.AllPlays
	if len(plays) == 0 {
		g.playViews = nil
		g.playLines = nil
		g.playLineOffsets = nil
		g.playsOffset = 0
		g.selectedPlay = 0
		g.selectedAtBat = -1
		return
	}
	snapshots := buildPlaySnapshots(plays)
	g.playViews = buildPlayViews(plays, snapshots)
	if len(g.playViews) == 0 {
		g.playLines = nil
		g.playLineOffsets = nil
		g.playsOffset = 0
		g.selectedPlay = 0
		g.selectedAtBat = -1
		return
	}
	if g.selectedAtBat >= 0 {
		if idx := g.indexForAtBat(g.selectedAtBat); idx >= 0 {
			g.selectedPlay = idx
		} else {
			g.selectedPlay = 0
		}
	} else {
		g.selectedPlay = 0
	}
	if g.selectedPlay >= len(g.playViews) {
		g.selectedPlay = len(g.playViews) - 1
	}
	g.selectedAtBat = g.playViews[g.selectedPlay].play.AtBatIndex
	g.rebuildPlayLines()
	g.scrollToSelected()
}

func (g *GameModel) renderPlaysView() string {
	if len(g.playLines) == 0 {
		return ""
	}
	if g.playsHeight <= 0 {
		lines := make([]string, 0, len(g.playLines))
		for i := range g.playLines {
			lines = append(lines, g.renderPlayLine(i))
		}
		return strings.Join(lines, "\n")
	}
	start := max(g.playsOffset, 0)
	maxOffset := g.maxPlaysOffset()
	if start > maxOffset {
		start = maxOffset
	}
	end := min(start+g.playsHeight, len(g.playLines))
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, g.renderPlayLine(i))
	}
	return strings.Join(lines, "\n")
}

func (g *GameModel) renderPlayLine(idx int) string {
	line := g.playLines[idx]
	text := line.text
	if line.playIndex == g.selectedPlay && line.isHeader {
		return selectedPlayStyle.Render(text)
	}
	return text
}

func (g *GameModel) rebuildPlayLines() {
	if len(g.playViews) == 0 {
		g.playLines = nil
		g.playLineOffsets = nil
		return
	}
	lines := make([]playLine, 0, len(g.playViews)*3)
	offsets := make([]int, len(g.playViews))
	linePos := 0
	for idx, view := range g.playViews {
		offsets[idx] = linePos
		for lineIdx, text := range view.lines {
			lines = append(lines, playLine{
				text:      text,
				playIndex: idx,
				isHeader:  lineIdx == 0,
			})
			linePos++
		}
	}
	g.playLines = lines
	g.playLineOffsets = offsets
}

func (g *GameModel) indexForAtBat(atBat int) int {
	for idx, view := range g.playViews {
		if view.play.AtBatIndex == atBat {
			return idx
		}
	}
	return -1
}

func (g *GameModel) scrollToSelected() {
	if g.playsHeight <= 0 {
		g.playsOffset = 0
		return
	}
	if len(g.playLineOffsets) == 0 {
		g.playsOffset = 0
		return
	}
	if g.selectedPlay < 0 {
		g.selectedPlay = 0
	}
	if g.selectedPlay >= len(g.playLineOffsets) {
		g.selectedPlay = len(g.playLineOffsets) - 1
	}
	start := g.playLineOffsets[g.selectedPlay]
	end := start + g.playViews[g.selectedPlay].lineCount
	if g.playsOffset > start {
		g.playsOffset = start
	} else if end > g.playsOffset+g.playsHeight {
		g.playsOffset = max(0, end-g.playsHeight)
	}
	g.enforcePlayOffset()
}

func (g *GameModel) moveSelection(delta int) {
	if len(g.playViews) == 0 {
		return
	}
	if delta == 0 {
		return
	}
	idx := max(g.selectedPlay+delta, 0)

	if idx >= len(g.playViews) {
		idx = len(g.playViews) - 1
	}
	if idx == g.selectedPlay {
		return
	}
	g.selectedPlay = idx
	g.selectedAtBat = g.playViews[g.selectedPlay].play.AtBatIndex
	g.scrollToSelected()
}

func (g *GameModel) visiblePlayCount() int {
	if g.playsHeight <= 0 || len(g.playLines) == 0 {
		return len(g.playViews)
	}
	start := max(g.playsOffset, 0)
	end := min(start+g.playsHeight, len(g.playLines))
	if end <= start {
		return 0
	}
	seen := make(map[int]struct{}, end-start)
	for i := start; i < end; i++ {
		seen[g.playLines[i].playIndex] = struct{}{}
	}
	return len(seen)
}

func (g *GameModel) pageDelta() int {
	count := g.visiblePlayCount()
	if count <= 1 {
		count = g.playsHeight
	}
	if count <= 1 {
		count = 5
	}
	return max(1, count-1)
}

func (g *GameModel) currentPlayView() *playView {
	if g.selectedPlay < 0 || g.selectedPlay >= len(g.playViews) {
		return nil
	}
	return &g.playViews[g.selectedPlay]
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
	if len(g.playLines) <= g.playsHeight {
		return 0
	}
	return len(g.playLines) - g.playsHeight
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
	selected := g.currentPlayView()
	var play mlb.Play
	playAvailable := false
	if selected != nil {
		linescore = selected.snapshot.linescore
		play = selected.play
		playAvailable = true
	} else {
		play = g.feed.LiveData.Plays.CurrentPlay
		playAvailable = true
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		renderInning(linescore),
		countStyle.Render(renderCount(linescore)),
		basesStyle.Render(renderBases(linescore)),
		lipgloss.NewStyle().PaddingLeft(2).Render(renderLineScoreTable(linescore, teams)),
	)

	matchup := ""
	atBat := ""
	if playAvailable {
		matchup = renderMatchup(play, g.feed.LiveData.Boxscore, teams)
		atBat = renderAtBat(play)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		lipgloss.NewStyle().Margin(1, 0).Render(matchup),
		atBat,
		"",
		lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Width(g.playsWidth).Render(g.renderPlaysView()),
	)

	return content
}

// func (g *GameModel) renderFinished() string {
// 	linescore := g.feed.LiveData.Linescore
// 	teams := g.feed.GameData.Teams
// 	decisions := g.feed.LiveData.Decisions
// 	box := g.feed.LiveData.Boxscore
//
// 	score := fmt.Sprintf("%s %d - %s %d",
// 		teams.Away.Abbreviation, linescore.Teams.Away.Runs,
// 		teams.Home.Abbreviation, linescore.Teams.Home.Runs,
// 	)
//
// 	dec := renderDecisions(decisions, box, g.feed.GameData.Teams)
//
// 	table := renderLineScoreTable(linescore, teams)
//
// 	return lipgloss.JoinVertical(lipgloss.Left,
// 		lipgloss.NewStyle().Bold(true).Render(score),
// 		table,
// 		"",
// 		dec,
// 	)
// }

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

func renderMatchup(play mlb.Play, box mlb.Boxscore, teams mlb.GameTeams) string {
	pitcher, pitchTeam := lookupPlayer(play.Matchup.Pitcher.ID, box, teams)
	batter, batTeam := lookupPlayer(play.Matchup.Batter.ID, box, teams)

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

func renderPlayLines(play mlb.Play) []string {
	header := fmt.Sprintf("[%s %d] %s", strings.ToUpper(play.About.HalfInning), play.About.Inning, colorForPlay(play))
	lines := []string{header}
	for j := len(play.PlayEvents) - 1; j >= 0; j-- {
		event := play.PlayEvents[j]
		if event.Details.Description == "" {
			continue
		}
		lines = append(lines, "  "+event.Details.Description)
	}
	return lines
}

func buildPlayViews(plays []mlb.Play, snapshots []playSnapshot) []playView {
	if len(plays) != len(snapshots) {
		return nil
	}
	views := make([]playView, 0, len(plays))
	for i := len(plays) - 1; i >= 0; i-- {
		lines := renderPlayLines(plays[i])
		if len(lines) == 0 {
			lines = []string{""}
		}
		views = append(views, playView{
			play:      plays[i],
			snapshot:  snapshots[i],
			lines:     lines,
			lineCount: len(lines),
		})
	}
	return views
}

func buildPlaySnapshots(plays []mlb.Play) []playSnapshot {
	if len(plays) == 0 {
		return nil
	}
	acc := newGameAccumulator()
	snapshots := make([]playSnapshot, len(plays))
	for i, play := range plays {
		acc.advance(play)
		snapshots[i] = acc.snapshot(play)
	}
	return snapshots
}

type gameAccumulator struct {
	scoreAway     int
	scoreHome     int
	hitsAway      int
	hitsHome      int
	errorsAway    int
	errorsHome    int
	bases         map[string]int
	innings       map[int]*inningTotals
	maxInning     int
	currentInning int
	currentIsTop  bool
}

type inningTotals struct {
	awayRuns   int
	homeRuns   int
	awayPlayed bool
	homePlayed bool
}

func newGameAccumulator() *gameAccumulator {
	return &gameAccumulator{
		bases: map[string]int{
			baseFirst:  0,
			baseSecond: 0,
			baseThird:  0,
		},
		innings:       make(map[int]*inningTotals),
		currentInning: -1,
	}
}

func (a *gameAccumulator) advance(play mlb.Play) {
	a.ensureHalfInning(play)
	a.applyRuns(play)
	a.applyHitsAndErrors(play)
	a.applyRunners(play)
}

func (a *gameAccumulator) ensureHalfInning(play mlb.Play) {
	if a.currentInning == play.About.Inning && a.currentIsTop == play.About.IsTopInning {
		return
	}
	a.currentInning = play.About.Inning
	a.currentIsTop = play.About.IsTopInning
	a.bases[baseFirst] = 0
	a.bases[baseSecond] = 0
	a.bases[baseThird] = 0
}

func (a *gameAccumulator) applyRuns(play mlb.Play) {
	prevAway, prevHome := a.scoreAway, a.scoreHome
	a.scoreAway = play.Result.AwayScore
	a.scoreHome = play.Result.HomeScore

	deltaAway := max(a.scoreAway-prevAway, 0)
	deltaHome := max(a.scoreHome-prevHome, 0)

	inning := play.About.Inning
	totals, ok := a.innings[inning]
	if !ok {
		totals = &inningTotals{}
		a.innings[inning] = totals
	}
	if play.About.IsTopInning {
		totals.awayRuns += deltaAway
		totals.awayPlayed = true
	} else {
		totals.homeRuns += deltaHome
		totals.homePlayed = true
	}
	if inning > a.maxInning {
		a.maxInning = inning
	}
}

func (a *gameAccumulator) applyHitsAndErrors(play mlb.Play) {
	eventType := strings.ToLower(play.Result.EventType)
	if eventType == "" {
		eventType = strings.ToLower(play.Result.Event)
	}
	if isHitEvent(eventType) {
		if play.About.IsTopInning {
			a.hitsAway++
		} else {
			a.hitsHome++
		}
	}
	if isErrorEvent(eventType) {
		if play.About.IsTopInning {
			a.errorsHome++
		} else {
			a.errorsAway++
		}
	}
}

func (a *gameAccumulator) applyRunners(play mlb.Play) {
	nextBases := map[string]int{
		baseFirst:  a.bases[baseFirst],
		baseSecond: a.bases[baseSecond],
		baseThird:  a.bases[baseThird],
	}
	for _, runner := range play.Runners {
		id := runner.Details.Runner.ID
		start := normalizeBase(runner.Movement.Start)
		if start == "" {
			start = normalizeBase(runner.Movement.OriginBase)
		}
		end := normalizeBase(runner.Movement.End)
		if start != "" {
			nextBases[start] = 0
		}
		if runner.Movement.IsOut {
			continue
		}
		if end != "" {
			nextBases[end] = id
		}
	}
	a.bases = nextBases
}

func (a *gameAccumulator) snapshot(play mlb.Play) playSnapshot {
	linescore := mlb.LiveLineScore{
		CurrentInning:        play.About.Inning,
		CurrentInningOrdinal: ordinal(play.About.Inning),
		InningState:          halfInningLabel(play.About.HalfInning),
		IsTopInning:          play.About.IsTopInning,
		Balls:                play.Count.Balls,
		Strikes:              play.Count.Strikes,
		Outs:                 play.Count.Outs,
		Teams: mlb.LineScoreTotals{
			Away: mlb.LineScoreTeam{Runs: a.scoreAway, Hits: a.hitsAway, Errors: a.errorsAway},
			Home: mlb.LineScoreTeam{Runs: a.scoreHome, Hits: a.hitsHome, Errors: a.errorsHome},
		},
		Offense: mlb.OffensiveState{
			First:  baseRunnerPtr(a.bases[baseFirst]),
			Second: baseRunnerPtr(a.bases[baseSecond]),
			Third:  baseRunnerPtr(a.bases[baseThird]),
		},
		Innings: a.buildInnings(),
	}
	return playSnapshot{linescore: linescore}
}

func (a *gameAccumulator) buildInnings() []mlb.InningLine {
	if a.maxInning == 0 {
		return nil
	}
	innings := make([]mlb.InningLine, 0, a.maxInning)
	for inning := 1; inning <= a.maxInning; inning++ {
		totals, ok := a.innings[inning]
		if !ok {
			totals = &inningTotals{}
		}
		line := mlb.InningLine{Num: inning}
		if totals.awayPlayed {
			runs := totals.awayRuns
			line.Away.Runs = &runs
		}
		if totals.homePlayed {
			runs := totals.homeRuns
			line.Home.Runs = &runs
		}
		innings = append(innings, line)
	}
	return innings
}

const (
	baseFirst  = "1B"
	baseSecond = "2B"
	baseThird  = "3B"
)

func normalizeBase(value string) string {
	switch strings.ToUpper(value) {
	case baseFirst:
		return baseFirst
	case baseSecond:
		return baseSecond
	case baseThird:
		return baseThird
	default:
		return ""
	}
}

func baseRunnerPtr(id int) *mlb.BaseRunner {
	if id == 0 {
		return nil
	}
	return &mlb.BaseRunner{ID: id}
}

func isHitEvent(eventType string) bool {
	switch eventType {
	case "single", "double", "triple", "ground_rule_double":
		return true
	}
	if strings.Contains(eventType, "home_run") {
		return true
	}
	return false
}

func isErrorEvent(eventType string) bool {
	if eventType == "" {
		return false
	}
	return strings.Contains(eventType, "error")
}

func halfInningLabel(half string) string {
	switch strings.ToLower(half) {
	case "top":
		return "Top"
	case "bottom":
		return "Bottom"
	default:
		if half == "" {
			return ""
		}
		lower := strings.ToLower(half)
		runes := []rune(lower)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		return string(runes)
	}
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 != 11 && n%100 != 12 && n%100 != 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
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

// func renderDecisions(decisions *mlb.Decisions, box mlb.Boxscore, teams mlb.GameTeams) string {
// 	if decisions == nil {
// 		return ""
// 	}
// 	var lines []string
// 	if decisions.Winner != nil {
// 		player, team := lookupPlayer(decisions.Winner.ID, box, teams)
// 		lines = append(lines, fmt.Sprintf("Win: %s %s (%d-%d)", safeTeam(team), safeName(player.Person.FullName), player.SeasonStats.Pitching.Wins, player.SeasonStats.Pitching.Losses))
// 	}
// 	if decisions.Loser != nil {
// 		player, team := lookupPlayer(decisions.Loser.ID, box, teams)
// 		lines = append(lines, fmt.Sprintf("Loss: %s %s (%d-%d)", safeTeam(team), safeName(player.Person.FullName), player.SeasonStats.Pitching.Wins, player.SeasonStats.Pitching.Losses))
// 	}
// 	if decisions.Save != nil {
// 		player, team := lookupPlayer(decisions.Save.ID, box, teams)
// 		lines = append(lines, fmt.Sprintf("Save: %s %s (%d)", safeTeam(team), safeName(player.Person.FullName), player.SeasonStats.Pitching.Saves))
// 	}
// 	return strings.Join(lines, "\n")
// }

func renderInning(linescore mlb.LiveLineScore) string {
	symbol := "▼"
	if linescore.IsTopInning {
		symbol = "▲"
	}
	return inningStyle.Render(fmt.Sprintf("%s\n%d", symbol, linescore.CurrentInning))
}

var (
	columnStyle       = lipgloss.NewStyle().Width(30).Align(lipgloss.Center).Padding(0, 2)
	tableStyle        = lipgloss.NewStyle().Padding(0, 1)
	inningStyle       = lipgloss.NewStyle().Align(lipgloss.Right)
	countStyle        = lipgloss.NewStyle().MarginLeft(2)
	basesStyle        = lipgloss.NewStyle().MarginLeft(2)
	selectedPlayStyle = lipgloss.NewStyle().Bold(true).Underline(true)
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
