package mlb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := f(req)
	if resp == nil {
		return nil, errors.New("roundTripFunc returned nil response")
	}
	return resp, nil
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestClientFetchScheduleSuccess(t *testing.T) {
	date := time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC)

	rt := roundTripFunc(func(req *http.Request) *http.Response {
		if got := req.Header.Get("User-Agent"); got != userAgent {
			t.Fatalf("expected user agent %q, got %q", userAgent, got)
		}
		if req.URL.Scheme != "https" {
			t.Fatalf("unexpected scheme %s", req.URL.Scheme)
		}
		if req.URL.Host != "statsapi.mlb.com" {
			t.Fatalf("unexpected host %s", req.URL.Host)
		}
		if got := req.URL.Query().Get("date"); got != date.Format("01/02/2006") {
			t.Fatalf("expected date query %q, got %q", date.Format("01/02/2006"), got)
		}
		body := fmt.Sprintf(`{
            "dates": [
                {
                    "date": "2024-04-01",
                    "games": [
                        {
                            "gamePk": 123,
                            "gameDate": "2024-04-01T19:05:00Z",
                            "doubleHeader": "N",
                            "gameNumber": 1,
                            "status": {"abstractGameCode": "P", "detailedState": "Scheduled"},
                            "teams": {
                                "away": {
                                    "team": {"teamName": "Away", "abbreviation": "AWY"},
                                    "leagueRecord": {"wins": 10, "losses": 5},
                                    "isWinner": false
                                },
                                "home": {
                                    "team": {"teamName": "Home", "abbreviation": "HME"},
                                    "leagueRecord": {"wins": 8, "losses": 7},
                                    "isWinner": false
                                }
                            }
                        }
                    ]
                }
            ]
        }`)
		return response(http.StatusOK, body)
	})

	client := &Client{http: &http.Client{Transport: rt}}

	resp, err := client.FetchSchedule(context.Background(), date)
	if err != nil {
		t.Fatalf("FetchSchedule returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("FetchSchedule returned nil response")
	}
	if len(resp.Dates) != 1 {
		t.Fatalf("expected 1 date, got %d", len(resp.Dates))
	}
	games := resp.Dates[0].Games
	if len(games) != 1 {
		t.Fatalf("expected 1 game, got %d", len(games))
	}
	if games[0].GameDate.IsZero() {
		t.Fatalf("expected GameDate to be parsed")
	}
	want := time.Date(2024, time.April, 1, 19, 5, 0, 0, time.UTC)
	if !games[0].GameDate.Equal(want) {
		t.Fatalf("expected GameDate %v, got %v", want, games[0].GameDate)
	}
}

func TestClientFetchScheduleErrorStatus(t *testing.T) {
	rt := roundTripFunc(func(*http.Request) *http.Response {
		return response(http.StatusInternalServerError, "oops")
	})
	client := &Client{http: &http.Client{Transport: rt}}

	_, err := client.FetchSchedule(context.Background(), time.Now())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if want := "unexpected status 500"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error to contain %q, got %q", want, err.Error())
	}
}

func TestClientFetchGameSuccess(t *testing.T) {
	rt := roundTripFunc(func(req *http.Request) *http.Response {
		if !strings.Contains(req.URL.Path, "/api/v1.1/game/456/feed/live") {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		body := `{
            "gameData": {
                "status": {"abstractGameCode": "F"},
                "teams": {
                    "home": {"teamName": "Home", "abbreviation": "HME", "record": {"wins": 1, "losses": 2}},
                    "away": {"teamName": "Away", "abbreviation": "AWY", "record": {"wins": 3, "losses": 4}}
                },
                "venue": {"name": "Venue"},
                "datetime": {"dateTime": "2024-04-01T19:05:00Z"},
                "probablePitchers": {"home": null, "away": null}
            },
            "liveData": {
                "plays": {"currentPlay": {"about": {"inning": 1, "halfInning": "top", "isTopInning": true}, "playEvents": [], "result": {}, "count": {}, "matchup": {}}, "allPlays": []},
                "linescore": {"currentInning": 1, "currentInningOrdinal": "1st", "inningState": "Top", "isTopInning": true, "teams": {"away": {"runs": 0, "hits": 0, "errors": 0}, "home": {"runs": 0, "hits": 0, "errors": 0}}},
                "boxscore": {"teams": {"home": {"players": {}}, "away": {"players": {}}}},
                "decisions": null
            },
            "metaData": {"wait": 30, "timeStamp": "20240401"}
        }`
		return response(http.StatusOK, body)
	})

	client := &Client{http: &http.Client{Transport: rt}}

	feed, err := client.FetchGame(context.Background(), 456)
	if err != nil {
		t.Fatalf("FetchGame returned error: %v", err)
	}
	if feed == nil {
		t.Fatalf("expected feed, got nil")
	}
	if feed.MetaData.Wait != 30 {
		t.Fatalf("expected wait 30, got %d", feed.MetaData.Wait)
	}
}

func TestPlayEventTypeUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want string
	}{
		{"null", []byte("null"), ""},
		{"object", []byte(`{"description":"Called Strike"}`), "Called Strike"},
		{"string", []byte(`"Swinging Strike"`), "Swinging Strike"},
		{"empty", []byte{}, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var typ PlayEventType
			if err := typ.UnmarshalJSON(tc.data); err != nil {
				t.Fatalf("UnmarshalJSON returned error: %v", err)
			}
			if typ.Description != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, typ.Description)
			}
		})
	}
}
