package logger

import (
	"log/slog"
	"os"
)

var ProgramLevel = new(slog.LevelVar)

// SetupLogger initialiserer loggeren med JSON-format og standard nivå.
func SetupLogger() {
	ProgramLevel.Set(slog.LevelInfo)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     ProgramLevel,
		AddSource: false,
	}))
	slog.SetDefault(logger)
}

// SetDebug setter loggnivået til Debug hvis debug er true.
func SetDebug(debug bool) {
	if debug {
		ProgramLevel.Set(slog.LevelDebug)
	}
}
