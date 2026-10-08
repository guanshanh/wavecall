package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/guanshanh/wavecall/internal/auth"
	"github.com/guanshanh/wavecall/internal/config"
	"github.com/guanshanh/wavecall/internal/dispatch"
	"github.com/guanshanh/wavecall/internal/room"
	"github.com/guanshanh/wavecall/internal/signaling"
)

func main() {
	cfg := config.Default()

	flag.IntVar(&cfg.Port, "port", cfg.Port, "HTTP/WebSocket listen port")
	flag.StringVar(&cfg.PublicIP, "public-ip", cfg.PublicIP, "server public IP for ICE host candidates (cloud 1:1 NAT)")
	flag.IntVar(&cfg.UDPPort, "udp-port", cfg.UDPPort, "single UDP port for WebRTC media (0 = ephemeral ports)")
	flag.StringVar(&cfg.STUNServer, "stun", cfg.STUNServer, "optional STUN server fallback")
	webDir := flag.String("web", "", "directory of web client static files (enables static hosting)")
	clusterConfig := flag.String("cluster-config", "", "shared cluster TOML (same file as dispatch -config)")
	nodeID := flag.String("node", "", "node id in cluster config (e.g. n1); requires -cluster-config")
	usersPath := flag.String("users", "configs/users.toml", "path to users.toml (same file as dispatch -users)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := applyClusterFlags(cfg, *clusterConfig, *nodeID); err != nil {
		slog.Error("cluster config", "err", err)
		os.Exit(1)
	}

	users, err := auth.Load(*usersPath)
	if err != nil {
		slog.Error("load users", "err", err)
		os.Exit(1)
	}

	manager := room.NewManager()
	handler, err := signaling.NewHandler(manager, cfg, users)
	if err != nil {
		slog.Error("failed to create signaling handler", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
	if *webDir != "" {
		mux.Handle("/", spaHandler(*webDir))
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down server")
		server.Close()
		handler.Close()
	}()

	slog.Info("wavecall server starting",
		"addr", addr,
		"publicIP", cfg.PublicIP,
		"udpPort", cfg.UDPPort,
		"node", *nodeID,
	)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

// applyClusterFlags overlays PublicIP / UDPPort / listen port from the shared
// node table. Explicit -port / -public-ip / -udp-port flags win over the file.
func applyClusterFlags(cfg *config.Config, clusterPath, nodeID string) error {
	if clusterPath == "" && nodeID == "" {
		return nil
	}
	if clusterPath == "" || nodeID == "" {
		return fmt.Errorf("-cluster-config and -node must be set together")
	}

	fileCfg, err := dispatch.LoadConfig(clusterPath)
	if err != nil {
		return err
	}
	node, err := dispatch.LookupNode(fileCfg.Nodes, nodeID)
	if err != nil {
		return err
	}
	params, err := dispatch.ServerParamsFromNode(node)
	if err != nil {
		return err
	}

	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if !set["port"] {
		cfg.Port = params.Port
	}
	if !set["public-ip"] {
		cfg.PublicIP = params.PublicIP
	}
	if !set["udp-port"] && params.UDPPort != 0 {
		cfg.UDPPort = params.UDPPort
	}
	return nil
}

func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
