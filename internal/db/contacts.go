package db

func (s *Store) UpsertContact(c *Contact) error {
	_, err := s.db.Exec(`
		INSERT INTO contacts (contact_id, name, number)
		VALUES (?, ?, ?)
		ON CONFLICT(contact_id) DO UPDATE SET
			name=excluded.name,
			number=excluded.number
	`, c.ContactID, c.Name, c.Number)
	return err
}

func (s *Store) ListContacts(query string, limit int) ([]*Contact, error) {
	var rows_query string
	var args []any

	if query != "" {
		rows_query = `
			SELECT contact_id, name, number FROM contacts
			WHERE name LIKE ? OR number LIKE ?
			ORDER BY name
			LIMIT ?
		`
		like := "%" + query + "%"
		args = []any{like, like, limit}
	} else {
		rows_query = `
			SELECT contact_id, name, number FROM contacts
			ORDER BY name
			LIMIT ?
		`
		args = []any{limit}
	}

	rows, err := s.db.Query(rows_query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		c := &Contact{}
		if err := rows.Scan(&c.ContactID, &c.Name, &c.Number); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}
