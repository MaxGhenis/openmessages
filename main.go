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
		fmt.Fprintln(os.Stderr, "Usage: openmessages <pair|serve|send>")
		fmt.Fprintln(os.Stderr, "  pair                          - Pair with your phone via QR code")
		fmt.Fprintln(os.Stderr, "  serve                         - Start MCP server (stdio)")
		fmt.Fprintln(os.Stderr, "  send <conversation_id> <msg>  - Send message to a conversation")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "pair":
		err = cmd.RunPair(logger)
	case "serve":
		err = cmd.RunServe(logger)
	case "send":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: openmessages send <conversation_id> <message>")
			os.Exit(1)
		}
		err = cmd.RunSend(logger, os.Args[2], os.Args[3])
	case "debug-media":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: openmessages debug-media <conversation_id>")
			os.Exit(1)
		}
		err = cmd.RunDebugMedia(logger, os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "Usage: openmessages <pair|serve|send>")
		os.Exit(1)
	}

	if err != nil {
		logger.Fatal().Err(err).Msg("Fatal error")
	}
}
