package repository

import (
	"database/sql"
	"encoding/json"

	addressdb "i9pay/internal/database/dbgen/address"

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
