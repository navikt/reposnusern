package runner

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
)

func RunApp(ctx context.Context, cfg config.Config, deps RunnerDeps) error {
	start := time.Now()

	err := Run(ctx, cfg, deps)
	if err != nil {
		slog.Error("Runner feilet", "error", err)
		os.Exit(1)
	}

	LogMemoryStats()

	elapsed := time.Since(start)
	slog.Info("✅ Ferdig!", "varighet", elapsed.String())

	return nil
}

func LogMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("📊 Minnebruk",
		"alloc", ByteSize(m.Alloc),
		"totalAlloc", ByteSize(m.TotalAlloc),
		"sys", ByteSize(m.Sys),
		"numGC", m.NumGC)
}

func ByteSize(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func SetupLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))
	slog.SetDefault(logger)
}

func CheckDatabaseConnection(ctx context.Context, dsn string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("Kunne ikke åpne DB-forbindelse", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("⚠️ Klarte ikke å lukke testDB", "error", err)
		}
	}()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("❌ Klarte ikke å nå databasen", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ DB-tilkobling OK")
}
