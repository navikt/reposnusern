package main

import (
	"log/slog"
	"os"

	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))
	slog.SetDefault(logger)

	org := os.Getenv("ORG")
	if org == "" {
		slog.Error("Du må angi organisasjon via ORG=<orgnavn>")
		os.Exit(1)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		slog.Error("Mangler GITHUB_TOKEN i environment")
		os.Exit(1)
	}

	// Hent full repo-metadata som map
	repos := fetcher.GetAllRepos(org, token)

	if err := fetcher.StoreRepoDumpJSON("data", org, repos); err != nil {
		slog.Error("Feil under lagring av repo-dump", "error", err)
		os.Exit(1)
	}

	//allData := fetcher.GetDetailsActiveRepos(org, token, repos)
	allData := fetcher.GetDetailsActiveReposGraphQL(org, token, repos)

	if err := fetcher.StoreRepoDetailedJSON("data", org, allData); err != nil {
		slog.Error("Feil under lagring av repo-dump", "error", err)
		os.Exit(1)
	}

}
