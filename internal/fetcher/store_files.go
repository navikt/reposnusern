package fetcher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
)

func StoreRepoDumpJSON(dir, org string, repos []map[string]interface{}) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("kunne ikke opprette katalog %s: %w", dir, err)
	}

	rawOut, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return fmt.Errorf("kunne ikke serialisere repos til JSON: %w", err)
	}

	rawFile := path.Join(dir, fmt.Sprintf("%s_repos_raw_dump.json", org))
	if err := os.WriteFile(rawFile, rawOut, 0644); err != nil {
		return fmt.Errorf("kunne ikke skrive til fil %s: %w", rawFile, err)
	}

	slog.Info("Lagret full repo-metadata", "count", len(repos), "file", rawFile)
	return nil
}
