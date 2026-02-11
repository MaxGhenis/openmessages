package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/maxghenis/openmessage/internal/app"
)

func getMessagesTool() mcp.Tool {
	return mcp.NewTool("get_messages",
		mcp.WithDescription("Get recent messages with optional filters by phone number, date range, and limit"),
		mcp.WithString("phone_number", mcp.Description("Filter by sender phone number")),
		mcp.WithString("after", mcp.Description("Only messages after this ISO-8601 date (e.g., 2026-02-01)")),
		mcp.WithString("before", mcp.Description("Only messages before this ISO-8601 date")),
		mcp.WithNumber("limit", mcp.Description("Maximum messages to return (default 20)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func getMessagesHandler(a *app.App) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		phone := strArg(args, "phone_number")
		limit := intArg(args, "limit", 20)

		var afterMS, beforeMS int64
		if after := strArg(args, "after"); after != "" {
			t, err := time.Parse("2006-01-02", after)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid 'after' date: %v", err)), nil
			}
			afterMS = t.UnixMilli()
		}
		if before := strArg(args, "before"); before != "" {
			t, err := time.Parse("2006-01-02", before)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid 'before' date: %v", err)), nil
			}
			beforeMS = t.Add(24*time.Hour - time.Millisecond).UnixMilli()
		}

		msgs, err := a.Store.GetMessages(phone, afterMS, beforeMS, limit)
		if err != nil {
			return errorResult(fmt.Sprintf("query failed: %v", err)), nil
		}

		if len(msgs) == 0 {
			return textResult("No messages found."), nil
		}

		var sb strings.Builder
		sb.WriteString(messagePreamble)
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
			display := formatMessageBody(m.Body, m.MediaID, m.MimeType, m.MessageID)
			fmt.Fprintf(&sb, "[%s] %s %s: «%s»\n", ts, direction, sender, display)
		}
		return textResult(sb.String()), nil
	}
}
