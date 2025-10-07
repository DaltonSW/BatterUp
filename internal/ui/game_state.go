package ui

import (
	"fmt"
	"strings"

	"go.dalton.dog/batterup/internal/mlb"
)

type playView struct {
	play        mlb.Play
	snapshot    playSnapshot
	lines       []string
	headerIndex int
	lineCount   int
}

type playLine struct {
	text      string
	playIndex int
	isHeader  bool
}

type playSnapshot struct {
	linescore mlb.LiveLineScore
}

func (g *GameModel) resetPlayState() {
	g.playViews = nil
	g.playLines = nil
	g.playLineOffsets = nil
	g.playsOffset = 0
	g.selectedPlay = 0
	g.selectedAtBat = -1
}

func (g *GameModel) refreshViewport() {
	if g.feed == nil {
		g.resetPlayState()
		return
	}
	wasFollowing := g.isFollowingLatest()
	prevAtBat := g.selectedAtBat
	plays := g.feed.LiveData.Plays.AllPlays
	if len(plays) == 0 {
		g.resetPlayState()
		return
	}
	snapshots := buildPlaySnapshots(plays)
	g.playViews = buildPlayViews(plays, snapshots)
	if len(g.playViews) == 0 {
		g.resetPlayState()
		return
	}
	switch {
	case wasFollowing:
		g.selectedPlay = len(g.playViews) - 1
	case prevAtBat >= 0:
		if idx := g.indexForAtBat(prevAtBat); idx >= 0 {
			g.selectedPlay = idx
		} else {
			g.selectedPlay = len(g.playViews) - 1
		}
	default:
		g.selectedPlay = len(g.playViews) - 1
	}
	if g.selectedPlay >= len(g.playViews) {
		g.selectedPlay = len(g.playViews) - 1
	}
	g.selectedAtBat = g.playViews[g.selectedPlay].play.AtBatIndex
	g.rebuildPlayLines()
	g.scrollToSelected()
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
		headerIndex := view.headerIndex
		if headerIndex < 0 || headerIndex >= len(view.lines) {
			headerIndex = 0
		}
		for lineIdx, text := range view.lines {
			lines = append(lines, playLine{
				text:      text,
				playIndex: idx,
				isHeader:  lineIdx == headerIndex,
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
	if len(g.playViews) == 0 || delta == 0 {
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

func (g *GameModel) moveToStart() {
	if len(g.playViews) == 0 {
		return
	}
	g.selectedPlay = 0
	g.selectedAtBat = g.playViews[0].play.AtBatIndex
	g.scrollToSelected()
}

func (g *GameModel) moveToEnd() {
	if len(g.playViews) == 0 {
		return
	}
	g.selectedPlay = len(g.playViews) - 1
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
	maxOffset := max(g.maxPlaysOffset(), 0)

	if g.playsOffset > maxOffset {
		g.playsOffset = maxOffset
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

func (g *GameModel) isFollowingLatest() bool {
	if len(g.playViews) == 0 {
		return true
	}
	if g.selectedPlay != len(g.playViews)-1 {
		return false
	}
	if g.playsHeight <= 0 || len(g.playLines) == 0 {
		return true
	}
	return g.playsOffset >= g.maxPlaysOffset()
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
