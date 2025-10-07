package ui

import (
	"strings"
	"testing"

	"go.dalton.dog/batterup/internal/mlb"
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
		Result: mlb.PlayResult{Description: "Player doubled"},
		PlayEvents: []mlb.PlayEvent{
			{Details: mlb.PlayEventDetails{Description: "Pitch 1"}},
			{Details: mlb.PlayEventDetails{Description: "Pitch 2"}},
		},
	}
	out := renderAtBat(play)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "Player doubled" {
		t.Fatalf("expected result description first, got %q", lines[0])
	}
	if lines[1] != "Pitch 1" || lines[2] != "Pitch 2" {
		t.Fatalf("expected events in chronological order, got %v", lines[1:])
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
		About:      mlb.PlayAbout{HalfInning: "top", Inning: 3},
		Result:     mlb.PlayResult{Event: "Single"},
		PlayEvents: []mlb.PlayEvent{{Details: mlb.PlayEventDetails{Description: "Line drive"}}},
	}
	lines := renderPlayLines(play)
	if len(lines) != 2 {
		t.Fatalf("expected event summary and one detail, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Single") {
		t.Fatalf("expected first line to include event, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "Line drive") || !strings.HasPrefix(lines[1], "  ") {
		t.Fatalf("expected indented detail line, got %q", lines[1])
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
