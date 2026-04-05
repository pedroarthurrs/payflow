package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"testing/quick"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ============================================================
// Unit tests
// ============================================================

func TestGetByID_NotFound_ReturnsDomainError(t *testing.T) {
	customers := &mockCustomerRepo{getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
		return nil, domain.ErrCustomerNotFound
	}}
	svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestUpdate_NotFound_ReturnsDomainError(t *testing.T) {
	customers := &mockCustomerRepo{updateFn: func(_ context.Context, _, _ uuid.UUID, _ domain.CustomerInput) (*domain.Customer, error) {
		return nil, domain.ErrCustomerNotFound
	}}
	svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
	_, _, err := svc.Update(context.Background(), uuid.New(), uuid.New(), domain.CustomerInput{
		Name: "Test", Email: "test@example.com", Document: "123", DocumentType: "cpf",
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestCreate_UniqueEmailViolation_ReturnsConflict(t *testing.T) {
	customers := &mockCustomerRepo{createFn: func(_ context.Context, _ uuid.UUID, _ domain.CustomerInput) (*domain.Customer, error) {
		return nil, domain.ErrDuplicateEmail
	}}
	svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
	_, _, err := svc.Create(context.Background(), uuid.New(), domain.CustomerInput{
		Name: "Test", Email: "dup@example.com", Document: "123", DocumentType: "cpf",
	})
	if !errors.Is(err, domain.ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestHardDelete_Success(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	deleteCalled := false
	customers := &mockCustomerRepo{hardDeleteFn: func(_ context.Context, orgID, custID uuid.UUID) error {
		if orgID == organizationID && custID == customerID {
			deleteCalled = true
		}
		return nil
	}}
	svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
	err := svc.HardDelete(context.Background(), organizationID, customerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected HardDeleteCustomer to be called")
	}
}

func TestHardDelete_DatabaseError(t *testing.T) {
	customers := &mockCustomerRepo{hardDeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
		return domain.ErrInternal
	}}
	svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
	err := svc.HardDelete(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

// ============================================================
// Transaction tests
// ============================================================

func TestCreate_WithAddress_BothCreated(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	now := time.Now().Truncate(time.Second)

	customerCreated := false
	addressCreated := false

	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(tx domain.TxHandle) error) error {
			return fn(&mockTxHandle{
				customers: &mockCustomerRepo{
					createFn: func(_ context.Context, orgID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
						customerCreated = true
						return &domain.Customer{
							ID: customerID, OrganizationID: orgID,
							Name: input.Name, Email: input.Email,
							Document: input.Document, DocumentType: input.DocumentType,
							Status: "active", CreatedAt: now, UpdatedAt: now,
						}, nil
					},
				},
				addresses: &mockAddressRepo{
					createFn: func(_ context.Context, orgID, custID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
						addressCreated = true
						return &domain.Address{
							ID: addressID, OrganizationID: orgID, CustomerID: custID,
							Type: "billing", PostalCode: input.PostalCode, Street: input.Street,
							City: input.City, State: input.State, Country: "BR", IsMain: true,
							CreatedAt: now,
						}, nil
					},
				},
			})
		},
	}

	svc := NewCustomerService(&mockCustomerRepo{}, &mockAddressRepo{}, txMgr)
	cust, addr, err := svc.Create(context.Background(), organizationID, domain.CustomerInput{
		Name: "Alice", Email: "alice@test.com", Document: "12345", DocumentType: "cpf",
		Address: &domain.AddressInput{
			PostalCode: "12345-678", Street: "Main St", City: "Springfield", State: "IL",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cust == nil {
		t.Fatal("expected customer, got nil")
	}
	if addr == nil {
		t.Fatal("expected address, got nil")
	}
	if !customerCreated {
		t.Fatal("customer was not created in tx")
	}
	if !addressCreated {
		t.Fatal("address was not created in tx")
	}
}

func TestCreate_WithoutAddress_OnlyCustomerCreated(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	customers := &mockCustomerRepo{
		createFn: func(_ context.Context, orgID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
			return &domain.Customer{
				ID: uuid.New(), OrganizationID: orgID,
				Name: input.Name, Email: input.Email,
				Document: input.Document, DocumentType: input.DocumentType,
				Status: "active", CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
	cust, addr, err := svc.Create(context.Background(), uuid.New(), domain.CustomerInput{
		Name: "Bob", Email: "bob@test.com", Document: "67890", DocumentType: "cpf",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cust == nil {
		t.Fatal("expected customer, got nil")
	}
	if addr != nil {
		t.Fatalf("expected nil address, got %v", addr)
	}
}

func TestCreate_WithAddress_AddressFailure_Rollback(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	now := time.Now().Truncate(time.Second)

	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(tx domain.TxHandle) error) error {
			return fn(&mockTxHandle{
				customers: &mockCustomerRepo{
					createFn: func(_ context.Context, orgID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
						return &domain.Customer{
							ID: customerID, OrganizationID: orgID,
							Name: input.Name, Email: input.Email,
							Document: input.Document, DocumentType: input.DocumentType,
							Status: "active", CreatedAt: now, UpdatedAt: now,
						}, nil
					},
				},
				addresses: &mockAddressRepo{
					createFn: func(_ context.Context, _, _ uuid.UUID, _ domain.AddressInput) (*domain.Address, error) {
						return nil, errors.New("address creation failed")
					},
				},
			})
		},
	}

	svc := NewCustomerService(&mockCustomerRepo{}, &mockAddressRepo{}, txMgr)
	cust, addr, err := svc.Create(context.Background(), organizationID, domain.CustomerInput{
		Name: "Charlie", Email: "charlie@test.com", Document: "11111", DocumentType: "cpf",
		Address: &domain.AddressInput{
			PostalCode: "12345-678", Street: "Main St", City: "Springfield", State: "IL",
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if cust != nil {
		t.Fatal("expected nil customer on rollback")
	}
	if addr != nil {
		t.Fatal("expected nil address on rollback")
	}
}

func TestUpdate_WithoutAddress_OnlyCustomerUpdated(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	customers := &mockCustomerRepo{
		updateFn: func(_ context.Context, orgID, custID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
			return &domain.Customer{
				ID: custID, OrganizationID: orgID,
				Name: input.Name, Email: input.Email,
				Document: input.Document, DocumentType: input.DocumentType,
				Status: "active", CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
	cust, addr, err := svc.Update(context.Background(), uuid.New(), uuid.New(), domain.CustomerInput{
		Name: "Updated", Email: "updated@test.com", Document: "123", DocumentType: "cpf",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cust == nil {
		t.Fatal("expected customer")
		return
	}
	if addr != nil {
		t.Fatalf("expected nil address, got %v", addr)
	}
	if cust.Name != "Updated" {
		t.Fatalf("expected name Updated, got %s", cust.Name)
	}
}

// ============================================================
// Property-based tests
// ============================================================

func TestProperty_CreateAndGetByID_RoundTrip(t *testing.T) {
	f := func(seed uint64) bool {
		r := seed
		randStr := func(prefix string, n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				r = r*6364136223846793005 + 1
				buf[i] = chars[r%uint64(len(chars))]
			}
			return prefix + string(buf)
		}
		name := randStr("name-", 8)
		email := randStr("", 6) + "@test.com"
		document := randStr("doc-", 10)
		documentType := randStr("dt-", 4)
		phoneStr := randStr("", 10)
		var phone *string
		if seed%2 == 0 {
			phone = &phoneStr
		}
		var metadata json.RawMessage
		if seed%3 == 0 {
			metadata = json.RawMessage(`{"key":"` + randStr("v-", 4) + `"}`)
		}
		organizationID := uuid.New()
		generatedID := uuid.New()
		now := time.Now().Truncate(time.Second)
		var stored *domain.Customer
		customers := &mockCustomerRepo{
			createFn: func(_ context.Context, orgID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
				stored = &domain.Customer{
					ID: generatedID, OrganizationID: orgID,
					Name: input.Name, Email: input.Email,
					Phone: input.Phone, Document: input.Document, DocumentType: input.DocumentType,
					Status: "active", Metadata: input.Metadata,
					CreatedAt: now, UpdatedAt: now,
				}
				return stored, nil
			},
			getByIDFn: func(_ context.Context, orgID, custID uuid.UUID) (*domain.Customer, error) {
				if stored != nil && custID == stored.ID && orgID == stored.OrganizationID {
					return stored, nil
				}
				return nil, domain.ErrCustomerNotFound
			},
		}
		svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
		ctx := context.Background()
		created, _, err := svc.Create(ctx, organizationID, domain.CustomerInput{
			Name: name, Email: email, Phone: phone, Document: document, DocumentType: documentType, Metadata: metadata,
		})
		if err != nil {
			t.Logf("Create failed: %v", err)
			return false
		}
		got, err := svc.GetByID(ctx, organizationID, created.ID)
		if err != nil {
			t.Logf("GetByID failed: %v", err)
			return false
		}
		if (got.ID[6] >> 4) != 4 {
			return false
		}
		if (got.ID[8] >> 6) != 2 {
			return false
		}
		if got.Status != "active" {
			return false
		}
		if got.OrganizationID != organizationID {
			return false
		}
		if got.Name != name || got.Email != email || got.Document != document || got.DocumentType != documentType {
			return false
		}
		if phone == nil {
			if got.Phone != nil {
				return false
			}
		} else {
			if got.Phone == nil || *got.Phone != *phone {
				return false
			}
		}
		if metadata == nil {
			if got.Metadata != nil {
				return false
			}
		} else {
			if string(got.Metadata) != string(metadata) {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
}

func TestProperty_MultiOrganizationIsolation(t *testing.T) {
	f := func(seed uint64) bool {
		r := seed
		randStr := func(prefix string, n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				r = r*6364136223846793005 + 1
				buf[i] = chars[r%uint64(len(chars))]
			}
			return prefix + string(buf)
		}
		organizationA := uuid.New()
		organizationB := uuid.New()
		name := randStr("name-", 8)
		email := randStr("", 6) + "@test.com"
		document := randStr("doc-", 10)
		documentType := randStr("dt-", 4)
		customerID := uuid.New()
		now := time.Now().Truncate(time.Second)
		store := make(map[uuid.UUID][]*domain.Customer)
		customers := &mockCustomerRepo{
			createFn: func(_ context.Context, orgID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
				c := &domain.Customer{
					ID: customerID, OrganizationID: orgID, Name: input.Name, Email: input.Email,
					Phone: input.Phone, Document: input.Document, DocumentType: input.DocumentType,
					Status: "active", Metadata: input.Metadata,
					CreatedAt: now, UpdatedAt: now,
				}
				store[orgID] = append(store[orgID], c)
				return c, nil
			},
			listFn: func(_ context.Context, orgID uuid.UUID) ([]domain.Customer, error) {
				result := make([]domain.Customer, len(store[orgID]))
				for i, c := range store[orgID] {
					result[i] = *c
				}
				return result, nil
			},
			getByIDFn: func(_ context.Context, orgID, custID uuid.UUID) (*domain.Customer, error) {
				for _, c := range store[orgID] {
					if c.ID == custID {
						return c, nil
					}
				}
				return nil, domain.ErrCustomerNotFound
			},
			updateFn: func(_ context.Context, orgID, custID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
				for _, c := range store[orgID] {
					if c.ID == custID {
						c.Name = input.Name
						return c, nil
					}
				}
				return nil, domain.ErrCustomerNotFound
			},
			softDeleteFn: func(_ context.Context, orgID, custID uuid.UUID) error {
				for _, c := range store[orgID] {
					if c.ID == custID {
						return nil
					}
				}
				return domain.ErrCustomerNotFound
			},
		}
		svc := NewCustomerService(customers, &mockAddressRepo{}, &mockTxManager{})
		ctx := context.Background()
		created, _, err := svc.Create(ctx, organizationA, domain.CustomerInput{Name: name, Email: email, Document: document, DocumentType: documentType})
		if err != nil {
			return false
		}
		listB, _ := svc.List(ctx, organizationB)
		if len(listB) != 0 {
			return false
		}
		_, err = svc.GetByID(ctx, organizationB, created.ID)
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			return false
		}
		_, _, err = svc.Update(ctx, organizationB, created.ID, domain.CustomerInput{Name: "hacked", Email: "h@e.com", Document: document, DocumentType: documentType})
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			return false
		}
		err = svc.Delete(ctx, organizationB, created.ID)
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			return false
		}
		listA, _ := svc.List(ctx, organizationA)
		if len(listA) != 1 || listA[0].ID != created.ID {
			return false
		}
		updatedA, _, err := svc.Update(ctx, organizationA, created.ID, domain.CustomerInput{Name: "updated", Email: email, Document: document, DocumentType: documentType})
		if err != nil || updatedA.Name != "updated" {
			return false
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatalf("multi-organization isolation failed: %v", err)
	}
}

func TestProperty_AtomicCustomerWithAddress(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		customerID := uuid.New()
		addressID := uuid.New()
		now := time.Now().Truncate(time.Second)

		name := rapid.StringMatching(`[A-Za-z ]{3,30}`).Draw(t, "name")
		email := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "email_local") + "@test.com"
		document := rapid.StringMatching(`[0-9]{5,15}`).Draw(t, "document")
		documentType := rapid.SampledFrom([]string{"cpf", "cnpj"}).Draw(t, "doc_type")

		postalCode := rapid.StringMatching(`[0-9]{5}-[0-9]{3}`).Draw(t, "postal_code")
		street := rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "street")
		city := rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "city")
		state := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "state")

		customerCreated := false
		addressCreated := false

		txMgr := &mockTxManager{
			withTxFn: func(_ context.Context, fn func(tx domain.TxHandle) error) error {
				return fn(&mockTxHandle{
					customers: &mockCustomerRepo{
						createFn: func(_ context.Context, orgID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
							customerCreated = true
							return &domain.Customer{
								ID: customerID, OrganizationID: orgID,
								Name: input.Name, Email: input.Email,
								Document: input.Document, DocumentType: input.DocumentType,
								Status: "active", CreatedAt: now, UpdatedAt: now,
							}, nil
						},
					},
					addresses: &mockAddressRepo{
						createFn: func(_ context.Context, orgID, custID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
							addressCreated = true
							return &domain.Address{
								ID: addressID, OrganizationID: orgID, CustomerID: custID,
								Type: "billing", PostalCode: input.PostalCode, Street: input.Street,
								City: input.City, State: input.State, Country: "BR", IsMain: true,
								CreatedAt: now,
							}, nil
						},
					},
				})
			},
		}

		svc := NewCustomerService(&mockCustomerRepo{}, &mockAddressRepo{}, txMgr)
		cust, addr, err := svc.Create(context.Background(), organizationID, domain.CustomerInput{
			Name: name, Email: email, Document: document, DocumentType: documentType,
			Address: &domain.AddressInput{
				PostalCode: postalCode, Street: street, City: city, State: state,
			},
		})

		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if cust == nil {
			t.Fatal("expected customer, got nil")
			return
		}
		if addr == nil {
			t.Fatal("expected address, got nil")
			return
		}

		if !customerCreated {
			t.Fatal("customer not created")
		}
		if !addressCreated {
			t.Fatal("address not created")
		}

		if cust.OrganizationID != organizationID {
			t.Fatalf("expected organization %v, got %v", organizationID, cust.OrganizationID)
		}
		if addr.OrganizationID != organizationID {
			t.Fatalf("addr organization mismatch")
		}
		if addr.CustomerID != customerID {
			t.Fatalf("addr customer mismatch")
		}
	})
}

func TestProperty_CreateWithoutAddress_NoAddressCreated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		now := time.Now().Truncate(time.Second)

		name := rapid.StringMatching(`[A-Za-z ]{3,30}`).Draw(t, "name")
		email := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "email_local") + "@test.com"
		document := rapid.StringMatching(`[0-9]{5,15}`).Draw(t, "document")
		documentType := rapid.SampledFrom([]string{"cpf", "cnpj"}).Draw(t, "doc_type")

		txCalled := false
		customers := &mockCustomerRepo{
			createFn: func(_ context.Context, orgID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
				return &domain.Customer{
					ID: uuid.New(), OrganizationID: orgID,
					Name: input.Name, Email: input.Email,
					Document: input.Document, DocumentType: input.DocumentType,
					Status: "active", CreatedAt: now, UpdatedAt: now,
				}, nil
			},
		}
		txMgr := &mockTxManager{
			withTxFn: func(_ context.Context, fn func(tx domain.TxHandle) error) error {
				txCalled = true
				return fn(&mockTxHandle{})
			},
		}

		svc := NewCustomerService(customers, &mockAddressRepo{}, txMgr)
		cust, addr, err := svc.Create(context.Background(), organizationID, domain.CustomerInput{
			Name: name, Email: email, Document: document, DocumentType: documentType,
		})

		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if cust == nil {
			t.Fatal("expected customer, got nil")
		}
		if addr != nil {
			t.Fatalf("expected nil address, got %v", addr)
		}
		if txCalled {
			t.Fatal("tx should not be called when no address")
		}
	})
}
