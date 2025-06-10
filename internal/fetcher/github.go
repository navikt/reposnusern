package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

type TreeFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

func doRequestWithRateLimit(method, url, token string, body []byte, out interface{}) error {
	for {
		slog.Info("Henter URL", "url", url)

		req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if rl := resp.Header.Get("X-RateLimit-Remaining"); rl == "0" {
			reset := resp.Header.Get("X-RateLimit-Reset")
			if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
				wait := time.Until(time.Unix(ts, 0)) + time.Second
				slog.Warn("Rate limit nådd", "venter", wait.Truncate(time.Second))
				time.Sleep(wait)
				continue
			}
		}

		return json.NewDecoder(resp.Body).Decode(out)
	}
}

func GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=all&page=%d", cfg.Org, page)
	var pageRepos []models.RepoMeta
	slog.Info("Henter repos", "page", page)
	err := doRequestWithRateLimit("GET", url, cfg.Token, nil, &pageRepos)
	if err != nil {
		return nil, err
	}

	return pageRepos, nil
}
