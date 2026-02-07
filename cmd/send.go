package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"

	"github.com/maxghenis/openmessages/internal/app"
)

func RunSend(logger zerolog.Logger, conversationID, message string) error {
	a, err := app.New(logger)
	if err != nil {
		return fmt.Errorf("init app: %w", err)
	}
	defer a.Close()

	if err := a.LoadAndConnect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// Look up conversation to get outgoing participant ID
	conv, err := a.Store.GetConversation(conversationID)
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}
	if conv == nil {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	tmpID := uuid.NewString()
	_, err = a.Client.GM.SendMessage(&gmproto.SendMessageRequest{
		ConversationID: conversationID,
		TmpID:          tmpID,
		MessagePayload: &gmproto.MessagePayload{
			TmpID:          tmpID,
			TmpID2:         tmpID,
			ConversationID: conversationID,
			MessageInfo: []*gmproto.MessageInfo{
				{
					Data: &gmproto.MessageInfo_MessageContent{
						MessageContent: &gmproto.MessageContent{
							Content: message,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	logger.Info().Str("conversation", conversationID).Msg("Message sent")
	return nil
}
