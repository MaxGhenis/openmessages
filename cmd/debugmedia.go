package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/maxghenis/openmessages/internal/app"
	"github.com/maxghenis/openmessages/internal/client"
)

func RunDebugMedia(logger zerolog.Logger, convID string) error {
	a, err := app.New(logger)
	if err != nil {
		return err
	}
	defer a.Close()
	if err := a.LoadAndConnect(); err != nil {
		return err
	}

	resp, err := a.Client.GM.FetchMessages(convID, 10, nil)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	for _, msg := range resp.GetMessages() {
		mi := client.ExtractMediaInfo(msg)
		if mi != nil {
			fmt.Printf("MsgID=%s MediaID=%q MimeType=%s KeyLen=%d\n", msg.GetMessageID(), mi.MediaID, mi.MimeType, len(mi.DecryptionKey))
		}
		for _, info := range msg.GetMessageInfo() {
			if mc := info.GetMediaContent(); mc != nil {
				raw, _ := json.Marshal(mc)
				fmt.Printf("  Raw MediaContent: %s\n", raw)
			}
		}
	}
	return nil
}
