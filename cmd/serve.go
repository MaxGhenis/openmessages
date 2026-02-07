package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/maxghenis/openmessages/internal/app"
	"github.com/maxghenis/openmessages/internal/tools"
	"github.com/maxghenis/openmessages/internal/web"
)

func RunServe(logger zerolog.Logger) error {
	a, err := app.New(logger)
	if err != nil {
		return fmt.Errorf("init app: %w", err)
	}
	defer a.Close()

	// Connect to Google Messages
	if err := a.LoadAndConnect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// Backfill existing conversations and messages
	go func() {
		if err := a.Backfill(); err != nil {
			logger.Warn().Err(err).Msg("Backfill failed")
		}
	}()

	// Start web server
	port := os.Getenv("OPENMESSAGES_PORT")
	if port == "" {
		port = "7007"
	}

	// Create MCP server
	mcpSrv := mcpserver.NewMCPServer(
		"openmessages",
		"0.1.0",
		mcpserver.WithToolCapabilities(true),
	)
	tools.Register(mcpSrv, a)

	// Create SSE transport for MCP, mounted at /mcp/
	sseSrv := mcpserver.NewSSEServer(mcpSrv,
		mcpserver.WithBaseURL(fmt.Sprintf("http://localhost:%s", port)),
		mcpserver.WithStaticBasePath("/mcp"),
	)

	httpHandler := web.APIHandler(a.Store, a.Client, logger, sseSrv, a.DeepBackfill)
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", port, err)
	}
	go func() {
		logger.Info().Str("port", port).Msg("Web UI available at http://localhost:" + port)
		logger.Info().Str("port", port).Msg("MCP SSE available at http://localhost:" + port + "/mcp/sse")
		if err := http.Serve(ln, httpHandler); err != nil {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Block until signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info().Msg("Shutting down")
	return nil
}

// LogLevel returns the zerolog level based on OPENMESSAGES_LOG_LEVEL env var.
func LogLevel() zerolog.Level {
	switch os.Getenv("OPENMESSAGES_LOG_LEVEL") {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "trace":
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}
