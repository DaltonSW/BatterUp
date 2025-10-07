package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss/v2"
	"go.dalton.dog/batterup/internal/mlb"
	"go.dalton.dog/batterup/internal/styles"
)

func TestRenderEventLineFormatsPitch(t *testing.T) {
	line := renderEventLine(mlb.PlayEvent{
		IsPitch:   true,
		PitchData: &mlb.PitchData{StartSpeed: 95.6},
		Details:   mlb.PlayEventDetails{Description: "Called Strike"},
	})
	if !strings.Contains(line, "MPH") || !strings.Contains(line, "Called Strike") {
		t.Fatalf("unexpected pitch line: %q", line)
	}
}

func TestRenderEventLineFormatsNonPitch(t *testing.T) {
	line := renderEventLine(mlb.PlayEvent{
		IsPitch: false,
		Details: mlb.PlayEventDetails{Description: "Pickoff attempt", Event: "Pickoff"},
	})
	if !strings.HasPrefix(line, "[Pickoff] ") {
		t.Fatalf("expected event tag for non-pitch: %q", line)
	}
}

func TestRenderEventLineSkipsWhenEmpty(t *testing.T) {
	if got := renderEventLine(mlb.PlayEvent{}); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestRenderAtBatOrdersEvents(t *testing.T) {
	play := mlb.Play{
		AtBatIndex: 5,
		Result:     mlb.PlayResult{Description: "Player doubled"},
		PlayEvents: []mlb.PlayEvent{
			{Details: mlb.PlayEventDetails{Description: "Pitch 1"}},
			{Details: mlb.PlayEventDetails{Description: "Pitch 2"}},
		},
	}
	out := renderAtBat(play)
	raw := strings.Split(out, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) != 3 {
		t.Fatalf("expected header and two events, got %d lines", len(lines))
	}
	expectedHeader := fmt.Sprintf("#%d", play.AtBatIndex+1)
	if !strings.Contains(lines[0], expectedHeader) || !strings.Contains(lines[0], "Player doubled") {
		t.Fatalf("expected numbered result line, got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "Pitch 1" || strings.TrimSpace(lines[2]) != "Pitch 2" {
		t.Fatalf("expected events in chronological order, got %q and %q", lines[1], lines[2])
	}
}

func TestRenderBasesHighlightsOccupied(t *testing.T) {
	ls := mlb.LiveLineScore{
		Offense: mlb.OffensiveState{
			First:  &mlb.BaseRunner{ID: 1},
			Second: nil,
			Third:  &mlb.BaseRunner{ID: 2},
		},
	}
	out := renderBases(ls)
	if strings.Count(out, "◆") != 2 {
		t.Fatalf("expected two occupied bases, got: %q", out)
	}
}

func TestRenderPlayLinesIncludesEventAndDetails(t *testing.T) {
	play := mlb.Play{
		AtBatIndex: 12,
		About:      mlb.PlayAbout{HalfInning: "top", Inning: 3},
		Result:     mlb.PlayResult{Event: "Single"},
		PlayEvents: []mlb.PlayEvent{{Details: mlb.PlayEventDetails{Description: "Line drive"}}},
	}
	lines := renderPlayLines(play)
	if len(lines) != 2 {
		t.Fatalf("expected event summary and one detail, got %d", len(lines))
	}
	expectedNumber := fmt.Sprintf("%d", play.AtBatIndex+1)
	if !strings.Contains(lines[0], "Single") || !strings.Contains(lines[0], expectedNumber) {
		t.Fatalf("expected first line to include number and event, got %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "Line drive" {
		t.Fatalf("expected detail line with description, got %q", lines[1])
	}
}

func TestColorForPlayMarksOuts(t *testing.T) {
	play := mlb.Play{
		Result: mlb.PlayResult{
			Event:     "Flyout",
			IsOut:     true,
			EventType: "flyout",
		},
		About: mlb.PlayAbout{
			HasOut: true,
		},
		PlayEvents: []mlb.PlayEvent{
			{Details: mlb.PlayEventDetails{IsInPlay: true}},
		},
	}
	if got := colorForPlay(play); got != lipgloss.NewStyle().Foreground(styles.OutColor).Render("Flyout") {
		t.Fatalf("expected out color for flyout, got %q", got)
	}
}

func TestColorForPlayMarksHits(t *testing.T) {
	play := mlb.Play{
		Result: mlb.PlayResult{
			Event: "Single",
			IsOut: false,
		},
		About: mlb.PlayAbout{
			HasOut: false,
		},
		PlayEvents: []mlb.PlayEvent{
			{Details: mlb.PlayEventDetails{IsInPlay: true}},
		},
	}
	if got := colorForPlay(play); got != lipgloss.NewStyle().Foreground(styles.InPlayNoOutColor).Render("Single") {
		t.Fatalf("expected in-play no-out color for single, got %q", got)
	}
}

func TestRenderMatchupUsesBoxscoreData(t *testing.T) {
	play := mlb.Play{
		Matchup: mlb.PlayMatchup{
			Pitcher: mlb.PersonRef{ID: 1},
			Batter:  mlb.PersonRef{ID: 2},
		},
	}
	box := mlb.Boxscore{}
	box.Teams.Home.Players = map[string]mlb.BoxscorePlayer{
		"ID1": {
			Person: mlb.PersonInfo{FullName: "Pitcher One"},
			Stats: mlb.BoxscorePlayerStats{
				Pitching: mlb.PitchingStats{InningsPitched: "5.0", PitchesThrown: 72},
			},
			SeasonStats: mlb.BoxscorePlayerSeason{Pitching: mlb.SeasonPitching{ERA: "3.10"}},
		},
	}
	box.Teams.Away.Players = map[string]mlb.BoxscorePlayer{
		"ID2": {
			Person: mlb.PersonInfo{FullName: "Batter Two"},
			Stats: mlb.BoxscorePlayerStats{
				Batting: mlb.BattingStats{Hits: 2, AtBats: 3},
			},
			SeasonStats: mlb.BoxscorePlayerSeason{Batting: mlb.SeasonBatting{AVG: ".320", HomeRuns: 10}},
		},
	}
	teams := mlb.GameTeams{
		Home: mlb.GameTeam{Abbreviation: "HME"},
		Away: mlb.GameTeam{Abbreviation: "AWY"},
	}
	out := renderMatchup(play, box, teams)
	if !strings.Contains(out, "HME Pitching") {
		t.Fatalf("expected home team pitching line, got %q", out)
	}
	if !strings.Contains(out, "Batter Two") {
		t.Fatalf("expected batter name, got %q", out)
	}
}

func TestRenderMatchupFallbacks(t *testing.T) {
	out := renderMatchup(mlb.Play{Matchup: mlb.PlayMatchup{}}, mlb.Boxscore{}, mlb.GameTeams{})
	if !strings.Contains(out, "UNK Pitching") {
		t.Fatalf("expected fallback team abbreviation, got %q", out)
	}
	if !strings.Contains(out, "Unknown") {
		t.Fatalf("expected fallback name, got %q", out)
	}
}

func TestRenderLineScoreTableIncludesTotals(t *testing.T) {
	two := 2
	ls := mlb.LiveLineScore{
		CurrentInning: 3,
		Innings: []mlb.InningLine{
			{Num: 1, Away: mlb.InningTeamRuns{Runs: &two}},
			{Num: 2},
			{Num: 3},
		},
		Teams: mlb.LineScoreTotals{
			Away: mlb.LineScoreTeam{Runs: 2, Hits: 5, Errors: 1},
			Home: mlb.LineScoreTeam{Runs: 1, Hits: 4, Errors: 0},
		},
	}
	teams := mlb.GameTeams{
		Away: mlb.GameTeam{Abbreviation: "AWY"},
		Home: mlb.GameTeam{Abbreviation: "HME"},
	}
	out := renderLineScoreTable(ls, teams)
	if !strings.Contains(out, "AWY") || !strings.Contains(out, "HME") {
		t.Fatalf("expected team abbreviations, got %q", out)
	}
	if !strings.Contains(out, " 2") || !strings.Contains(out, " 1") {
		t.Fatalf("expected totals in output, got %q", out)
	}
}

func TestRenderInningTopBottom(t *testing.T) {
	top := renderInning(mlb.LiveLineScore{CurrentInning: 5, IsTopInning: true})
	if !strings.Contains(top, "▲") {
		t.Fatalf("expected top indicator, got %q", top)
	}
	bottom := renderInning(mlb.LiveLineScore{CurrentInning: 5, IsTopInning: false})
	if !strings.Contains(bottom, "▼") {
		t.Fatalf("expected bottom indicator, got %q", bottom)
	}
}

func TestSafeHelpers(t *testing.T) {
	if got := safeName("   "); got != "Unknown" {
		t.Fatalf("expected Unknown fallback, got %q", got)
	}
	if got := safeTeam("  "); got != "UNK" {
		t.Fatalf("expected UNK fallback, got %q", got)
	}
}
