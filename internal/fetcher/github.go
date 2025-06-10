package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// Injecter en klient (for testbarhet)
var HttpClient = http.DefaultClient

func DoRequestWithRateLimit(method, url, token string, body []byte, out interface{}) error {
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

		resp, err := HttpClient.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("advarsel: klarte ikke å lukke body: %v", err)
			}
		}()

		if rl := resp.Header.Get("X-RateLimit-Remaining"); rl == "0" {
			reset := resp.Header.Get("X-RateLimit-Reset")
			if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
				wait := time.Until(time.Unix(ts, 0)) + time.Second
				slog.Warn("Rate limit nådd", "venter", wait.Truncate(time.Second))
				time.Sleep(wait)
				continue
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			slog.Error("GitHub-feil", "status", resp.StatusCode, "body", string(bodyBytes))
			return fmt.Errorf("GitHub API-feil: status %d – %s", resp.StatusCode, string(bodyBytes))
		}

		return json.NewDecoder(resp.Body).Decode(out)
	}
}

func GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=all&page=%d", cfg.Org, page)
	var pageRepos []models.RepoMeta
	slog.Info("Henter repos", "page", page)
	err := DoRequestWithRateLimit("GET", url, cfg.Token, nil, &pageRepos)
	if err != nil {
		return nil, err
	}

	return pageRepos, nil
}
