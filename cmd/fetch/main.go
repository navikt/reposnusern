package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/storage"

	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()

	db, err := sql.Open("sqlite", "file:temp.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// --- Initialiser database ---
	schemaBytes, err := os.ReadFile("db/schema.sql")
	if err != nil {
		log.Fatalf("kunne ikke lese schema.sql: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(schemaBytes)); err != nil {
		log.Fatalf("kunne ikke kjøre schema.sql: %v", err)
	}

	q := storage.New(db)

	// --- Les og importer repository-data ---
	repoBytes, err := os.ReadFile("tempdata/repos_raw_dump.json")
	if err != nil {
		log.Fatal("klarte ikke å lese repos_raw_dump.json:", err)
	}

	var repos []map[string]interface{}
	if err := json.Unmarshal(repoBytes, &repos); err != nil {
		log.Fatal("klarte ikke å parse JSON for repoer:", err)
	}

	for _, r := range repos {
		topics := []string{}
		if t, ok := r["topics"].([]interface{}); ok {
			for _, topic := range t {
				if s, ok := topic.(string); ok {
					topics = append(topics, s)
				}
			}
		}

		_ = q.InsertRepo(ctx, storage.InsertRepoParams{
			ID:           int64(r["id"].(float64)),
			Name:         getString(r["name"]),
			FullName:     getString(r["full_name"]),
			Description:  getString(r["description"]),
			Stars:        int64(r["stargazers_count"].(float64)),
			Forks:        int64(r["forks_count"].(float64)),
			Archived:     r["archived"].(bool),
			Private:      r["private"].(bool),
			IsFork:       r["fork"].(bool),
			Language:     getString(r["language"]),
			SizeMb:       float64(r["size"].(float64)) / 1024.0,
			UpdatedAt:    getString(r["updated_at"]),
			PushedAt:     getString(r["pushed_at"]),
			CreatedAt:    getString(r["created_at"]),
			HtmlUrl:      getString(r["html_url"]),
			Topics:       strings.Join(topics, ","),
			Visibility:   getString(r["visibility"]),
			License:      extractLicenseName(r["license"]),
			OpenIssues:   int64(r["open_issues"].(float64)),
			LanguagesUrl: getString(r["languages_url"]),
		})
	}

	log.Println("📦 Repositories importert.")

	repoNameToID := make(map[string]int64)
	for _, r := range repos {
		name := r["name"].(string)
		repoID := int64(r["id"].(float64))
		repoNameToID[name] = repoID
	}

	// --- Les og importer Dockerfiles ---
	dfBytes, err := os.ReadFile("tempdata/dockerfiles.json")
	if err != nil {
		log.Fatalf("kunne ikke lese dockerfiles.json: %v", err)
	}

	var dockerfiles []struct {
		RepoID     int64  `json:"repo_id"`
		FullName   string `json:"full_name"`
		Dockerfile string `json:"dockerfile"`
	}
	if err := json.Unmarshal(dfBytes, &dockerfiles); err != nil {
		log.Fatalf("kunne ikke tolke dockerfiles.json: %v", err)
	}

	for _, df := range dockerfiles {
		err := q.InsertDockerfile(ctx, storage.InsertDockerfileParams{
			RepoID:   df.RepoID,
			FullName: df.FullName,
			Content:  df.Dockerfile,
		})
		if err != nil {
			log.Printf("❌ klarte ikke å lagre Dockerfile for repo %d: %v", df.RepoID, err)
		}
	}
	log.Println("🐳 Dockerfiles importert.")

	// --- Les og importer språkstatistikk ---
	langBytes, err := os.ReadFile("tempdata/repo_lang.json")
	if err != nil {
		log.Fatalf("kunne ikke lese repo_lang.json: %v", err)
	}

	var langs []struct {
		Repo      string           `json:"repo"`
		Languages map[string]int64 `json:"languages"`
	}
	if err := json.Unmarshal(langBytes, &langs); err != nil {
		log.Fatalf("kunne ikke tolke repo_lang.json: %v", err)
	}

	for _, entry := range langs {
		repoID, ok := repoNameToID[entry.Repo]
		if !ok {
			log.Printf("⚠️ repo-navn %s finnes ikke i kartet, hopper over", entry.Repo)
			continue
		}
		for lang, bytes := range entry.Languages {
			err := q.InsertLanguage(ctx, storage.InsertLanguageParams{
				RepoID:   repoID,
				Language: lang,
				Bytes:    bytes,
			})
			if err != nil {
				log.Printf("❌ klarte ikke å lagre språkdata for repo %s: %v", entry.Repo, err)
			}
		}
	}

	log.Println("🗣️ Språkstatistikk importert.")
	log.Println("✅ Ferdig.")
}

func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	return v.(string)
}

func extractLicenseName(license interface{}) string {
	if license == nil {
		return ""
	}
	if l, ok := license.(map[string]interface{}); ok {
		if name, ok := l["name"].(string); ok {
			return name
		}
	}
	return ""
}
