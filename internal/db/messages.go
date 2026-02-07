package db

import (
	"fmt"
	"strings"
)

func (s *Store) UpsertMessage(m *Message) error {
	_, err := s.db.Exec(`
		INSERT INTO messages (message_id, conversation_id, sender_name, sender_number, body, timestamp_ms, status, is_from_me)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			conversation_id=excluded.conversation_id,
			sender_name=excluded.sender_name,
			sender_number=excluded.sender_number,
			body=excluded.body,
			timestamp_ms=excluded.timestamp_ms,
			status=excluded.status,
			is_from_me=excluded.is_from_me
	`, m.MessageID, m.ConversationID, m.SenderName, m.SenderNumber, m.Body, m.TimestampMS, m.Status, m.IsFromMe)
	return err
}

func (s *Store) GetMessagesByConversation(conversationID string, limit int) ([]*Message, error) {
	rows, err := s.db.Query(`
		SELECT message_id, conversation_id, sender_name, sender_number, body, timestamp_ms, status, is_from_me
		FROM messages
		WHERE conversation_id = ?
		ORDER BY timestamp_ms DESC
		LIMIT ?
	`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (s *Store) GetMessages(phoneNumber string, afterMS, beforeMS int64, limit int) ([]*Message, error) {
	var conditions []string
	var args []any

	if phoneNumber != "" {
		conditions = append(conditions, "sender_number = ?")
		args = append(args, phoneNumber)
	}
	if afterMS > 0 {
		conditions = append(conditions, "timestamp_ms >= ?")
		args = append(args, afterMS)
	}
	if beforeMS > 0 {
		conditions = append(conditions, "timestamp_ms <= ?")
		args = append(args, beforeMS)
	}

	query := `SELECT message_id, conversation_id, sender_name, sender_number, body, timestamp_ms, status, is_from_me FROM messages`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY timestamp_ms DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (s *Store) SearchMessages(query, phoneNumber string, limit int) ([]*Message, error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "body LIKE ?")
	args = append(args, "%"+query+"%")

	if phoneNumber != "" {
		conditions = append(conditions, "sender_number = ?")
		args = append(args, phoneNumber)
	}

	q := `SELECT message_id, conversation_id, sender_name, sender_number, body, timestamp_ms, status, is_from_me FROM messages`
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY timestamp_ms DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func scanMessages(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]*Message, error) {
	var msgs []*Message
	for rows.Next() {
		m := &Message{}
		if err := rows.Scan(&m.MessageID, &m.ConversationID, &m.SenderName, &m.SenderNumber, &m.Body, &m.TimestampMS, &m.Status, &m.IsFromMe); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
