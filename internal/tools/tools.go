package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/maxghenis/openmessages/internal/app"
)

func Register(s *server.MCPServer, a *app.App) {
	s.AddTool(getMessagesTool(), getMessagesHandler(a))
	s.AddTool(getConversationTool(), getConversationHandler(a))
	s.AddTool(searchMessagesTool(), searchMessagesHandler(a))
	s.AddTool(sendMessageTool(), sendMessageHandler(a))
	s.AddTool(listConversationsTool(), listConversationsHandler(a))
	s.AddTool(listContactsTool(), listContactsHandler(a))
	s.AddTool(getStatusTool(), getStatusHandler(a))
}

func strArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intArg(args map[string]any, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(text)},
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(msg)},
		IsError: true,
	}
}
