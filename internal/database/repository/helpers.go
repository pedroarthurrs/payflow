package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	addressdb "i9pay/internal/database/dbgen/address"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// toNullString converts a *string to sql.NullString.
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// toNullBool converts a *bool to sql.NullBool.
func toNullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

// toNullRawMessage converts a json.RawMessage to pqtype.NullRawMessage.
func toNullRawMessage(data json.RawMessage) pqtype.NullRawMessage {
	if data == nil {
		return pqtype.NullRawMessage{}
	}
	return pqtype.NullRawMessage{RawMessage: data, Valid: true}
}

// toNullAddressType converts a *string to addressdb.NullAddressType.
func toNullAddressType(s *string) addressdb.NullAddressType {
	if s == nil {
		return addressdb.NullAddressType{}
	}
	return addressdb.NullAddressType{AddressType: addressdb.AddressType(*s), Valid: true}
}

// toNullUUID converts a *uuid.UUID to uuid.NullUUID.
func toNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

// formatDecimal converts a float64 to a string suitable for NUMERIC columns.
func formatDecimal(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

// parseDecimal parses a NUMERIC string returned by SQLC into float64.
func parseDecimal(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// toNullTime converts a *time.Time to sql.NullTime — defined in credential.go
