package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/maxghenis/openmessages/internal/app"
)

func listConversationsTool() mcp.Tool {
	return mcp.NewTool("list_conversations",
		mcp.WithDescription("List recent conversations, sorted by most recent message"),
		mcp.WithNumber("limit", mcp.Description("Maximum conversations to return (default 20)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func listConversationsHandler(a *app.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		limit := intArg(args, "limit", 20)

		convs, err := a.Store.ListConversations(limit)
		if err != nil {
			return errorResult(fmt.Sprintf("query failed: %v", err)), nil
		}

		if len(convs) == 0 {
			return textResult("No conversations found. Messages may not have synced yet."), nil
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "%d conversations:\n\n", len(convs))
		for _, c := range convs {
			ts := time.UnixMilli(c.LastMessageTS).Format(time.RFC3339)
			group := ""
			if c.IsGroup {
				group = " [group]"
			}
			unread := ""
			if c.UnreadCount > 0 {
				unread = fmt.Sprintf(" (%d unread)", c.UnreadCount)
			}
			fmt.Fprintf(&sb, "- %s%s%s (ID: %s, last: %s)\n", c.Name, group, unread, c.ConversationID, ts)
		}
		return textResult(sb.String()), nil
	}
}
