package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client interface {
	FetchAllStreamers(ctx context.Context) ([]Streamer, error)
}

type HTTPClient struct {
	BaseURL string
}

type StreamerSubscriber struct {
	PlayerID string `json:"playerId"`
}

type Streamer struct {
	StreamerID  string               `json:"streamerId"`
	Subscribers []StreamerSubscriber `json:"subscribers"`
}

func (c *HTTPClient) FetchAllStreamers(ctx context.Context) ([]Streamer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/streamers", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("signal server returned status %d", resp.StatusCode)
	}

	var out []Streamer
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}
