package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
)

func RunApp(ctx context.Context, cfg config.Config) error {
	start := time.Now()

	deps := RealDeps{
		GitHub: &fetcher.GitHubAPI{},
	}

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
