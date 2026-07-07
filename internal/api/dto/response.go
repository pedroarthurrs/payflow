package dto

import (
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// PagedResponse is a generic paginated list response for offset-based pagination.
type PagedResponse[T any] struct {
	Data   []T   `json:"data"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
	Total  int64 `json:"total"`
}

// CursorPagedResponse is a generic paginated list response for keyset pagination.
type CursorPagedResponse[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor,omitempty"`
	Limit      int64   `json:"limit"`
}

// AddressResponse is the response DTO for an address.
type AddressResponse struct {
	ID           uuid.UUID `json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	Type         string    `json:"type"`
	PostalCode   string    `json:"postal_code"`
	Street       string    `json:"street"`
	Number       *string   `json:"number,omitempty"`
	Complement   *string   `json:"complement,omitempty"`
	Neighborhood *string   `json:"neighborhood,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	Country      string    `json:"country"`
	IsMain       bool      `json:"is_main"`
	CreatedAt    time.Time `json:"created_at"`
}

// ToAddressResponse converts a domain.Address to an AddressResponse.
// Returns nil if the input pointer is nil.
func ToAddressResponse(a *domain.Address) *AddressResponse {
	if a == nil {
		return nil
	}
	return &AddressResponse{
		ID:           a.ID,
		CustomerID:   a.CustomerID,
		Type:         a.Type,
		PostalCode:   a.PostalCode,
		Street:       a.Street,
		Number:       a.Number,
		Complement:   a.Complement,
		Neighborhood: a.Neighborhood,
		City:         a.City,
		State:        a.State,
		Country:      a.Country,
		IsMain:       a.IsMain,
		CreatedAt:    a.CreatedAt,
	}
}
