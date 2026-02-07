package client

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"
)

type Client struct {
	GM     *libgm.Client
	Logger zerolog.Logger
}

func NewFromSession(sessionData *SessionData, logger zerolog.Logger) (*Client, error) {
	authData := libgm.NewAuthData()
	if err := json.Unmarshal(sessionData.AuthDataJSON, authData); err != nil {
		return nil, fmt.Errorf("unmarshal auth data: %w", err)
	}

	var pushKeys *libgm.PushKeys
	if len(sessionData.PushKeysJSON) > 0 {
		pushKeys = &libgm.PushKeys{}
		if err := json.Unmarshal(sessionData.PushKeysJSON, pushKeys); err != nil {
			return nil, fmt.Errorf("unmarshal push keys: %w", err)
		}
	}

	cli := libgm.NewClient(authData, pushKeys, logger)
	return &Client{GM: cli, Logger: logger}, nil
}

func NewForPairing(logger zerolog.Logger) *Client {
	authData := libgm.NewAuthData()
	cli := libgm.NewClient(authData, nil, logger)
	return &Client{GM: cli, Logger: logger}
}

func (c *Client) SessionData() (*SessionData, error) {
	authJSON, err := json.Marshal(c.GM.AuthData)
	if err != nil {
		return nil, fmt.Errorf("marshal auth data: %w", err)
	}
	var pushJSON json.RawMessage
	if c.GM.PushKeys != nil {
		pushJSON, err = json.Marshal(c.GM.PushKeys)
		if err != nil {
			return nil, fmt.Errorf("marshal push keys: %w", err)
		}
	}
	return &SessionData{
		AuthDataJSON: authJSON,
		PushKeysJSON: pushJSON,
	}, nil
}

// ExtractMessageBody extracts text content from a protobuf Message.
func ExtractMessageBody(msg *gmproto.Message) string {
	for _, info := range msg.GetMessageInfo() {
		if mc := info.GetMessageContent(); mc != nil {
			return mc.GetContent()
		}
	}
	return ""
}

// MediaInfo holds extracted media metadata from a protobuf Message.
type MediaInfo struct {
	MediaID       string
	MimeType      string
	MediaName     string
	DecryptionKey []byte
	Size          int64
}

// ExtractMediaInfo extracts media content from a protobuf Message.
// Returns nil if the message has no media attachment.
func ExtractMediaInfo(msg *gmproto.Message) *MediaInfo {
	for _, info := range msg.GetMessageInfo() {
		if mc := info.GetMediaContent(); mc != nil {
			mime := mc.GetMimeType()
			if mime == "" {
				// Derive from format enum
				switch {
				case mc.GetFormat() >= 1 && mc.GetFormat() <= 7:
					mime = "image/jpeg"
				default:
					mime = "application/octet-stream"
				}
			}
			return &MediaInfo{
				MediaID:       mc.GetMediaID(),
				MimeType:      mime,
				MediaName:     mc.GetMediaName(),
				DecryptionKey: mc.GetDecryptionKey(),
				Size:          mc.GetSize(),
			}
		}
	}
	return nil
}

// Reaction holds an emoji and how many people reacted with it.
type Reaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

// ExtractReactions extracts reaction data from a protobuf Message.
// Returns nil if there are no reactions.
func ExtractReactions(msg *gmproto.Message) []Reaction {
	entries := msg.GetReactions()
	if len(entries) == 0 {
		return nil
	}
	var reactions []Reaction
	for _, entry := range entries {
		if data := entry.GetData(); data != nil {
			emoji := data.GetUnicode()
			if emoji == "" {
				continue
			}
			reactions = append(reactions, Reaction{
				Emoji: emoji,
				Count: len(entry.GetParticipantIDs()),
			})
		}
	}
	if len(reactions) == 0 {
		return nil
	}
	return reactions
}

// ExtractReplyToID extracts the replied-to message ID, if any.
func ExtractReplyToID(msg *gmproto.Message) string {
	if rm := msg.GetReplyMessage(); rm != nil {
		return rm.GetMessageID()
	}
	return ""
}

// ExtractSenderInfo gets the sender name and number from a Message.
func ExtractSenderInfo(msg *gmproto.Message) (name, number string) {
	if p := msg.GetSenderParticipant(); p != nil {
		name = p.GetFullName()
		if name == "" {
			name = p.GetFirstName()
		}
		if id := p.GetID(); id != nil {
			number = id.GetNumber()
		}
		if number == "" {
			number = p.GetFormattedNumber()
		}
	}
	return
}
