package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"github.com/maxghenis/openmessages/cmd"
)

func main() {
	level := cmd.LogLevel()
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Timestamp().Logger().Level(level)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: openmessages <pair|serve>")
		fmt.Fprintln(os.Stderr, "  pair   - Pair with your phone via QR code")
		fmt.Fprintln(os.Stderr, "  serve  - Start MCP server (stdio)")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "pair":
		err = cmd.RunPair(logger)
	case "serve":
		err = cmd.RunServe(logger)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "Usage: openmessages <pair|serve>")
		os.Exit(1)
	}

	if err != nil {
		logger.Fatal().Err(err).Msg("Fatal error")
	}
}
