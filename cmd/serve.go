package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"
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

	// Start web server in background
	port := os.Getenv("OPENMESSAGES_PORT")
	if port == "" {
		port = "7007"
	}
	httpHandler := web.APIHandler(a.Store, a.Client, logger)
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", port, err)
	}
	go func() {
		logger.Info().Str("port", port).Msg("Web UI available at http://localhost:" + port)
		if err := http.Serve(ln, httpHandler); err != nil {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Create MCP server
	s := server.NewMCPServer(
		"openmessages",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	tools.Register(s, a)

	// Serve MCP over stdio (blocks)
	logger.Info().Msg("Starting MCP server on stdio")
	if err := server.ServeStdio(s); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
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
