// toolkit/db/mongodb/errors.go
package mongodb

import (
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// IsDup reports whether err is a Mongo duplicate-key error (E11000).
// It handles WriteException, BulkWriteException, CommandError, and
// falls back to a string contains check for maximum robustness.
func IsDup(err error) bool {
	if err == nil {
		return false
	}

	// Bulk write
	var bwe mongo.BulkWriteException
	if errors.As(err, &bwe) {
		for _, we := range bwe.WriteErrors {
			if we.Code == 11000 {
				return true
			}
		}
		if bwe.WriteConcernError != nil && bwe.WriteConcernError.Code == 11000 {
			return true
		}
	}

	// Regular write
	var we mongo.WriteException
	if errors.As(err, &we) {
		for _, e := range we.WriteErrors {
			if e.Code == 11000 {
				return true
			}
		}
		if we.WriteConcernError != nil && we.WriteConcernError.Code == 11000 {
			return true
		}
	}

	// Command error
	var ce mongo.CommandError
	if errors.As(err, &ce) && ce.Code == 11000 {
		return true
	}

	// Belt & suspenders: some drivers/hosts surface "E11000" as text.
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "e11000") || strings.Contains(s, "duplicate key")
}
