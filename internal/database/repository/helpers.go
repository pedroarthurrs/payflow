package repository

import (
	"database/sql"
	"encoding/json"

	"i9pay/internal/database/dbgen"

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

// toNullAddressType converts a *string to dbgen.NullAddressType.
func toNullAddressType(s *string) dbgen.NullAddressType {
	if s == nil {
		return dbgen.NullAddressType{}
	}
	return dbgen.NullAddressType{AddressType: dbgen.AddressType(*s), Valid: true}
}
