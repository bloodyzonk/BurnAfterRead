package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	// Get DB path from env or default
	Config := LoadConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Init DB
	db, err := InitStore(Config.DBPath)
	if err != nil {
		slog.Error("failed to initialize store",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer db.Close()

	// Create server
	srv := NewServer(db, Config, logger)

	// Background cleanup
	go srv.StartCleanup()

	// Get address from env or default
	addr := ":" + Config.Port

	slog.Info("server starting",
		slog.String("addr", addr),
	)

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		slog.Error("server crashed",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
