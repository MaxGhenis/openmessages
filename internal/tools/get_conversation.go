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

func getConversationTool() mcp.Tool {
	return mcp.NewTool("get_conversation",
		mcp.WithDescription("Get messages in a specific conversation by ID"),
		mcp.WithString("conversation_id", mcp.Required(), mcp.Description("The conversation ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum messages to return (default 50)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func getConversationHandler(a *app.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		convID := strArg(args, "conversation_id")
		if convID == "" {
			return errorResult("conversation_id is required"), nil
		}
		limit := intArg(args, "limit", 50)

		msgs, err := a.Store.GetMessagesByConversation(convID, limit)
		if err != nil {
			return errorResult(fmt.Sprintf("query failed: %v", err)), nil
		}

		if len(msgs) == 0 {
			return textResult("No messages found in this conversation."), nil
		}

		var sb strings.Builder
		// Show conversation info
		conv, err := a.Store.GetConversation(convID)
		if err == nil && conv != nil {
			fmt.Fprintf(&sb, "Conversation: %s (ID: %s)\n", conv.Name, conv.ConversationID)
			if conv.IsGroup {
				sb.WriteString("Type: Group\n")
			}
			sb.WriteString("---\n")
		}

		for _, m := range msgs {
			ts := time.UnixMilli(m.TimestampMS).Format(time.RFC3339)
			direction := "←"
			if m.IsFromMe {
				direction = "→"
			}
			sender := m.SenderName
			if sender == "" {
				sender = m.SenderNumber
			}
			if sender == "" {
				sender = "Unknown"
			}
			fmt.Fprintf(&sb, "[%s] %s %s: %s\n", ts, direction, sender, m.Body)
		}
		return textResult(sb.String()), nil
	}
}
