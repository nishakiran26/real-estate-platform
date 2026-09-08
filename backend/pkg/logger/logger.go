package logger

import (
	"log/slog"
	"os"
	"strings"
)

var Log *slog.Logger

func Init(env string) {
	lvl := slog.LevelInfo
	switch strings.ToLower(env) {
	case "production", "prod":
		lvl = slog.LevelInfo
	case "development", "dev":
		lvl = slog.LevelDebug
	}
	var handler slog.Handler
	if env == "development" || env == "dev" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	}
	Log = slog.New(handler)
}
