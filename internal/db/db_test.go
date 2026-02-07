package db

import (
	"testing"
	"time"
)

func TestNewDB(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()
}

func TestConversationCRUD(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	conv := &Conversation{
		ConversationID: "conv-1",
		Name:           "Alice",
		IsGroup:        false,
		Participants:   `[{"name":"Alice","number":"+15551234567"}]`,
		LastMessageTS:  time.Now().UnixMilli(),
		UnreadCount:    2,
	}

	// Upsert
	err = store.UpsertConversation(conv)
	if err != nil {
		t.Fatalf("upsert conversation: %v", err)
	}

	// Get
	got, err := store.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("get conversation: %v", err)
	}
	if got.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", got.Name)
	}
	if got.UnreadCount != 2 {
		t.Errorf("expected unread 2, got %d", got.UnreadCount)
	}

	// Update
	conv.Name = "Alice Smith"
	conv.UnreadCount = 0
	err = store.UpsertConversation(conv)
	if err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	got, err = store.GetConversation("conv-1")
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Name != "Alice Smith" {
		t.Errorf("expected name Alice Smith, got %s", got.Name)
	}

	// List
	convs, err := store.ListConversations(10)
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(convs) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(convs))
	}
}

func TestListConversationsOrdering(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	now := time.Now().UnixMilli()
	for i, name := range []string{"Old", "New", "Middle"} {
		ts := now + int64((i-1)*1000) // Old=-1s, New=0s, Middle=+1s
		err := store.UpsertConversation(&Conversation{
			ConversationID: name,
			Name:           name,
			LastMessageTS:  ts,
		})
		if err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	convs, err := store.ListConversations(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(convs) != 3 {
		t.Fatalf("expected 3, got %d", len(convs))
	}
	// Should be ordered by last_message_ts DESC
	if convs[0].Name != "Middle" {
		t.Errorf("expected first=Middle, got %s", convs[0].Name)
	}
	if convs[2].Name != "Old" {
		t.Errorf("expected last=Old, got %s", convs[2].Name)
	}
}

func TestMessageCRUD(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	now := time.Now().UnixMilli()

	msg := &Message{
		MessageID:      "msg-1",
		ConversationID: "conv-1",
		SenderName:     "Alice",
		SenderNumber:   "+15551234567",
		Body:           "Hello world",
		TimestampMS:    now,
		Status:         "delivered",
		IsFromMe:       false,
	}

	err = store.UpsertMessage(msg)
	if err != nil {
		t.Fatalf("upsert message: %v", err)
	}

	// Get by conversation
	msgs, err := store.GetMessagesByConversation("conv-1", 10)
	if err != nil {
		t.Fatalf("get by conversation: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Body != "Hello world" {
		t.Errorf("expected body 'Hello world', got %s", msgs[0].Body)
	}

	// Get recent with filters
	msgs, err = store.GetMessages("+15551234567", now-1000, now+1000, 10)
	if err != nil {
		t.Fatalf("get messages filtered: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1, got %d", len(msgs))
	}

	// Filter by wrong number
	msgs, err = store.GetMessages("+15559999999", 0, 0, 10)
	if err != nil {
		t.Fatalf("get messages wrong number: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0, got %d", len(msgs))
	}
}

func TestSearchMessages(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	now := time.Now().UnixMilli()
	messages := []Message{
		{MessageID: "1", ConversationID: "c1", Body: "Hello world", TimestampMS: now},
		{MessageID: "2", ConversationID: "c1", Body: "Goodbye world", TimestampMS: now + 1},
		{MessageID: "3", ConversationID: "c2", Body: "Hello there", TimestampMS: now + 2},
		{MessageID: "4", ConversationID: "c2", Body: "Something else", TimestampMS: now + 3},
	}
	for i := range messages {
		if err := store.UpsertMessage(&messages[i]); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	// Search for "hello"
	results, err := store.SearchMessages("hello", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'hello', got %d", len(results))
	}

	// Search for "goodbye"
	results, err = store.SearchMessages("goodbye", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'goodbye', got %d", len(results))
	}

	// Search with no results
	results, err = store.SearchMessages("nonexistent", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0, got %d", len(results))
	}
}

func TestContactCRUD(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	contact := &Contact{
		ContactID: "contact-1",
		Name:      "Alice",
		Number:    "+15551234567",
	}

	err = store.UpsertContact(contact)
	if err != nil {
		t.Fatalf("upsert contact: %v", err)
	}

	// List all
	contacts, err := store.ListContacts("", 10)
	if err != nil {
		t.Fatalf("list contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(contacts))
	}
	if contacts[0].Name != "Alice" {
		t.Errorf("expected name Alice, got %s", contacts[0].Name)
	}

	// Search by name
	contacts, err = store.ListContacts("ali", 10)
	if err != nil {
		t.Fatalf("search contacts: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("expected 1, got %d", len(contacts))
	}

	// Search by number
	contacts, err = store.ListContacts("555123", 10)
	if err != nil {
		t.Fatalf("search by number: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("expected 1, got %d", len(contacts))
	}

	// No match
	contacts, err = store.ListContacts("bob", 10)
	if err != nil {
		t.Fatalf("search no match: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("expected 0, got %d", len(contacts))
	}
}

func TestGetMessagesNoFilters(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer store.Close()

	now := time.Now().UnixMilli()
	for i := 0; i < 5; i++ {
		err := store.UpsertMessage(&Message{
			MessageID:      "msg-" + string(rune('a'+i)),
			ConversationID: "c1",
			Body:           "Message",
			TimestampMS:    now + int64(i*1000),
		})
		if err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	// No filters, limit 3
	msgs, err := store.GetMessages("", 0, 0, 3)
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("expected 3, got %d", len(msgs))
	}
}
