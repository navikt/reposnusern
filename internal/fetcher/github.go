package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
)

type TreeFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

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

func GetAllRepos(cfg config.Config) []map[string]interface{} {
	repos := []map[string]interface{}{}
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=all&page=%d", cfg.Org, page)
		var pageRepos []map[string]interface{}
		slog.Info("Henter repos", "page", page)
		err := GetJSONWithRateLimit(url, cfg.Token, &pageRepos)
		if err != nil {
			slog.Error("Kunne ikke hente repo-metadata", "error", err)
			os.Exit(1)
		}
		if len(pageRepos) == 0 {
			break
		}

		if cfg.Debug {
			// Shuffle og velg 3 tilfeldig
			rand.Shuffle(len(pageRepos), func(i, j int) {
				pageRepos[i], pageRepos[j] = pageRepos[j], pageRepos[i]
			})
			repos = append(repos, pageRepos[:min(10, len(pageRepos))]...)
			break
		} else {
			repos = append(repos, pageRepos...)
		}

		page++
	}
	return repos
}
