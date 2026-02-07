package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/maxghenis/openmessages/internal/db"
)

type testServer struct {
	store  *db.Store
	server *httptest.Server
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	store, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	logger := zerolog.Nop()
	h := APIHandler(store, nil, logger)
	srv := httptest.NewServer(h)

	t.Cleanup(func() {
		srv.Close()
		store.Close()
	})

	return &testServer{store: store, server: srv}
}

func TestListConversations(t *testing.T) {
	ts := newTestServer(t)

	// Empty list
	resp, err := http.Get(ts.server.URL + "/api/conversations")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("got content-type %q, want application/json", ct)
	}

	var convos []db.Conversation
	if err := json.NewDecoder(resp.Body).Decode(&convos); err != nil {
		t.Fatal(err)
	}
	if len(convos) != 0 {
		t.Fatalf("got %d conversations, want 0", len(convos))
	}

	// Add some conversations
	ts.store.UpsertConversation(&db.Conversation{
		ConversationID: "c1", Name: "Alice", LastMessageTS: 200,
	})
	ts.store.UpsertConversation(&db.Conversation{
		ConversationID: "c2", Name: "Bob", LastMessageTS: 100,
	})

	resp2, err := http.Get(ts.server.URL + "/api/conversations")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	var convos2 []db.Conversation
	if err := json.NewDecoder(resp2.Body).Decode(&convos2); err != nil {
		t.Fatal(err)
	}
	if len(convos2) != 2 {
		t.Fatalf("got %d conversations, want 2", len(convos2))
	}
	// Should be ordered by last_message_ts DESC
	if convos2[0].Name != "Alice" {
		t.Fatalf("first conversation should be Alice (most recent), got %q", convos2[0].Name)
	}
}

func TestGetMessages(t *testing.T) {
	ts := newTestServer(t)

	ts.store.UpsertConversation(&db.Conversation{
		ConversationID: "c1", Name: "Alice", LastMessageTS: 200,
	})
	ts.store.UpsertMessage(&db.Message{
		MessageID: "m1", ConversationID: "c1", Body: "Hello",
		SenderName: "Alice", TimestampMS: 100,
	})
	ts.store.UpsertMessage(&db.Message{
		MessageID: "m2", ConversationID: "c1", Body: "World",
		SenderName: "Me", TimestampMS: 200, IsFromMe: true,
	})

	resp, err := http.Get(ts.server.URL + "/api/conversations/c1/messages")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var msgs []db.Message
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}
}

func TestGetMessagesWithLimit(t *testing.T) {
	ts := newTestServer(t)

	ts.store.UpsertConversation(&db.Conversation{
		ConversationID: "c1", Name: "Alice", LastMessageTS: 300,
	})
	for i := 0; i < 5; i++ {
		ts.store.UpsertMessage(&db.Message{
			MessageID:      "m" + string(rune('0'+i)),
			ConversationID: "c1",
			Body:           "msg",
			TimestampMS:    int64(i * 100),
		})
	}

	resp, err := http.Get(ts.server.URL + "/api/conversations/c1/messages?limit=2")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var msgs []db.Message
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}
}

func TestSearchMessages(t *testing.T) {
	ts := newTestServer(t)

	ts.store.UpsertMessage(&db.Message{
		MessageID: "m1", ConversationID: "c1", Body: "lunch tomorrow?",
		TimestampMS: 100,
	})
	ts.store.UpsertMessage(&db.Message{
		MessageID: "m2", ConversationID: "c1", Body: "sure!",
		TimestampMS: 200,
	})

	resp, err := http.Get(ts.server.URL + "/api/search?q=lunch")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var msgs []db.Message
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Body != "lunch tomorrow?" {
		t.Fatalf("got body %q, want %q", msgs[0].Body, "lunch tomorrow?")
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.server.URL + "/api/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Fatalf("got status %d, want 400", resp.StatusCode)
	}
}

func TestSendMessage(t *testing.T) {
	ts := newTestServer(t)

	// send_message requires a real libgm client, so we test that
	// it returns 503 when client is nil
	body := `{"phone_number": "+1234567890", "message": "Hello!"}`
	resp, err := http.Post(ts.server.URL+"/api/send", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 503 {
		t.Fatalf("got status %d, want 503 (no client)", resp.StatusCode)
	}
}

func TestSendMessageValidation(t *testing.T) {
	ts := newTestServer(t)

	// Missing message field
	body := `{"phone_number": "+1234567890"}`
	resp, err := http.Post(ts.server.URL+"/api/send", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Fatalf("got status %d, want 400", resp.StatusCode)
	}
}

func TestGetStatus(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.server.URL + "/api/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var status map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["connected"] != false {
		t.Fatal("expected connected=false when no client")
	}
}

func TestStaticFileServing(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("got status %d, want 200 for index", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("got content-type %q, want text/html", ct)
	}
}
