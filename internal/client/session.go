package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type SessionData struct {
	AuthDataJSON json.RawMessage `json:"auth_data"`
	PushKeysJSON json.RawMessage `json:"push_keys,omitempty"`
}

func SaveSession(path string, data *SessionData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

func LoadSession(path string) (*SessionData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var data SessionData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &data, nil
}
