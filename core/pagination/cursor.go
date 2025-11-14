package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// CursorData represents the data encoded in a cursor
type CursorData struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"ts"`
	Value     string    `json:"val,omitempty"` // For sorting by fields other than timestamp
}

// EncodeCursor encodes cursor data into a base64 string
func EncodeCursor(id string, timestamp time.Time, value string) (string, error) {
	data := CursorData{
		ID:        id,
		Timestamp: timestamp,
		Value:     value,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor data: %w", err)
	}

	encoded := base64.URLEncoding.EncodeToString(jsonData)
	return encoded, nil
}

// DecodeCursor decodes a base64 cursor string back to CursorData
func DecodeCursor(cursor string) (*CursorData, error) {
	if cursor == "" {
		return nil, nil
	}

	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %w", err)
	}

	var data CursorData
	if err := json.Unmarshal(decoded, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor data: %w", err)
	}

	return &data, nil
}

// SimpleCursorEncode encodes a simple string cursor
func SimpleCursorEncode(value string) string {
	return base64.URLEncoding.EncodeToString([]byte(value))
}

// SimpleCursorDecode decodes a simple string cursor
func SimpleCursorDecode(cursor string) (string, error) {
	if cursor == "" {
		return "", nil
	}

	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("failed to decode cursor: %w", err)
	}

	return string(decoded), nil
}
