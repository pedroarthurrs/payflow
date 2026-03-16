package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// addressMockQuerier implements dbgen.Querier for address service testing.
type addressMockQuerier struct {
	createAddressFn           func(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error)
	deleteAddressFn           func(ctx context.Context, arg dbgen.DeleteAddressParams) error
	getAddressByIDFn          func(ctx context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error)
	listAddressesByCustomerFn func(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error)
	updateAddressFn           func(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error)
	getCustomerByIDFn         func(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error)
}

func (m *addressMockQuerier) CreateAddress(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
	if m.createAddressFn != nil {
		return m.createAddressFn(ctx, arg)
	}
	return dbgen.Address{}, nil
}

func (m *addressMockQuerier) DeleteAddress(ctx context.Context, arg dbgen.DeleteAddressParams) error {
	if m.deleteAddressFn != nil {
		return m.deleteAddressFn(ctx, arg)
	}
	return nil
}

func (m *addressMockQuerier) GetAddressByID(ctx context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
	if m.getAddressByIDFn != nil {
		return m.getAddressByIDFn(ctx, arg)
	}
	return dbgen.Address{}, sql.ErrNoRows
}

func (m *addressMockQuerier) ListAddressesByCustomer(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
	if m.listAddressesByCustomerFn != nil {
		return m.listAddressesByCustomerFn(ctx, arg)
	}
	return nil, nil
}

func (m *addressMockQuerier) UpdateAddress(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
	if m.updateAddressFn != nil {
		return m.updateAddressFn(ctx, arg)
	}
	return dbgen.Address{}, sql.ErrNoRows
}

func (m *addressMockQuerier) GetCustomerByID(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
	if m.getCustomerByIDFn != nil {
		return m.getCustomerByIDFn(ctx, arg)
	}
	return dbgen.Customer{}, nil
}

// Stub methods to satisfy dbgen.Querier interface
func (m *addressMockQuerier) CreateCustomer(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}
func (m *addressMockQuerier) GetTenant(ctx context.Context, id uuid.UUID) (dbgen.Tenant, error) {
	return dbgen.Tenant{}, nil
}
func (m *addressMockQuerier) ListCustomersByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbgen.Customer, error) {
	return nil, nil
}
func (m *addressMockQuerier) ListTenants(ctx context.Context) ([]dbgen.Tenant, error) {
	return nil, nil
}
func (m *addressMockQuerier) SoftDeleteCustomer(ctx context.Context, arg dbgen.SoftDeleteCustomerParams) error {
	return nil
}
func (m *addressMockQuerier) UpdateCustomer(ctx context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
	return dbgen.Customer{}, nil
}
func (m *addressMockQuerier) HardDeleteCustomer(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error {
	return nil
}

// newAddressTestService creates an AddressService backed by the given mock.
func newAddressTestService(mock *addressMockQuerier) *AddressService {
	return NewAddressService(&database.DB{Queries: mock})
}

// customerExistsMock returns a getCustomerByIDFn that always finds the customer.
func customerExistsMock() func(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
	return func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
		return dbgen.Customer{ID: arg.ID, TenantID: arg.TenantID}, nil
	}
}

// --- Helper to build a dbgen.Address from params ---

func buildDBAddress(id, tenantID, customerID uuid.UUID, arg dbgen.CreateAddressParams) dbgen.Address {
	now := time.Now().Truncate(time.Second)
	return dbgen.Address{
		ID:           id,
		TenantID:     tenantID,
		CustomerID:   customerID,
		Type:         arg.Type,
		PostalCode:   arg.PostalCode,
		Street:       arg.Street,
		Number:       arg.Number,
		Complement:   arg.Complement,
		Neighborhood: arg.Neighborhood,
		City:         arg.City,
		State:        arg.State,
		Country:      arg.Country,
		IsMain:       arg.IsMain,
		CreatedAt:    sql.NullTime{Time: now, Valid: true},
	}
}

// ============================================================
// Property-Based Tests
// ============================================================

// Feature: customer-addresses, Property 1: Round trip de criação de endereço
// **Validates: Requirements 1.1, 4.1, 1.12**
func TestProperty_AddressCreateAndGetByID_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
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

		var stored dbgen.Address

		mock := &addressMockQuerier{
			getCustomerByIDFn: customerExistsMock(),
			createAddressFn: func(_ context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
				stored = buildDBAddress(addressID, tenantID, customerID, arg)
				return stored, nil
			},
			getAddressByIDFn: func(_ context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
				if arg.ID == addressID && arg.CustomerID == customerID && arg.TenantID == tenantID {
					return stored, nil
				}
				return dbgen.Address{}, sql.ErrNoRows
			},
		}

		svc := newAddressTestService(mock)
		ctx := context.Background()

		input := domain.CreateAddressInput{
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

		created, err := svc.Create(ctx, tenantID, customerID, input)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := svc.GetByID(ctx, tenantID, customerID, created.ID)
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

		// Verify tenant and customer IDs
		if got.TenantID != tenantID {
			t.Fatalf("expected TenantID %v, got %v", tenantID, got.TenantID)
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

// Feature: customer-addresses, Property 4: Valores padrão aplicados quando campos omitidos
// **Validates: Requirements 1.6, 1.7, 1.9**
func TestProperty_AddressDefaultValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		customerID := uuid.New()
		addressID := uuid.New()

		postalCode := rapid.StringMatching(`[0-9]{5}-[0-9]{3}`).Draw(t, "postal_code")
		street := rapid.StringMatching(`[A-Za-z ]{5,50}`).Draw(t, "street")
		city := rapid.StringMatching(`[A-Za-z ]{3,30}`).Draw(t, "city")
		state := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "state")

		mock := &addressMockQuerier{
			getCustomerByIDFn: customerExistsMock(),
			createAddressFn: func(_ context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
				return buildDBAddress(addressID, tenantID, customerID, arg), nil
			},
		}

		svc := newAddressTestService(mock)
		ctx := context.Background()

		// Input WITHOUT type, country, and is_main
		input := domain.CreateAddressInput{
			PostalCode: postalCode,
			Street:     street,
			City:       city,
			State:      state,
		}

		created, err := svc.Create(ctx, tenantID, customerID, input)
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
// Unit Tests (Sub-task 2.5)
// ============================================================

func TestAddressCreate_AllFields(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	addrType := "shipping"
	country := "US"
	isMain := false
	number := "42"
	complement := "Apt 3"
	neighborhood := "Downtown"

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		createAddressFn: func(_ context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
			return buildDBAddress(addressID, tenantID, customerID, arg), nil
		},
	}

	svc := newAddressTestService(mock)
	input := domain.CreateAddressInput{
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

	addr, err := svc.Create(context.Background(), tenantID, customerID, input)
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
	tenantID := uuid.New()
	customerID := uuid.New()

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		listAddressesByCustomerFn: func(_ context.Context, _ dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
			return []dbgen.Address{}, nil
		},
	}

	svc := newAddressTestService(mock)
	list, err := svc.List(context.Background(), tenantID, customerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d items", len(list))
	}
}

func TestAddressList_Multiple(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		listAddressesByCustomerFn: func(_ context.Context, _ dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
			return []dbgen.Address{
				{ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
					Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
					PostalCode: "11111", Street: "St A", City: "City A", State: "AA",
					Country:   sql.NullString{String: "BR", Valid: true},
					IsMain:    sql.NullBool{Bool: true, Valid: true},
					CreatedAt: sql.NullTime{Time: now, Valid: true}},
				{ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
					Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeShipping, Valid: true},
					PostalCode: "22222", Street: "St B", City: "City B", State: "BB",
					Country:   sql.NullString{String: "US", Valid: true},
					IsMain:    sql.NullBool{Bool: false, Valid: true},
					CreatedAt: sql.NullTime{Time: now.Add(-time.Hour), Valid: true}},
			}, nil
		},
	}

	svc := newAddressTestService(mock)
	list, err := svc.List(context.Background(), tenantID, customerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(list))
	}
}

func TestAddressGetByID_Found(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{
				ID: addressID, TenantID: tenantID, CustomerID: customerID,
				Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
				PostalCode: "12345", Street: "Main St", City: "City", State: "ST",
				Country:   sql.NullString{String: "BR", Valid: true},
				IsMain:    sql.NullBool{Bool: true, Valid: true},
				CreatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
	}

	svc := newAddressTestService(mock)
	addr, err := svc.GetByID(context.Background(), tenantID, customerID, addressID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if addr.ID != addressID {
		t.Fatalf("expected ID %v, got %v", addressID, addr.ID)
	}
}

func TestAddressGetByID_NotFound(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	_, err := svc.GetByID(context.Background(), tenantID, customerID, uuid.New())
	if !errors.Is(err, domain.ErrAddressNotFound) {
		t.Fatalf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestAddressUpdate_Found(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	now := time.Now().Truncate(time.Second)
	addrType := "home"
	country := "BR"
	isMain := true

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		updateAddressFn: func(_ context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
			return dbgen.Address{
				ID: addressID, TenantID: tenantID, CustomerID: customerID,
				Type:         arg.Type,
				PostalCode:   arg.PostalCode,
				Street:       arg.Street,
				City:         arg.City,
				State:        arg.State,
				Country:      arg.Country,
				IsMain:       arg.IsMain,
				Number:       arg.Number,
				Complement:   arg.Complement,
				Neighborhood: arg.Neighborhood,
				CreatedAt:    sql.NullTime{Time: now, Valid: true},
			}, nil
		},
	}

	svc := newAddressTestService(mock)
	addr, err := svc.Update(context.Background(), tenantID, customerID, addressID, domain.UpdateAddressInput{
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
	tenantID := uuid.New()
	customerID := uuid.New()

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		updateAddressFn: func(_ context.Context, _ dbgen.UpdateAddressParams) (dbgen.Address, error) {
			return dbgen.Address{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	_, err := svc.Update(context.Background(), tenantID, customerID, uuid.New(), domain.UpdateAddressInput{
		PostalCode: "12345", Street: "St", City: "C", State: "ST",
	})
	if !errors.Is(err, domain.ErrAddressNotFound) {
		t.Fatalf("expected ErrAddressNotFound, got %v", err)
	}
}

func TestAddressDelete_Found(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	now := time.Now().Truncate(time.Second)

	deleteCalled := false
	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{
				ID: addressID, TenantID: tenantID, CustomerID: customerID,
				PostalCode: "12345", Street: "St", City: "C", State: "ST",
				CreatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
		deleteAddressFn: func(_ context.Context, arg dbgen.DeleteAddressParams) error {
			deleteCalled = true
			return nil
		},
	}

	svc := newAddressTestService(mock)
	err := svc.Delete(context.Background(), tenantID, customerID, addressID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected DeleteAddress to be called")
	}
}

func TestAddressDelete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	mock := &addressMockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	err := svc.Delete(context.Background(), tenantID, customerID, uuid.New())
	if !errors.Is(err, domain.ErrAddressNotFound) {
		t.Fatalf("expected ErrAddressNotFound, got %v", err)
	}
}

// --- Customer existence verification tests ---

func TestAddressCreate_CustomerNotFound(t *testing.T) {
	mock := &addressMockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), domain.CreateAddressInput{
		PostalCode: "12345", Street: "St", City: "C", State: "ST",
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressList_CustomerNotFound(t *testing.T) {
	mock := &addressMockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	_, err := svc.List(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressGetByID_CustomerNotFound(t *testing.T) {
	mock := &addressMockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressUpdate_CustomerNotFound(t *testing.T) {
	mock := &addressMockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), uuid.New(), domain.UpdateAddressInput{
		PostalCode: "12345", Street: "St", City: "C", State: "ST",
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestAddressDelete_CustomerNotFound(t *testing.T) {
	mock := &addressMockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}

	svc := newAddressTestService(mock)
	err := svc.Delete(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}
