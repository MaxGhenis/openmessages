---
description: Context for OpenMessages MCP integration
---

# OpenMessages

OpenMessages connects to Google Messages via the libgm protocol, bridging SMS/RCS to a local web UI and MCP server.

## Architecture

- **Go backend** (`openmessages serve`): Connects to Google Messages, serves web UI on port 7007, and exposes MCP via SSE at `/mcp/sse`
- **macOS app**: Swift wrapper that launches the Go backend and displays the web UI in a WKWebView
- **MCP server**: SSE transport at `http://localhost:7007/mcp/sse` — provides tools for listing conversations, reading/sending messages, searching

## MCP tools

The MCP server exposes these tools (prefix: `mcp__openmessages__`):

| Tool | Description | Key params |
|------|-------------|------------|
| `list_conversations` | Recent conversations with unread counts | `limit` |
| `get_conversation` | Single conversation details | `conversation_id` |
| `get_messages` | Messages in a conversation | `conversation_id`, `limit` |
| `search_messages` | Full-text search across all messages | `query`, `limit` |
| `send_message` | Send SMS/RCS to a phone number | `phone_number`, `message` |
| `list_contacts` | Known contacts from message history | — |
| `get_status` | Connection status to Google Messages | — |

## Prerequisites

The OpenMessages macOS app must be running (it starts the backend). If the MCP connection fails, the user needs to launch OpenMessages.app.
