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

var OpenSQL = sql.Open

func RunApp(ctx context.Context, cfg config.Config, deps RunnerDeps) error {
	return RunAppSafe(ctx, cfg, deps)
}

func RunAppSafe(ctx context.Context, cfg config.Config, deps RunnerDeps) error {
	start := time.Now()

	err := Run(ctx, cfg, deps)
	if err != nil {
		slog.Debug("Runner feilet", "error", err)
		return err
	}

	LogMemoryStats()
	slog.Info("Ferdig!", "varighet", time.Since(start).String())
	return nil
}

func LogMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Debug("Minnebruk",
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

func SetupLogger(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: false,
	}))
	slog.SetDefault(logger)
}

func CheckDatabaseConnection(ctx context.Context, dsn string) error {
	db, err := OpenSQL("postgres", dsn)
	if err != nil {
		slog.Debug("Klarte ikke å åpne databaseforbindelse", "dsn", dsn, "error", err)

		return fmt.Errorf("DB open-feil: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		// Lukker eksplisitt på feil, og returnerer
		if cerr := db.Close(); cerr != nil {
			slog.Warn("Klarte ikke å lukke testDB", "error", cerr)
		}
		slog.Debug("Ping mot database feilet", "dsn", dsn, "error", err)

		return fmt.Errorf("DB ping-feil: %w", err)
	}

	// Normal defer for clean exit
	defer func() {
		if cerr := db.Close(); cerr != nil {
			slog.Warn("Klarte ikke å lukke testDB", "error", cerr)
		}
	}()

	slog.Info("DB-tilkobling OK")
	return nil
}
