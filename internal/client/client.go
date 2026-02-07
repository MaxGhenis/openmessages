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
