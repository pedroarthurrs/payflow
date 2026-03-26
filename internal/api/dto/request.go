package dto

// CreateAddressRequest is the DTO for address creation, used by customer and address handlers.
type CreateAddressRequest struct {
	Type         *string `json:"type"         binding:"omitempty,oneof=billing shipping home"`
	PostalCode   string  `json:"postal_code"  binding:"required,max=10"`
	Street       string  `json:"street"       binding:"required"`
	Number       *string `json:"number"       binding:"omitempty,max=10"`
	Complement   *string `json:"complement"   binding:"omitempty,max=100"`
	Neighborhood *string `json:"neighborhood" binding:"omitempty,max=100"`
	City         string  `json:"city"         binding:"required,max=100"`
	State        string  `json:"state"        binding:"required,len=2"`
	Country      *string `json:"country"      binding:"omitempty,len=2"`
	IsMain       *bool   `json:"is_main"`
}
