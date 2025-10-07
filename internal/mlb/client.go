package mlb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	userAgent        = "go.dalton.dog/batterup/1.0"
	scheduleEndpoint = "https://statsapi.mlb.com/api/v1/schedule"
	gameEndpointFmt  = "https://statsapi.mlb.com/api/v1.1/game/%d/feed/live"
)

func (c *Client) get(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// Client wraps MLB StatsAPI access used by the TUI.
type Client struct {
	http *http.Client
}

// NewClient returns a Client with a default HTTP client and timeout.
func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 15 * time.Second}}
}

// FetchSchedule retrieves the MLB schedule for a specific day.
func (c *Client) FetchSchedule(ctx context.Context, date time.Time) (*ScheduleResponse, error) {
	queryVals := url.Values{}
	queryVals.Set("sportId", "1")
	queryVals.Set("hydrate", "team,linescore")
	queryVals.Set("date", date.Format("01/02/2006"))

	endpoint := fmt.Sprintf("%s?%s", scheduleEndpoint, queryVals.Encode())

	var resp ScheduleResponse
	if err := c.get(ctx, endpoint, &resp); err != nil {
		return nil, fmt.Errorf("schedule request failed: %w", err)
	}

	for dateIdx := range resp.Dates {
		for gameIdx := range resp.Dates[dateIdx].Games {
			game := &resp.Dates[dateIdx].Games[gameIdx]
			parsed, err := time.Parse(time.RFC3339, game.GameDateRaw)
			if err == nil {
				game.GameDate = parsed
			}
		}
	}

	return &resp, nil
}

// FetchGame returns the live feed for a specific MLB game.
func (c *Client) FetchGame(ctx context.Context, gameID int) (*GameFeed, error) {
	endpoint := fmt.Sprintf(gameEndpointFmt, gameID)
	var feed GameFeed
	if err := c.get(ctx, endpoint, &feed); err != nil {
		return nil, fmt.Errorf("game feed request failed: %w", err)
	}
	return &feed, nil
}
