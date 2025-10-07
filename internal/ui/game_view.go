package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"

	"go.dalton.dog/batterup/internal/mlb"
	"go.dalton.dog/batterup/internal/styles"
)

// Region: Game Preview

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
			startTime = t.Local().Format("Monday, January 2, 2006 3:04 PM MST")
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
			lines = append(lines, fmt.Sprintf("\n(#%s) %s", player.JerseyNumber, player.Person.FullName))
			lines = append(lines, fmt.Sprintf("%d-%d", player.SeasonStats.Pitching.Wins, player.SeasonStats.Pitching.Losses))
			lines = append(lines, fmt.Sprintf("%s ERA %d K", player.SeasonStats.Pitching.ERA, player.SeasonStats.Pitching.StrikeOuts))
		}
	} else {
		lines = append(lines, "", "Probable: TBD")
	}
	return lines
}

// End Region: Game Preview

// Region: Live Game

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

	lineScoreTable := lipgloss.NewStyle().PaddingRight(1).Render(renderLineScoreTable(linescore, teams))

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		renderInning(linescore),
		countStyle.Render(renderCount(linescore)),
		basesStyle.Render(renderBases(linescore)),
	)

	if g.width/2 < (lipgloss.Width(header) + lipgloss.Width(lineScoreTable)) {
		header = lipgloss.JoinVertical(lipgloss.Center,
			header,
			lineScoreTable,
		)
	} else {
		header = lipgloss.JoinHorizontal(lipgloss.Center,
			header,
			lineScoreTable,
		)
	}

	matchup := ""
	atBat := ""
	if playAvailable {
		matchup = renderMatchup(play, g.feed.LiveData.Boxscore, teams)
		atBat = renderAtBat(play)
	}

	leftContent := lipgloss.JoinVertical(lipgloss.Left,
		header,
		lipgloss.NewStyle().Margin(1, 0).Render(matchup),
		atBat,
	)

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Height(g.height)
	leftContent = style.Width(lipgloss.Width(header) + 3).Render(leftContent)

	plays := style.Width(g.width - lipgloss.Width(leftContent) - 2).Render(g.renderPlaysView())
	return lipgloss.JoinHorizontal(lipgloss.Top, leftContent, plays)
}

// Left Half

func renderLineScoreTable(linescore mlb.LiveLineScore, teams mlb.GameTeams) string {
	tbl := table.New().Border(lipgloss.RoundedBorder())
	totalInnings := max(len(linescore.Innings), 9)

	header := []string{"   "}
	for i := 1; i <= totalInnings; i++ {
		header = append(header, fmt.Sprintf("%2d", i))
	}
	header = append(header, " R", " H", " E")

	tbl = tbl.Headers(header...)

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

	tbl = tbl.Row(awayLine...).Row(homeLine...)

	return tableStyle.Render(tbl.String())
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
	var lines []string
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
	if !event.IsPitch && event.Details.Event != "" {
		line = fmt.Sprintf("[%s] %s", event.Details.Event, line)
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

func bullet(count, total int, clr color.Color) string {
	active := lipgloss.NewStyle().Foreground(clr).Render("●")
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

// Right Half

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

func colorForPlay(play mlb.Play) string {
	clr := styles.OtherEventColor
	last := lastPlayEvent(play)
	if last != nil {
		if last.Details.IsBall {
			clr = styles.WalkColor
		} else if last.Details.IsStrike {
			clr = styles.StrikeOutColor
		} else if last.Details.IsInPlay {
			if play.About.HasOut {
				clr = styles.InPlayOutColor
			} else {
				clr = styles.InPlayNoOutColor
			}
		}
	}
	style := lipgloss.NewStyle().Foreground(clr).Bold(play.About.IsScoringPlay)
	text := play.Result.Event
	if text == "" {
		text = play.Result.Description
	}
	return style.Render(text)
}

// Gets the most recent play event from the given play
func lastPlayEvent(play mlb.Play) *mlb.PlayEvent {
	if len(play.PlayEvents) == 0 {
		return nil
	}
	return &play.PlayEvents[len(play.PlayEvents)-1]
}

// Render the inning, with the symbol above or below the digit
func renderInning(linescore mlb.LiveLineScore) string {
	if linescore.IsTopInning {
		return inningStyle.Render(fmt.Sprintf("▲\n%d\n", linescore.CurrentInning))
	}
	return inningStyle.Render(fmt.Sprintf("\n%d\n▼", linescore.CurrentInning))
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
