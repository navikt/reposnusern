package fetcher

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// GetJSONWithRateLimit henter JSON fra en URL og respekterer GitHub rate-limiting.
func GetJSONWithRateLimit(url, token string, out interface{}) error {
	for {
		slog.Info("Henter URL", "url", url)
		req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if rl := resp.Header.Get("X-RateLimit-Remaining"); rl == "0" {
			reset := resp.Header.Get("X-RateLimit-Reset")
			if reset != "" {
				if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
					wait := time.Until(time.Unix(ts, 0)) + time.Second
					slog.Warn("Rate limit nådd", "venter", wait.Truncate(time.Second))
					time.Sleep(wait)
					continue
				}
			}
		}

		return json.NewDecoder(resp.Body).Decode(out)
	}
}
