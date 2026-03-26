package dto

import (
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

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
