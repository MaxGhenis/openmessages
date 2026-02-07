# openmessages

MCP server for Google Messages (SMS + RCS). Gives Claude Code full read/write access to text messages without browser automation.

Built on [mautrix/gmessages](https://github.com/mautrix/gmessages) (libgm) for the Google Messages protocol and [mcp-go](https://github.com/mark3labs/mcp-go) for the MCP server.

## Setup

### 1. Build

```bash
cd openmessages
go build -o openmessages .
```

### 2. Pair with your phone

```bash
./openmessages pair
```

Scan the QR code with Google Messages (Settings > Device pairing > Pair a device). Session is saved to `~/.local/share/openmessages/session.json`.

### 3. Add to MCP config

Add to `~/.mcp.json`:

```json
{
  "mcpServers": {
    "gmessages": {
      "command": "/path/to/openmessages",
      "args": ["serve"]
    }
  }
}
```

### 4. Restart Claude Code

The 7 tools will appear automatically.

## Tools

| Tool | Description |
|------|-------------|
| `get_messages` | Recent messages with filters (phone, date range, limit) |
| `get_conversation` | Messages in a specific conversation |
| `search_messages` | Full-text search across all messages |
| `send_message` | Send SMS/RCS to a phone number |
| `list_conversations` | List recent conversations |
| `list_contacts` | List/search contacts |
| `get_status` | Connection status and paired phone info |

## Configuration

| Env var | Default | Purpose |
|---------|---------|---------|
| `OPENMESSAGES_DATA_DIR` | `~/.local/share/openmessages` | Data directory (DB + session) |
| `OPENMESSAGES_LOG_LEVEL` | `info` | Log level (debug/info/warn/error/trace) |

## Architecture

- **libgm** handles the Google Messages protocol (pairing, encryption, long-polling)
- **SQLite** (WAL mode) stores messages, conversations, and contacts locally
- Real-time events from the phone are written to SQLite as they arrive
- MCP tool handlers read from SQLite for queries, call libgm for sends
- Auth tokens auto-refresh and persist to `session.json`

## Development

```bash
go test ./...
```
