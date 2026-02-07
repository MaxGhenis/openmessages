package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/maxghenis/openmessages/internal/app"
)

func getStatusTool() mcp.Tool {
	return mcp.NewTool("get_status",
		mcp.WithDescription("Get connection status and paired phone information"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func getStatusHandler(a *app.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var sb strings.Builder

		if a.Client == nil {
			sb.WriteString("Status: not connected\n")
			sb.WriteString("Run 'gmessages-mcp pair' to connect.\n")
			return textResult(sb.String()), nil
		}

		connected := a.Client.GM.IsConnected()
		loggedIn := a.Client.GM.IsLoggedIn()

		sb.WriteString("Status: ")
		if connected {
			sb.WriteString("connected\n")
		} else {
			sb.WriteString("disconnected\n")
		}

		fmt.Fprintf(&sb, "Logged in: %v\n", loggedIn)

		if ad := a.Client.GM.AuthData; ad != nil {
			if ad.Mobile != nil {
				fmt.Fprintf(&sb, "Phone ID: %s\n", ad.Mobile.GetSourceID())
			}
			if ad.Browser != nil {
				fmt.Fprintf(&sb, "Browser ID: %s\n", ad.Browser.GetSourceID())
			}
			fmt.Fprintf(&sb, "Session ID: %s\n", ad.SessionID.String())
		}

		fmt.Fprintf(&sb, "Data dir: %s\n", a.DataDir)

		return textResult(sb.String()), nil
	}
}
