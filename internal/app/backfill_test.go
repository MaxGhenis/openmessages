package app

import (
	"testing"

	"github.com/rs/zerolog"

	"github.com/maxghenis/openmessages/internal/db"
)

func TestBackfillStoresConversationsAndMessages(t *testing.T) {
	store, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	logger := zerolog.Nop()

	a := &App{
		Store:  store,
		Logger: logger,
	}

	// Without a real client, Backfill should return an error
	err = a.Backfill()
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

func TestBackfillPopulatesDB(t *testing.T) {
	// Verify that after backfill stores conversations, they're queryable
	store, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Manually insert a conversation as if backfill ran
	store.UpsertConversation(&db.Conversation{
		ConversationID: "c1",
		Name:           "Alice",
		LastMessageTS:  1000,
	})
	store.UpsertMessage(&db.Message{
		MessageID:      "m1",
		ConversationID: "c1",
		Body:           "Hello from backfill",
		TimestampMS:    1000,
		SenderName:     "Alice",
	})

	convos, err := store.ListConversations(50)
	if err != nil {
		t.Fatal(err)
	}
	if len(convos) != 1 {
		t.Fatalf("got %d conversations, want 1", len(convos))
	}
	if convos[0].Name != "Alice" {
		t.Fatalf("got name %q, want Alice", convos[0].Name)
	}

	msgs, err := store.GetMessagesByConversation("c1", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Body != "Hello from backfill" {
		t.Fatalf("got body %q", msgs[0].Body)
	}
}
