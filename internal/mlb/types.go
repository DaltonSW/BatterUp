package mlb

import "time"

// ScheduleResponse represents the MLB schedule API response.
type ScheduleResponse struct {
	Dates []struct {
		Date  string         `json:"date"`
		Games []ScheduleGame `json:"games"`
	} `json:"dates"`
}

// ScheduleGame holds the fields surfaced on the schedule screen.
type ScheduleGame struct {
	GamePk       int            `json:"gamePk"`
	GameDate     time.Time      `json:"-"`
	GameDateRaw  string         `json:"gameDate"`
	DoubleHeader string         `json:"doubleHeader"`
	GameNumber   int            `json:"gameNumber"`
	Status       GameStatus     `json:"status"`
	Linescore    *GameLineScore `json:"linescore"`
	Teams        ScheduleTeams  `json:"teams"`
}

// ScheduleTeams groups home/away club info for the schedule view.
type ScheduleTeams struct {
	Away ScheduleTeam `json:"away"`
	Home ScheduleTeam `json:"home"`
}

// ScheduleTeam holds high-level team data for display.
type ScheduleTeam struct {
	Team         TeamInfo     `json:"team"`
	LeagueRecord LeagueRecord `json:"leagueRecord"`
	IsWinner     bool         `json:"isWinner"`
}

// TeamInfo covers the common name fields.
type TeamInfo struct {
	TeamName     string `json:"teamName"`
	Abbreviation string `json:"abbreviation"`
}

// LeagueRecord exposes wins/losses for the current team.
type LeagueRecord struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
}

// GameStatus is shared across schedule and game views.
type GameStatus struct {
	AbstractGameCode string `json:"abstractGameCode"`
	DetailedState    string `json:"detailedState"`
	StartTimeTBD     bool   `json:"startTimeTBD"`
	Reason           string `json:"reason"`
}

// GameLineScore summarizes inning status and totals.
type GameLineScore struct {
	CurrentInning        int             `json:"currentInning"`
	CurrentInningOrdinal string          `json:"currentInningOrdinal"`
	InningState          string          `json:"inningState"`
	IsTopInning          bool            `json:"isTopInning"`
	Teams                LineScoreTotals `json:"teams"`
}

// LineScoreTotals shows totals for each club.
type LineScoreTotals struct {
	Away LineScoreTeam `json:"away"`
	Home LineScoreTeam `json:"home"`
}

// LineScoreTeam holds runs/hits/errors for a club.
type LineScoreTeam struct {
	Runs   int `json:"runs"`
	Hits   int `json:"hits"`
	Errors int `json:"errors"`
}

