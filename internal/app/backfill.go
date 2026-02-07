package app

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"

	"github.com/maxghenis/openmessages/internal/client"
	"github.com/maxghenis/openmessages/internal/db"
)

// Backfill fetches existing conversations and recent messages from
// Google Messages and stores them in the local database.
func (a *App) Backfill() error {
	if a.Client == nil {
		return fmt.Errorf("client not connected")
	}

	a.Logger.Info().Msg("Starting backfill of conversations and messages")

	resp, err := a.Client.GM.ListConversations(100, gmproto.ListConversationsRequest_INBOX)
	if err != nil {
		return fmt.Errorf("list conversations: %w", err)
	}

	convos := resp.GetConversations()
	a.Logger.Info().Int("count", len(convos)).Msg("Fetched conversations")

	for _, conv := range convos {
		if err := a.storeConversation(conv); err != nil {
			a.Logger.Error().Err(err).Str("conv_id", conv.GetConversationID()).Msg("Failed to store conversation")
			continue
		}

		// Fetch recent messages for each conversation
		msgResp, err := a.Client.GM.FetchMessages(conv.GetConversationID(), 20, nil)
		if err != nil {
			a.Logger.Warn().Err(err).Str("conv_id", conv.GetConversationID()).Msg("Failed to fetch messages")
			continue
		}

		for _, msg := range msgResp.GetMessages() {
			a.storeMessage(msg)
		}
	}

	a.Logger.Info().Int("conversations", len(convos)).Msg("Backfill complete")
	return nil
}

func (a *App) storeConversation(conv *gmproto.Conversation) error {
	participantsJSON := "[]"
	if ps := conv.GetParticipants(); len(ps) > 0 {
		type pInfo struct {
			Name   string `json:"name"`
			Number string `json:"number"`
			IsMe   bool   `json:"is_me,omitempty"`
		}
		var infos []pInfo
		for _, p := range ps {
			info := pInfo{
				Name: p.GetFullName(),
				IsMe: p.GetIsMe(),
			}
			if id := p.GetID(); id != nil {
				info.Number = id.GetNumber()
			}
			if info.Number == "" {
				info.Number = p.GetFormattedNumber()
			}
			infos = append(infos, info)
		}
		if b, err := json.Marshal(infos); err == nil {
			participantsJSON = string(b)
		}
	}

	unread := 0
	if conv.GetUnread() {
		unread = 1
	}

	return a.Store.UpsertConversation(&db.Conversation{
		ConversationID: conv.GetConversationID(),
		Name:           conv.GetName(),
		IsGroup:        conv.GetIsGroupChat(),
		Participants:   participantsJSON,
		LastMessageTS:  conv.GetLastMessageTimestamp() / 1000,
		UnreadCount:    unread,
	})
}

func (a *App) storeMessage(msg *gmproto.Message) {
	body := client.ExtractMessageBody(msg)
	senderName, senderNumber := client.ExtractSenderInfo(msg)

	status := "unknown"
	if ms := msg.GetMessageStatus(); ms != nil {
		status = ms.GetStatus().String()
	}

	dbMsg := &db.Message{
		MessageID:      msg.GetMessageID(),
		ConversationID: msg.GetConversationID(),
		SenderName:     senderName,
		SenderNumber:   senderNumber,
		Body:           body,
		TimestampMS:    msg.GetTimestamp() / 1000,
		Status:         status,
		IsFromMe:       msg.GetSenderParticipant() != nil && msg.GetSenderParticipant().GetIsMe(),
	}

	if media := client.ExtractMediaInfo(msg); media != nil {
		dbMsg.MediaID = media.MediaID
		dbMsg.MimeType = media.MimeType
		dbMsg.DecryptionKey = hex.EncodeToString(media.DecryptionKey)
	}

	if reactions := client.ExtractReactions(msg); reactions != nil {
		if b, err := json.Marshal(reactions); err == nil {
			dbMsg.Reactions = string(b)
		}
	}
	dbMsg.ReplyToID = client.ExtractReplyToID(msg)

	if err := a.Store.UpsertMessage(dbMsg); err != nil {
		a.Logger.Error().Err(err).Str("msg_id", dbMsg.MessageID).Msg("Failed to store backfill message")
	}
}
