package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadSession(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")

	// Save
	data := &SessionData{
		AuthDataJSON: []byte(`{"session_id":"test-123"}`),
		PushKeysJSON: []byte(`{"url":"https://example.com"}`),
	}
	err := SaveSession(path, data)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Load
	loaded, err := LoadSession(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Compare as parsed JSON (whitespace-insensitive)
	var authMap map[string]any
	if err := json.Unmarshal(loaded.AuthDataJSON, &authMap); err != nil {
		t.Fatalf("parse auth data: %v", err)
	}
	if authMap["session_id"] != "test-123" {
		t.Errorf("auth data session_id mismatch: %v", authMap)
	}

	var pushMap map[string]any
	if err := json.Unmarshal(loaded.PushKeysJSON, &pushMap); err != nil {
		t.Fatalf("parse push keys: %v", err)
	}
	if pushMap["url"] != "https://example.com" {
		t.Errorf("push keys url mismatch: %v", pushMap)
	}
}

func TestLoadSessionNotFound(t *testing.T) {
	_, err := LoadSession("/nonexistent/path/session.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSaveSessionCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "session.json")

	data := &SessionData{
		AuthDataJSON: []byte(`{}`),
	}
	err := SaveSession(path, data)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}
