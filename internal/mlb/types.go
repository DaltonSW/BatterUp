package mlb

import (
	"encoding/json"
	"time"
)

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

// GameFeed is the primary game endpoint consumed by the UI.
type GameFeed struct {
	GameData GameData `json:"gameData"`
	LiveData LiveData `json:"liveData"`
	MetaData MetaData `json:"metaData"`
}

// MetaData carries polling meta fields.
type MetaData struct {
	Wait      int    `json:"wait"`
	TimeStamp string `json:"timeStamp"`
}

// GameData holds static information about a particular game.
type GameData struct {
	Status           GameStatus       `json:"status"`
	Teams            GameTeams        `json:"teams"`
	Venue            Venue            `json:"venue"`
	Datetime         GameDateTime     `json:"datetime"`
	ProbablePitchers ProbablePitchers `json:"probablePitchers"`
}

// GameTeams groups home/away teams with more detailed info.
type GameTeams struct {
	Home GameTeam `json:"home"`
	Away GameTeam `json:"away"`
}

// GameTeam extends TeamInfo with record info.
type GameTeam struct {
	TeamName     string `json:"teamName"`
	Abbreviation string `json:"abbreviation"`
	Record       Record `json:"record"`
}

// Record contains wins and losses for the club.
type Record struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
}

// Venue covers the ballpark details.
type Venue struct {
	Name     string        `json:"name"`
	Location VenueLocation `json:"location"`
}

// VenueLocation exposes the city and state for display.
type VenueLocation struct {
	City        string `json:"city"`
	StateAbbrev string `json:"stateAbbrev"`
}

// GameDateTime carries the scheduled start.
type GameDateTime struct {
	DateTime string `json:"dateTime"`
}

// ProbablePitchers references the starting pitchers.
type ProbablePitchers struct {
	Home *PersonRef `json:"home"`
	Away *PersonRef `json:"away"`
}

// PersonRef references a player by identifier.
type PersonRef struct {
	ID int `json:"id"`
}

// LiveData includes the mutable game state.
type LiveData struct {
	Plays     Plays         `json:"plays"`
	Linescore LiveLineScore `json:"linescore"`
	Boxscore  Boxscore      `json:"boxscore"`
	Decisions *Decisions    `json:"decisions"`
}

// Plays contains the current play plus the full history.
type Plays struct {
	CurrentPlay Play   `json:"currentPlay"`
	AllPlays    []Play `json:"allPlays"`
}

// LiveLineScore extends GameLineScore with balls/strikes/outs and inning breakdown.
type LiveLineScore struct {
	CurrentInning        int             `json:"currentInning"`
	CurrentInningOrdinal string          `json:"currentInningOrdinal"`
	InningState          string          `json:"inningState"`
	IsTopInning          bool            `json:"isTopInning"`
	Balls                int             `json:"balls"`
	Strikes              int             `json:"strikes"`
	Outs                 int             `json:"outs"`
	Teams                LineScoreTotals `json:"teams"`
	Offense              OffensiveState  `json:"offense"`
	Innings              []InningLine    `json:"innings"`
}

// OffensiveState indicates which bases are occupied.
type OffensiveState struct {
	First  *BaseRunner `json:"first"`
	Second *BaseRunner `json:"second"`
	Third  *BaseRunner `json:"third"`
}

// BaseRunner is present when a base is occupied.
type BaseRunner struct {
	ID int `json:"id"`
}

// InningLine captures runs scored per inning.
type InningLine struct {
	Num  int            `json:"num"`
	Home InningTeamRuns `json:"home"`
	Away InningTeamRuns `json:"away"`
}

// InningTeamRuns contains the runs for an inning.
type InningTeamRuns struct {
	Runs *int `json:"runs"`
}

// Play represents a single play with all supporting metadata.
type Play struct {
	Result     PlayResult   `json:"result"`
	About      PlayAbout    `json:"about"`
	Count      PlayCount    `json:"count"`
	Matchup    PlayMatchup  `json:"matchup"`
	PlayEvents []PlayEvent  `json:"playEvents"`
	Runners    []PlayRunner `json:"runners"`
	AtBatIndex int          `json:"atBatIndex"`
}

// PlayResult summarises the outcome of a play.
type PlayResult struct {
	Description string `json:"description"`
	Event       string `json:"event"`
	EventType   string `json:"eventType"`
	AwayScore   int    `json:"awayScore"`
	HomeScore   int    `json:"homeScore"`
	RBI         int    `json:"rbi"`
	IsOut       bool   `json:"isOut"`
}

// PlayAbout contains inning, outs, and scoring context.
type PlayAbout struct {
	Inning        int    `json:"inning"`
	HalfInning    string `json:"halfInning"`
	IsTopInning   bool   `json:"isTopInning"`
	IsComplete    bool   `json:"isComplete"`
	IsScoringPlay bool   `json:"isScoringPlay"`
	HasOut        bool   `json:"hasOut"`
}

// PlayCount reflects the ball/strike/out situation at the end of a play.
type PlayCount struct {
	Balls   int `json:"balls"`
	Strikes int `json:"strikes"`
	Outs    int `json:"outs"`
}

// PlayMatchup identifies the batter and pitcher.
type PlayMatchup struct {
	Pitcher PersonRef `json:"pitcher"`
	Batter  PersonRef `json:"batter"`
}

// PlayEvent provides pitch-by-pitch or action detail.
type PlayEvent struct {
	Type          PlayEventType    `json:"type"`
	IsPitch       bool             `json:"isPitch"`
	IsScoringPlay bool             `json:"isScoringPlay"`
	Details       PlayEventDetails `json:"details"`
	Count         PlayCount        `json:"count"`
	PitchData     *PitchData       `json:"pitchData"`
}

// PlayRunner captures how individual runners advance on a play.
type PlayRunner struct {
	Movement RunnerMovement `json:"movement"`
	Details  RunnerDetails  `json:"details"`
}

// RunnerMovement describes the bases a runner traversed.
type RunnerMovement struct {
	OriginBase string `json:"originBase"`
	Start      string `json:"start"`
	End        string `json:"end"`
	IsOut      bool   `json:"isOut"`
	OutBase    string `json:"outBase"`
	OutNumber  *int   `json:"outNumber"`
}

// RunnerDetails contains the identifying information for a runner.
type RunnerDetails struct {
	Event          string     `json:"event"`
	EventType      string     `json:"eventType"`
	Runner         RunnerInfo `json:"runner"`
	RBI            bool       `json:"rbi"`
	Earned         bool       `json:"earned"`
	IsScoringEvent bool       `json:"isScoringEvent"`
}

// RunnerInfo identifies a runner by player id.
type RunnerInfo struct {
	ID int `json:"id"`
}

// PlayEventType is used for human-readable descriptions.
type PlayEventType struct {
	Description string `json:"description"`
}

func (t *PlayEventType) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Description = ""
		return nil
	}
	if len(data) == 0 {
		t.Description = ""
		return nil
	}
	if data[0] == '{' {
		var aux struct {
			Description string `json:"description"`
		}
		if err := json.Unmarshal(data, &aux); err != nil {
			return err
		}
		t.Description = aux.Description
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t.Description = s
	return nil
}

// PlayEventDetails holds textual descriptions and flags.
type PlayEventDetails struct {
	Description   string        `json:"description"`
	Event         string        `json:"event"`
	IsInPlay      bool          `json:"isInPlay"`
	IsStrike      bool          `json:"isStrike"`
	IsBall        bool          `json:"isBall"`
	Type          PlayEventType `json:"type"`
	IsScoringPlay bool          `json:"isScoringPlay"`
}

// PitchData provides velocity information when available.
type PitchData struct {
	StartSpeed float64 `json:"startSpeed"`
}

// Boxscore aggregates player level stats used in various panels.
type Boxscore struct {
	Teams struct {
		Home BoxscoreTeam `json:"home"`
		Away BoxscoreTeam `json:"away"`
	} `json:"teams"`
}

// BoxscoreTeam maps each player by key (e.g. ID12345).
type BoxscoreTeam struct {
	Players map[string]BoxscorePlayer `json:"players"`
}

// BoxscorePlayer is used for pitcher/batter details on the matchup card.
type BoxscorePlayer struct {
	Person       PersonInfo           `json:"person"`
	JerseyNumber string               `json:"jerseyNumber"`
	Stats        BoxscorePlayerStats  `json:"stats"`
	SeasonStats  BoxscorePlayerSeason `json:"seasonStats"`
}

// PersonInfo contains display name information.
type PersonInfo struct {
	FullName string `json:"fullName"`
}

// BoxscorePlayerStats captures current game performance splits.
type BoxscorePlayerStats struct {
	Pitching PitchingStats `json:"pitching"`
	Batting  BattingStats  `json:"batting"`
}

// PitchingStats provides info for the matchup panel.
type PitchingStats struct {
	InningsPitched string `json:"inningsPitched"`
	PitchesThrown  int    `json:"pitchesThrown"`
}

// BattingStats summarises at-bats and hits for the current game.
type BattingStats struct {
	Hits   int `json:"hits"`
	AtBats int `json:"atBats"`
}

// BoxscorePlayerSeason tracks season-long performance.
type BoxscorePlayerSeason struct {
	Pitching SeasonPitching `json:"pitching"`
	Batting  SeasonBatting  `json:"batting"`
}

// SeasonPitching contains wins/losses/ERA strikeouts.
type SeasonPitching struct {
	Wins       int    `json:"wins"`
	Losses     int    `json:"losses"`
	ERA        string `json:"era"`
	StrikeOuts int    `json:"strikeOuts"`
	Saves      int    `json:"saves"`
}

// SeasonBatting contains batting average and home runs.
type SeasonBatting struct {
	AVG      string `json:"avg"`
	HomeRuns int    `json:"homeRuns"`
}

// Decisions lists the pitcher of record(s).
type Decisions struct {
	Winner *PersonRef `json:"winner"`
	Loser  *PersonRef `json:"loser"`
	Save   *PersonRef `json:"save"`
}
