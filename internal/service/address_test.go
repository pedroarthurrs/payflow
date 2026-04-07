package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ============================================================
// Property-Based Tests
// ============================================================

// TestProperty_AddressCreateAndGetByID_RoundTrip verifies the round trip of address creation.
func TestProperty_AddressCreateAndGetByID_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		customerID := uuid.New()
		addressID := uuid.New()

		postalCode := rapid.StringMatching(`[0-9]{5}-[0-9]{3}`).Draw(t, "postal_code")
		street := rapid.StringMatching(`[A-Za-z ]{5,50}`).Draw(t, "street")
		city := rapid.StringMatching(`[A-Za-z ]{3,30}`).Draw(t, "city")
		state := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "state")
		addrType := rapid.SampledFrom([]string{"billing", "shipping", "home"}).Draw(t, "type")
		country := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "country")
		isMain := rapid.Bool().Draw(t, "is_main")

		hasNumber := rapid.Bool().Draw(t, "has_number")
		var number *string
		if hasNumber {
			n := rapid.StringMatching(`[0-9]{1,5}`).Draw(t, "number")
			number = &n
		}

		hasComplement := rapid.Bool().Draw(t, "has_complement")
		var complement *string
		if hasComplement {
			c := rapid.StringMatching(`[A-Za-z0-9 ]{1,20}`).Draw(t, "complement")
			complement = &c
		}

		hasNeighborhood := rapid.Bool().Draw(t, "has_neighborhood")
		var neighborhood *string
		if hasNeighborhood {
			nb := rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "neighborhood")
			neighborhood = &nb
		}

		var stored *domain.Address

		customers := &mockCustomerRepo{
			getByIDFn: customerExistsMock(),
		}
		addresses := &mockAddressRepo{
			createFn: func(_ context.Context, orgID, custID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
				now := time.Now().Truncate(time.Second)
				stored = &domain.Address{
					ID:             addressID,
					OrganizationID: orgID,
					CustomerID:     custID,
					Type:           *input.Type,
					PostalCode:     input.PostalCode,
					Street:         input.Street,
					Number:         input.Number,
					Complement:     input.Complement,
					Neighborhood:   input.Neighborhood,
					City:           input.City,
					State:          input.State,
					Country:        *input.Country,
					IsMain:         *input.IsMain,
					CreatedAt:      now,
				}
				return stored, nil
			},
			getByIDFn: func(_ context.Context, orgID, custID, addrID uuid.UUID) (*domain.Address, error) {
				if stored != nil && addrID == addressID && custID == customerID && orgID == organizationID {
					return stored, nil
				}
				return nil, domain.ErrAddressNotFound
			},
		}

		svc := NewAddressService(customers, addresses)
		ctx := context.Background()

		input := domain.AddressInput{
			Type:         &addrType,
			PostalCode:   postalCode,
			Street:       street,
			Number:       number,
			Complement:   complement,
			Neighborhood: neighborhood,
			City:         city,
			State:        state,
			Country:      &country,
			IsMain:       &isMain,
		}

		created, err := svc.Create(ctx, organizationID, customerID, input)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := svc.GetByID(ctx, organizationID, customerID, created.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		// Verify UUID v4 format
		idBytes := got.ID
		if (idBytes[6] >> 4) != 4 {
			t.Fatalf("ID version is not 4: got %d", idBytes[6]>>4)
		}
		if (idBytes[8] >> 6) != 2 {
			t.Fatalf("ID variant is not RFC 4122: got %d", idBytes[8]>>6)
		}

		// Verify organization and customer IDs
		if got.OrganizationID != organizationID {
			t.Fatalf("expected OrganizationID %v, got %v", organizationID, got.OrganizationID)
		}
		if got.CustomerID != customerID {
			t.Fatalf("expected CustomerID %v, got %v", customerID, got.CustomerID)
		}

		// Verify all input fields preserved
		if got.Type != addrType {
			t.Fatalf("expected Type %q, got %q", addrType, got.Type)
		}
		if got.PostalCode != postalCode {
			t.Fatalf("expected PostalCode %q, got %q", postalCode, got.PostalCode)
		}
		if got.Street != street {
			t.Fatalf("expected Street %q, got %q", street, got.Street)
		}
		if got.City != city {
			t.Fatalf("expected City %q, got %q", city, got.City)
		}
		if got.State != state {
			t.Fatalf("expected State %q, got %q", state, got.State)
		}
		if got.Country != country {
			t.Fatalf("expected Country %q, got %q", country, got.Country)
		}
		if got.IsMain != isMain {
			t.Fatalf("expected IsMain %v, got %v", isMain, got.IsMain)
		}

		// Verify optional fields
		if number == nil {
			if got.Number != nil {
				t.Fatalf("expected Number nil, got %v", got.Number)
			}
		} else {
			if got.Number == nil || *got.Number != *number {
				t.Fatalf("expected Number %q, got %v", *number, got.Number)
			}
		}
		if complement == nil {
			if got.Complement != nil {
				t.Fatalf("expected Complement nil, got %v", got.Complement)
			}
		} else {
			if got.Complement == nil || *got.Complement != *complement {
				t.Fatalf("expected Complement %q, got %v", *complement, got.Complement)
			}
		}
		if neighborhood == nil {
			if got.Neighborhood != nil {
				t.Fatalf("expected Neighborhood nil, got %v", got.Neighborhood)
			}
		} else {
			if got.Neighborhood == nil || *got.Neighborhood != *neighborhood {
				t.Fatalf("expected Neighborhood %q, got %v", *neighborhood, got.Neighborhood)
			}
		}
	})
}

// TestProperty_AddressDefaultValues verifies that default values are applied when fields are omitted.
func TestProperty_AddressDefaultValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		customerID := uuid.New()
		addressID := uuid.New()

		postalCode := rapid.StringMatching(`[0-9]{5}-[0-9]{3}`).Draw(t, "postal_code")
		street := rapid.StringMatching(`[A-Za-z ]{5,50}`).Draw(t, "street")
		city := rapid.StringMatching(`[A-Za-z ]{3,30}`).Draw(t, "city")
		state := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "state")

		customers := &mockCustomerRepo{
			getByIDFn: customerExistsMock(),
		}
		addresses := &mockAddressRepo{
			createFn: func(_ context.Context, orgID, custID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
				return &domain.Address{
					ID:             addressID,
					OrganizationID: orgID,
					CustomerID:     custID,
					Type:           *input.Type,
					PostalCode:     input.PostalCode,
					Street:         input.Street,
					City:           input.City,
					State:          input.State,
					Country:        *input.Country,
					IsMain:         *input.IsMain,
					CreatedAt:      time.Now().Truncate(time.Second),
				}, nil
			},
		}

		svc := NewAddressService(customers, addresses)
		ctx := context.Background()

		input := domain.AddressInput{
			PostalCode: postalCode,
			Street:     street,
			City:       city,
			State:      state,
		}

		created, err := svc.Create(ctx, organizationID, customerID, input)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if created.Type != "billing" {
			t.Fatalf("expected default Type 'billing', got %q", created.Type)
		}
		if created.Country != "BR" {
			t.Fatalf("expected default Country 'BR', got %q", created.Country)
		}
		if created.IsMain != true {
			t.Fatalf("expected default IsMain true, got %v", created.IsMain)
		}
	})
}

// ============================================================
// Unit Tests
// ============================================================

func TestAddressCreate_AllFields(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	addrType := "shipping"
	country := "US"
	isMain := false
	number := "42"
	complement := "Apt 3"
	neighborhood := "Downtown"

	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		createFn: func(_ context.Context, orgID, custID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
			return &domain.Address{
				ID:             addressID,
				OrganizationID: orgID,
				CustomerID:     custID,
				Type:           *input.Type,
				PostalCode:     input.PostalCode,
				Street:         input.Street,
				Number:         input.Number,
				Complement:     input.Complement,
				Neighborhood:   input.Neighborhood,
				City:           input.City,
				State:          input.State,
				Country:        *input.Country,
				IsMain:         *input.IsMain,
				CreatedAt:      time.Now().Truncate(time.Second),
			}, nil
		},
	}

	svc := NewAddressService(customers, addresses)
	input := domain.AddressInput{
		Type:         &addrType,
		PostalCode:   "12345-678",
		Street:       "Main St",
		Number:       &number,
		Complement:   &complement,
		Neighborhood: &neighborhood,
		City:         "Springfield",
		State:        "IL",
		Country:      &country,
		IsMain:       &isMain,
	}

	addr, err := svc.Create(context.Background(), organizationID, customerID, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if addr.ID != addressID {
		t.Fatalf("expected ID %v, got %v", addressID, addr.ID)
	}
	if addr.Type != "shipping" {
		t.Fatalf("expected Type shipping, got %s", addr.Type)
	}
	if addr.Country != "US" {
		t.Fatalf("expected Country US, got %s", addr.Country)
	}
	if addr.IsMain != false {
		t.Fatalf("expected IsMain false, got %v", addr.IsMain)
	}
}

func TestAddressList_Empty(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()

	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		listFn: func(_ context.Context, _, _ uuid.UUID, _ domain.ListParams) ([]domain.Address, error) {
			return []domain.Address{}, nil
		},
	}

	svc := NewAddressService(customers, addresses)
	list, err := svc.List(context.Background(), organizationID, customerID, domain.ListParams{Limit: 20})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d items", len(list))
	}
}

func TestAddressList_Multiple(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	now := time.Now().Truncate(time.Second)

	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		listFn: func(_ context.Context, _, _ uuid.UUID, _ domain.ListParams) ([]domain.Address, error) {
			return []domain.Address{
				{ID: uuid.New(), OrganizationID: organizationID, CustomerID: customerID,
					Type: "billing", PostalCode: "11111", Street: "St A", City: "City A", State: "AA",
					Country: "BR", IsMain: true, CreatedAt: now},
				{ID: uuid.New(), OrganizationID: organizationID, CustomerID: customerID,
					Type: "shipping", PostalCode: "22222", Street: "St B", City: "City B", State: "BB",
					Country: "US", IsMain: false, CreatedAt: now.Add(-time.Hour)},
			}, nil
		},
	}

	svc := NewAddressService(customers, addresses)
	list, err := svc.List(context.Background(), organizationID, customerID, domain.ListParams{Limit: 20})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(list))
	}
}

func TestAddressGetByID_Found(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	now := time.Now().Truncate(time.Second)

	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*domain.Address, error) {
			return &domain.Address{
				ID: addressID, OrganizationID: organizationID, CustomerID: customerID,
				Type: "billing", PostalCode: "12345", Street: "Main St", City: "City", State: "ST",
				Country: "BR", IsMain: true, CreatedAt: now,
			}, nil
		},
	}

	svc := NewAddressService(customers, addresses)
	addr, err := svc.GetByID(context.Background(), organizationID, customerID, addressID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if addr.ID != addressID {
		t.Fatalf("expected ID %v, got %v", addressID, addr.ID)
	}
}

func TestAddressGetByID_NotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		getByIDFn: func(_ context.Context, _, _, _ uuid.UUID) (*domain.Address, error) {
			return nil, domain.ErrAddressNotFound
		},
	}

	svc := NewAddressService(customers, addresses)
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrAddressNotFound) {
		t.Fatalf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestAddressUpdate_Found(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	now := time.Now().Truncate(time.Second)
	addrType := "home"
	country := "BR"
	isMain := true

	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		updateFn: func(_ context.Context, orgID, custID, addrID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
			return &domain.Address{
				ID: addrID, OrganizationID: orgID, CustomerID: custID,
				Type:         *input.Type,
				PostalCode:   input.PostalCode,
				Street:       input.Street,
				City:         input.City,
				State:        input.State,
				Country:      *input.Country,
				IsMain:       *input.IsMain,
				Number:       input.Number,
				Complement:   input.Complement,
				Neighborhood: input.Neighborhood,
				CreatedAt:    now,
			}, nil
		},
	}

	svc := NewAddressService(customers, addresses)
	addr, err := svc.Update(context.Background(), organizationID, customerID, addressID, domain.AddressInput{
		Type:       &addrType,
		PostalCode: "99999",
		Street:     "Updated St",
		City:       "New City",
		State:      "NC",
		Country:    &country,
		IsMain:     &isMain,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if addr.Street != "Updated St" {
		t.Fatalf("expected Street 'Updated St', got %q", addr.Street)
	}
	if addr.Type != "home" {
		t.Fatalf("expected Type 'home', got %q", addr.Type)
	}
}

func TestAddressUpdate_NotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		updateFn: func(_ context.Context, _, _, _ uuid.UUID, _ domain.AddressInput) (*domain.Address, error) {
			return nil, domain.ErrAddressNotFound
		},
	}

	svc := NewAddressService(customers, addresses)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), uuid.New(), domain.AddressInput{
		PostalCode: "12345", Street: "St", City: "C", State: "ST",
	})
	if !errors.Is(err, domain.ErrAddressNotFound) {
		t.Fatalf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestAddressDelete_Found(t *testing.T) {
	deleteCalled := false
	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		deleteFn: func(_ context.Context, _, _, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}

	svc := NewAddressService(customers, addresses)
	err := svc.Delete(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected Delete to be called")
	}
}

func TestAddressDelete_NotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	addresses := &mockAddressRepo{
		deleteFn: func(_ context.Context, _, _, _ uuid.UUID) error {
			return domain.ErrAddressNotFound
		},
	}

	svc := NewAddressService(customers, addresses)
	err := svc.Delete(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrAddressNotFound) {
		t.Fatalf("expected ErrAddressNotFound, got %v", err)
	}
}

// --- Customer existence verification tests ---

func TestAddressCreate_CustomerNotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
			return nil, domain.ErrCustomerNotFound
		},
	}

	svc := NewAddressService(customers, &mockAddressRepo{})
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), domain.AddressInput{
		PostalCode: "12345", Street: "St", City: "C", State: "ST",
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressList_CustomerNotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
			return nil, domain.ErrCustomerNotFound
		},
	}

	svc := NewAddressService(customers, &mockAddressRepo{})
	_, err := svc.List(context.Background(), uuid.New(), uuid.New(), domain.ListParams{Limit: 20})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressGetByID_CustomerNotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
			return nil, domain.ErrCustomerNotFound
		},
	}

	svc := NewAddressService(customers, &mockAddressRepo{})
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressUpdate_CustomerNotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
			return nil, domain.ErrCustomerNotFound
		},
	}

	svc := NewAddressService(customers, &mockAddressRepo{})
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), uuid.New(), domain.AddressInput{
		PostalCode: "12345", Street: "St", City: "C", State: "ST",
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressDelete_CustomerNotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
			return nil, domain.ErrCustomerNotFound
		},
	}

	svc := NewAddressService(customers, &mockAddressRepo{})
	err := svc.Delete(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}
