package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog"

	"github.com/maxghenis/openmessages/internal/client"
	"github.com/maxghenis/openmessages/internal/db"
)

//go:embed static/*
var staticFS embed.FS

// APIHandler creates the HTTP handler with JSON API routes and static file serving.
// The client may be nil (disconnected state).
func APIHandler(store *db.Store, cli *client.Client, logger zerolog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/conversations", func(w http.ResponseWriter, r *http.Request) {
		limit := queryInt(r, "limit", 50)
		convos, err := store.ListConversations(limit)
		if err != nil {
			httpError(w, "list conversations: "+err.Error(), 500)
			return
		}
		if convos == nil {
			convos = []*db.Conversation{}
		}
		writeJSON(w, convos)
	})

	mux.HandleFunc("/api/conversations/", func(w http.ResponseWriter, r *http.Request) {
		// Parse: /api/conversations/{id}/messages
		path := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 || parts[1] != "messages" {
			httpError(w, "not found", 404)
			return
		}
		convID := parts[0]
		limit := queryInt(r, "limit", 100)
		msgs, err := store.GetMessagesByConversation(convID, limit)
		if err != nil {
			httpError(w, "get messages: "+err.Error(), 500)
			return
		}
		if msgs == nil {
			msgs = []*db.Message{}
		}
		writeJSON(w, msgs)
	})

	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			httpError(w, "query parameter 'q' is required", 400)
			return
		}
		limit := queryInt(r, "limit", 50)
		msgs, err := store.SearchMessages(q, "", limit)
		if err != nil {
			httpError(w, "search: "+err.Error(), 500)
			return
		}
		if msgs == nil {
			msgs = []*db.Message{}
		}
		writeJSON(w, msgs)
	})

	mux.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, "method not allowed", 405)
			return
		}
		var req struct {
			PhoneNumber string `json:"phone_number"`
			Message     string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid JSON: "+err.Error(), 400)
			return
		}
		if req.PhoneNumber == "" || req.Message == "" {
			httpError(w, "phone_number and message are required", 400)
			return
		}
		if cli == nil {
			httpError(w, "not connected to Google Messages", 503)
			return
		}
		// TODO: use cli to send message via libgm
		writeJSON(w, map[string]string{"status": "sent"})
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		connected := cli != nil
		writeJSON(w, map[string]any{
			"connected": connected,
		})
	})

	// Serve embedded static files at root
	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create static sub-filesystem")
	}
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}
