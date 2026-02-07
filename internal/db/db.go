package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Conversation struct {
	ConversationID string
	Name           string
	IsGroup        bool
	Participants   string // JSON array
	LastMessageTS  int64
	UnreadCount    int
}

type Message struct {
	MessageID      string
	ConversationID string
	SenderName     string
	SenderNumber   string
	Body           string
	TimestampMS    int64
	Status         string
	IsFromMe       bool
}

type Contact struct {
	ContactID string
	Name      string
	Number    string
}

func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		conversation_id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		is_group INTEGER NOT NULL DEFAULT 0,
		participants TEXT NOT NULL DEFAULT '[]',
		last_message_ts INTEGER NOT NULL DEFAULT 0,
		unread_count INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS messages (
		message_id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL DEFAULT '',
		sender_name TEXT NOT NULL DEFAULT '',
		sender_number TEXT NOT NULL DEFAULT '',
		body TEXT NOT NULL DEFAULT '',
		timestamp_ms INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT '',
		is_from_me INTEGER NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_messages_conv_ts ON messages(conversation_id, timestamp_ms);
	CREATE INDEX IF NOT EXISTS idx_messages_ts ON messages(timestamp_ms DESC);

	CREATE TABLE IF NOT EXISTS contacts (
		contact_id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		number TEXT NOT NULL DEFAULT ''
	);
	`
	_, err := s.db.Exec(schema)
	return err
}
