package fetcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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

func GetJSONMap(url, token string) map[string]interface{} {
	var out map[string]interface{}
	err := GetJSONWithRateLimit(url, token, &out)
	if err != nil {
		slog.Error("Feil ved henting", "url", url, "error", err)
		return nil
	}
	return out
}

func GetReadme(fullName, token string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/readme", fullName)
	var payload map[string]interface{}
	if err := GetJSONWithRateLimit(url, token, &payload); err != nil {
		return ""
	}
	if content, ok := payload["content"].(string); ok {
		decoded, _ := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		return string(decoded)
	}
	return ""
}
