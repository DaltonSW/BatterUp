package ui

import (
	"strings"
	"testing"
	"time"

	"go.dalton.dog/batterup/internal/mlb"
	"go.dalton.dog/batterup/internal/styles"
)

func TestSameDay(t *testing.T) {
	a := time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)
	b := time.Date(2024, time.April, 1, 23, 59, 0, 0, time.UTC)
	if !sameDay(a, b) {
		t.Fatalf("expected same day to be true")
	}
	c := b.Add(2 * time.Hour)
	if sameDay(a, c) {
		t.Fatalf("expected different days to be false")
	}
}

func TestDescribeGameStatusPreGameVariants(t *testing.T) {
	base := mlb.ScheduleGame{
		Status:   mlb.GameStatus{AbstractGameCode: "P", DetailedState: "Scheduled"},
		GameDate: time.Date(2024, time.July, 4, 17, 5, 0, 0, time.FixedZone("EDT", -4*3600)),
	}

	game := base
	game.DoubleHeader = "Y"
	game.GameNumber = 2
	if got := describeGameStatus(game); got != "Game 2" {
		t.Fatalf("expected double header label, got %q", got)
	}

	game = base
	game.Status.StartTimeTBD = true
	if got := describeGameStatus(game); got != "Start time TBD" {
		t.Fatalf("expected TBD label, got %q", got)
	}

	game = base
	want := game.GameDate.Local().Format("Starts @ 3:04 PM MST")
	if got := describeGameStatus(game); got != want {
		t.Fatalf("expected formatted start time %q, got %q", want, got)
	}
}

func TestDescribeGameStatusLive(t *testing.T) {
	game := mlb.ScheduleGame{
		Status:    mlb.GameStatus{AbstractGameCode: "L", DetailedState: "In Progress"},
		Linescore: &mlb.GameLineScore{InningState: "Top", CurrentInningOrdinal: "4th"},
	}
	if got := describeGameStatus(game); got != "Top 4th" {
		t.Fatalf("expected inning status, got %q", got)
	}

	game.Linescore.InningState = ""
	game.Linescore.CurrentInningOrdinal = ""
	if got := describeGameStatus(game); got != "In Progress" {
		t.Fatalf("expected detailed state fallback, got %q", got)
	}
}

func TestDescribeGameStatusFinal(t *testing.T) {
	game := mlb.ScheduleGame{
		Status: mlb.GameStatus{
			AbstractGameCode: "F",
			DetailedState:    "Final",
			Reason:           "Rain",
		},
	}
	if got := describeGameStatus(game); got != "Final | Rain" {
		t.Fatalf("expected reason appended, got %q", got)
	}

	game.Status.Reason = ""
	if got := describeGameStatus(game); got != "Final" {
		t.Fatalf("expected detailed state when no reason, got %q", got)
	}
}

func TestFormatScheduleTeam(t *testing.T) {
	team := mlb.ScheduleTeam{
		Team:         mlb.TeamInfo{TeamName: "Rockies"},
		LeagueRecord: mlb.LeagueRecord{Wins: 41, Losses: 121},
	}
	got := formatScheduleTeam(team, styles.ScheduleWinnerTeam, 24)
	if !strings.Contains(got, "Rockies") || !strings.Contains(got, "(41-121)") {
		t.Fatalf("expected team name and record in %q", got)
	}
}

func TestFormatScheduleTeamTruncatesWhenNarrow(t *testing.T) {
	team := mlb.ScheduleTeam{
		Team:         mlb.TeamInfo{TeamName: "Some Extremely Long Team Name That Will Overflow"},
		LeagueRecord: mlb.LeagueRecord{Wins: 102, Losses: 60},
	}
	got := formatScheduleTeam(team, styles.ScheduleWinnerTeam, 18)
	if !strings.Contains(got, "…") {
		t.Fatalf("expected truncation for narrow width, got %q", got)
	}
	if !strings.Contains(got, "(102-60)") {
		t.Fatalf("expected record present even when truncated, got %q", got)
	}
}

func TestFormatScheduleTeamRecordOnlyWhenTooTight(t *testing.T) {
	team := mlb.ScheduleTeam{
		Team:         mlb.TeamInfo{TeamName: "Team"},
		LeagueRecord: mlb.LeagueRecord{Wins: 99, Losses: 63},
	}
	got := formatScheduleTeam(team, styles.ScheduleWinnerTeam, 4)
	if !strings.Contains(got, "(") {
		t.Fatalf("expected record fallback when width tight, got %q", got)
	}
}
