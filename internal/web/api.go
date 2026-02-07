package web

import (
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/rand"
	"time"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"

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
			ConversationID string `json:"conversation_id"`
			Message        string `json:"message"`
			ReplyToID      string `json:"reply_to_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid JSON: "+err.Error(), 400)
			return
		}
		if req.ConversationID == "" || req.Message == "" {
			httpError(w, "conversation_id and message are required", 400)
			return
		}
		if cli == nil {
			httpError(w, "not connected to Google Messages", 503)
			return
		}
		// Fetch conversation to get SIM and participant info
		conv, err := cli.GM.GetConversation(req.ConversationID)
		if err != nil {
			httpError(w, "get conversation: "+err.Error(), 502)
			return
		}

		// Find our participant ID and SIM payload
		var myParticipantID string
		var simPayload *gmproto.SIMPayload
		for _, p := range conv.GetParticipants() {
			if p.GetIsMe() {
				if id := p.GetID(); id != nil {
					myParticipantID = id.GetNumber()
				}
				simPayload = p.GetSimPayload()
				break
			}
		}

		// Also try SIM card from conversation itself
		var convSIMPayload *gmproto.SIMPayload
		if sc := conv.GetSimCard(); sc != nil {
			convSIMPayload = sc.GetSIMData().GetSIMPayload()
		}
		if simPayload == nil {
			simPayload = convSIMPayload
		}

		payload := BuildSendPayload(req.ConversationID, req.Message, req.ReplyToID, myParticipantID, simPayload)

		logger.Info().
			Str("conv_id", req.ConversationID).
			Str("participant_id", myParticipantID).
			Bool("has_sim", simPayload != nil).
			Msg("Sending message")

		resp, err := cli.GM.SendMessage(payload)
		if err != nil {
			httpError(w, "send message: "+err.Error(), 502)
			return
		}
		success := resp.GetStatus() == gmproto.SendMessageResponse_SUCCESS
		if success {
			// Store sent message in DB immediately so UI shows it
			now := time.Now().UnixMilli()
			store.UpsertMessage(&db.Message{
				MessageID:      payload.TmpID,
				ConversationID: req.ConversationID,
				Body:           req.Message,
				IsFromMe:       true,
				TimestampMS:    now,
				Status:         "OUTGOING_SENDING",
				ReplyToID:      req.ReplyToID,
			})
			// Bump conversation to top of list
			store.UpdateConversationTimestamp(req.ConversationID, now)
		}
		writeJSON(w, map[string]any{
			"status":  resp.GetStatus().String(),
			"success": success,
		})
	})

	mux.HandleFunc("/api/media/", func(w http.ResponseWriter, r *http.Request) {
		msgID := strings.TrimPrefix(r.URL.Path, "/api/media/")
		if msgID == "" {
			httpError(w, "message_id required", 400)
			return
		}
		msg, err := store.GetMessageByID(msgID)
		if err != nil {
			httpError(w, "get message: "+err.Error(), 500)
			return
		}
		if msg == nil || msg.MediaID == "" {
			httpError(w, "no media for this message", 404)
			return
		}
		if cli == nil {
			httpError(w, "not connected to Google Messages", 503)
			return
		}
		// Decode hex decryption key
		key, err := hex.DecodeString(msg.DecryptionKey)
		if err != nil {
			httpError(w, "invalid decryption key", 500)
			return
		}
		data, err := cli.GM.DownloadMedia(msg.MediaID, key)
		if err != nil {
			httpError(w, "download media: "+err.Error(), 502)
			return
		}
		w.Header().Set("Content-Type", msg.MimeType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(data)
	})

	mux.HandleFunc("/api/react", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, "method not allowed", 405)
			return
		}
		var req struct {
			MessageID string `json:"message_id"`
			Emoji     string `json:"emoji"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid JSON: "+err.Error(), 400)
			return
		}
		if req.MessageID == "" || req.Emoji == "" {
			httpError(w, "message_id and emoji are required", 400)
			return
		}
		if cli == nil {
			httpError(w, "not connected to Google Messages", 503)
			return
		}
		// TODO: call cli.GM.SendReaction
		writeJSON(w, map[string]string{"status": "ok"})
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

// BuildSendPayload constructs a SendMessageRequest matching the format used by
// the mautrix bridge: MessageInfo array (not MessagePayloadContent), TmpID in 3
// places, SIMPayload, and ParticipantID.
func BuildSendPayload(conversationID, message, replyToID, participantID string, sim *gmproto.SIMPayload) *gmproto.SendMessageRequest {
	tmpID := fmt.Sprintf("tmp_%012d", rand.Int63n(1e12))
	req := &gmproto.SendMessageRequest{
		ConversationID: conversationID,
		MessagePayload: &gmproto.MessagePayload{
			TmpID:                 tmpID,
			MessagePayloadContent: nil,
			MessageInfo: []*gmproto.MessageInfo{{
				Data: &gmproto.MessageInfo_MessageContent{MessageContent: &gmproto.MessageContent{
					Content: message,
				}},
			}},
			ConversationID: conversationID,
			ParticipantID:  participantID,
			TmpID2:         tmpID,
		},
		SIMPayload: sim,
		TmpID:      tmpID,
	}
	if replyToID != "" {
		req.Reply = &gmproto.ReplyPayload{
			MessageID: replyToID,
		}
	}
	return req
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
