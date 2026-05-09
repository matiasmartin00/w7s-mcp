package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcpgoserver "github.com/mark3labs/mcp-go/server"
	"github.com/matiasmartin00/w7s-mcp/internal/mcpserver"
)

// Build-time variables set by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Structured logger — writes to stderr so it doesn't pollute the transport output.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting w7s-mcp server",
		"version", version,
		"commit", commit,
		"built", date,
		"transport", "streamable-http",
	)

	s := mcpserver.New()
	httpSrv := buildServer(s)

	addr := ":" + defaultPort()
	slog.Info("listening", "addr", addr, "endpoint", addr+"/mcp")

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

// buildServer wires the MCP server into a net/http.Server with the /mcp endpoint.
func buildServer(s *mcpgoserver.MCPServer) *http.Server {
	streamHandler := mcpgoserver.NewStreamableHTTPServer(s,
		mcpgoserver.WithEndpointPath("/mcp"),
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", streamHandler)
	// Health check endpoint — useful for infra probes.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Version endpoint — returns build info as plain text.
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "w7s-mcp %s (%s) built %s\n", version, commit, date)
	})

	return &http.Server{
		Addr:              ":" + defaultPort(),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// defaultPort returns the PORT from environment, defaulting to "4004".
func defaultPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "4004"
}
