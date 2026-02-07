package db

func (s *Store) UpsertConversation(c *Conversation) error {
	_, err := s.db.Exec(`
		INSERT INTO conversations (conversation_id, name, is_group, participants, last_message_ts, unread_count)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(conversation_id) DO UPDATE SET
			name=excluded.name,
			is_group=excluded.is_group,
			participants=excluded.participants,
			last_message_ts=excluded.last_message_ts,
			unread_count=excluded.unread_count
	`, c.ConversationID, c.Name, c.IsGroup, c.Participants, c.LastMessageTS, c.UnreadCount)
	return err
}

func (s *Store) GetConversation(id string) (*Conversation, error) {
	c := &Conversation{}
	err := s.db.QueryRow(`
		SELECT conversation_id, name, is_group, participants, last_message_ts, unread_count
		FROM conversations WHERE conversation_id = ?
	`, id).Scan(&c.ConversationID, &c.Name, &c.IsGroup, &c.Participants, &c.LastMessageTS, &c.UnreadCount)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) ListConversations(limit int) ([]*Conversation, error) {
	rows, err := s.db.Query(`
		SELECT conversation_id, name, is_group, participants, last_message_ts, unread_count
		FROM conversations
		ORDER BY last_message_ts DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []*Conversation
	for rows.Next() {
		c := &Conversation{}
		if err := rows.Scan(&c.ConversationID, &c.Name, &c.IsGroup, &c.Participants, &c.LastMessageTS, &c.UnreadCount); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}
