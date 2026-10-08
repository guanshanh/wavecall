package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/guanshanh/wavecall/internal/auth"
	"github.com/guanshanh/wavecall/internal/dispatch"
)

func main() {
	configPath := flag.String("config", "configs/cluster.example.toml", "path to shared cluster TOML (dispatch + nodes)")
	usersPath := flag.String("users", "configs/users.toml", "path to users.toml (same file as the SFU -users)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := dispatch.LoadConfig(*configPath)
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	tab, err := dispatch.LoadTable(cfg.Nodes)
	if err != nil {
		slog.Error("load table", "err", err)
		os.Exit(1)
	}

	users, err := auth.Load(*usersPath)
	if err != nil {
		slog.Error("load users", "err", err)
		os.Exit(1)
	}
	srv := &http.Server{Addr: cfg.Bind, Handler: dispatch.NewHandler(tab, users)}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down dispatch")
		_ = srv.Close()
	}()

	slog.Info("dispatch listening", "addr", cfg.Bind)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
