// pantry/mongo/cursor.go
package mongo

import (
	"encoding/base64"
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Cursor is a lightweight keyset cursor (folded key + stable _id tiebreak).
type Cursor struct {
	CI string             `json:"ci"`
	ID primitive.ObjectID `json:"id"`
}

// EncodeCursor encodes a (ci,id) pair into a URL-safe string.
func EncodeCursor(ci string, id primitive.ObjectID) string {
	b, _ := json.Marshal(Cursor{CI: ci, ID: id})
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor decodes a cursor string; returns false on invalid input.
func DecodeCursor(s string) (Cursor, bool) {
	if s == "" {
		return Cursor{}, false
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, false
	}
	var out Cursor
	if err := json.Unmarshal(b, &out); err != nil {
		return Cursor{}, false
	}
	return out, true
}
