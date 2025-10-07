package ui

import (
	"testing"

	"go.dalton.dog/batterup/internal/mlb"
)

func TestBuildPlaySnapshotsAccumulatesRunsHitsAndBases(t *testing.T) {
	plays := []mlb.Play{
		{
			Result: mlb.PlayResult{
				EventType: "single",
				AwayScore: 0,
				HomeScore: 0,
			},
			About: mlb.PlayAbout{
				Inning:      1,
				HalfInning:  "top",
				IsTopInning: true,
			},
			Count: mlb.PlayCount{Balls: 1, Strikes: 0, Outs: 0},
			Runners: []mlb.PlayRunner{
				{
					Movement: mlb.RunnerMovement{End: "1B"},
					Details:  mlb.RunnerDetails{Runner: mlb.RunnerInfo{ID: 101}},
				},
			},
		},
		{
			Result: mlb.PlayResult{
				EventType: "home_run",
				AwayScore: 2,
				HomeScore: 0,
			},
			About: mlb.PlayAbout{
				Inning:      1,
				HalfInning:  "top",
				IsTopInning: true,
				HasOut:      false,
			},
			Count: mlb.PlayCount{Balls: 0, Strikes: 0, Outs: 0},
			Runners: []mlb.PlayRunner{
				{
					Movement: mlb.RunnerMovement{Start: "1B", End: "HOME"},
					Details:  mlb.RunnerDetails{Runner: mlb.RunnerInfo{ID: 101}},
				},
				{
					Movement: mlb.RunnerMovement{End: "HOME"},
					Details:  mlb.RunnerDetails{Runner: mlb.RunnerInfo{ID: 102}},
				},
			},
		},
	}

	snapshots := buildPlaySnapshots(plays)
	if len(snapshots) != len(plays) {
		t.Fatalf("expected %d snapshots, got %d", len(plays), len(snapshots))
	}

	first := snapshots[0].linescore
	if first.Teams.Away.Hits != 1 {
		t.Fatalf("expected 1 away hit, got %d", first.Teams.Away.Hits)
	}
	if first.Offense.First == nil || first.Offense.First.ID != 101 {
		t.Fatalf("expected runner on first with ID 101")
	}

	second := snapshots[1].linescore
	if second.Teams.Away.Runs != 2 {
		t.Fatalf("expected 2 away runs, got %d", second.Teams.Away.Runs)
	}
	if second.Teams.Away.Hits != 2 {
		t.Fatalf("expected 2 away hits, got %d", second.Teams.Away.Hits)
	}
	if second.Offense.First != nil {
		t.Fatalf("expected bases empty after home run")
	}
	if len(second.Innings) == 0 || second.Innings[0].Away.Runs == nil || *second.Innings[0].Away.Runs != 2 {
		t.Fatalf("expected inning run total of 2")
	}
}

func TestBuildPlayViewsOrdersAscending(t *testing.T) {
	plays := []mlb.Play{
		{
			AtBatIndex: 1,
			Result:     mlb.PlayResult{Event: "Groundout"},
			About:      mlb.PlayAbout{Inning: 1, HalfInning: "top", IsTopInning: true},
		},
		{
			AtBatIndex: 2,
			Result:     mlb.PlayResult{Event: "Single"},
			About:      mlb.PlayAbout{Inning: 1, HalfInning: "top", IsTopInning: true},
		},
		{
			AtBatIndex: 3,
			Result:     mlb.PlayResult{Event: "Walk"},
			About:      mlb.PlayAbout{Inning: 1, HalfInning: "bottom", IsTopInning: false},
		},
	}
	snapshots := []playSnapshot{{}, {}, {}}

	views := buildPlayViews(plays, snapshots)
	if len(views) != len(plays) {
		t.Fatalf("expected %d views, got %d", len(plays), len(views))
	}
	if views[0].play.AtBatIndex != 1 {
		t.Fatalf("expected earliest play first")
	}
	if views[len(views)-1].play.AtBatIndex != 3 {
		t.Fatalf("expected latest play last")
	}
	if views[0].headerIndex != 1 {
		t.Fatalf("expected first play to include separator header index")
	}
	if views[1].headerIndex != 0 {
		t.Fatalf("expected subsequent play in same half to use zero header index")
	}
	if views[2].headerIndex != 1 {
		t.Fatalf("expected new half to introduce separator header index")
	}
	if views[0].lineCount == 0 {
		t.Fatalf("expected play lines to be populated")
	}
}

func TestNormalizeBase(t *testing.T) {
	cases := map[string]string{
		"1b":   baseFirst,
		"2B":   baseSecond,
		"3b":   baseThird,
		"home": "",
		"":     "",
	}
	for input, want := range cases {
		if got := normalizeBase(input); got != want {
			t.Fatalf("normalizeBase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsHitEvent(t *testing.T) {
	hits := []string{"single", "double", "triple", "ground_rule_double", "walk_off_home_run"}
	for _, event := range hits {
		if !isHitEvent(event) {
			t.Fatalf("expected %q to be hit event", event)
		}
	}
	if isHitEvent("strikeout") {
		t.Fatalf("strikeout should not be hit event")
	}
}

func TestIsErrorEvent(t *testing.T) {
	if !isErrorEvent("field_error") {
		t.Fatalf("expected field_error to be error event")
	}
	if isErrorEvent("") {
		t.Fatalf("empty event should not be error")
	}
}

func TestHalfInningLabel(t *testing.T) {
	cases := map[string]string{
		"top":    "Top",
		"bottom": "Bottom",
		"mid":    "Mid",
		"":       "",
	}
	for input, want := range cases {
		if got := halfInningLabel(input); got != want {
			t.Fatalf("halfInningLabel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestOrdinal(t *testing.T) {
	cases := map[int]string{
		1:  "1st",
		2:  "2nd",
		3:  "3rd",
		4:  "4th",
		11: "11th",
		12: "12th",
		13: "13th",
		21: "21st",
	}
	for input, want := range cases {
		if got := ordinal(input); got != want {
			t.Fatalf("ordinal(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestGameModelMoveSelection(t *testing.T) {
	gm := GameModel{
		playViews: []playView{
			{play: mlb.Play{AtBatIndex: 1}},
			{play: mlb.Play{AtBatIndex: 2}},
			{play: mlb.Play{AtBatIndex: 3}},
		},
		playsHeight: 5,
	}
	gm.playLineOffsets = []int{0, 1, 2}
	gm.selectedPlay = 0

	gm.moveSelection(1)
	if gm.selectedPlay != 1 || gm.selectedAtBat != 2 {
		t.Fatalf("expected selection to move to index 1 with at bat 2")
	}

	gm.moveSelection(10)
	if gm.selectedPlay != 2 {
		t.Fatalf("expected selection to clamp to last item")
	}

	gm.moveSelection(-5)
	if gm.selectedPlay != 0 {
		t.Fatalf("expected selection to clamp to first item")
	}
}

func TestGameModelVisiblePlayCount(t *testing.T) {
	gm := GameModel{
		playsHeight: 3,
		playLines: []playLine{
			{playIndex: 0},
			{playIndex: 0},
			{playIndex: 1},
			{playIndex: 1},
			{playIndex: 2},
		},
	}
	gm.playsOffset = 1

	if got := gm.visiblePlayCount(); got != 2 {
		t.Fatalf("expected 2 visible plays, got %d", got)
	}

	gm.playsOffset = 3
	if got := gm.visiblePlayCount(); got != 2 {
		t.Fatalf("expected 2 visible plays near end, got %d", got)
	}
}

func TestGameModelPageDelta(t *testing.T) {
	gm := GameModel{playsHeight: 5}
	if got := gm.pageDelta(); got != 4 {
		t.Fatalf("expected fallback delta of 4, got %d", got)
	}

	gm.playsHeight = 2
	gm.playLines = []playLine{{playIndex: 0}, {playIndex: 1}, {playIndex: 2}}
	if got := gm.pageDelta(); got != 1 {
		t.Fatalf("expected page delta of 1 when only one visible play, got %d", got)
	}

	gm.playsHeight = 3
	gm.playLines = []playLine{{playIndex: 0}, {playIndex: 1}, {playIndex: 2}, {playIndex: 2}}
	if got := gm.pageDelta(); got != 2 {
		t.Fatalf("expected page delta to use visible count - 1, got %d", got)
	}
}
