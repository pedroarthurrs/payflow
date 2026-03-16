package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"testing/quick"
	"time"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"
)

// mockQuerier implements dbgen.Querier for testing.
type mockQuerier struct {
	createCustomerFn        func(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error)
	getCustomerByIDFn       func(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error)
	getTenantFn             func(ctx context.Context, id uuid.UUID) (dbgen.Tenant, error)
	listCustomersByTenantFn func(ctx context.Context, tenantID uuid.UUID) ([]dbgen.Customer, error)
	listTenantsFn           func(ctx context.Context) ([]dbgen.Tenant, error)
	softDeleteCustomerFn    func(ctx context.Context, arg dbgen.SoftDeleteCustomerParams) error
	updateCustomerFn        func(ctx context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error)
	hardDeleteCustomerFn    func(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error
}

func (m *mockQuerier) CreateCustomer(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
	if m.createCustomerFn != nil {
		return m.createCustomerFn(ctx, arg)
	}
	return dbgen.Customer{}, nil
}

func (m *mockQuerier) GetCustomerByID(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
	if m.getCustomerByIDFn != nil {
		return m.getCustomerByIDFn(ctx, arg)
	}
	return dbgen.Customer{}, nil
}

func (m *mockQuerier) GetTenant(ctx context.Context, id uuid.UUID) (dbgen.Tenant, error) {
	if m.getTenantFn != nil {
		return m.getTenantFn(ctx, id)
	}
	return dbgen.Tenant{}, nil
}

func (m *mockQuerier) ListCustomersByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbgen.Customer, error) {
	if m.listCustomersByTenantFn != nil {
		return m.listCustomersByTenantFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockQuerier) ListTenants(ctx context.Context) ([]dbgen.Tenant, error) {
	if m.listTenantsFn != nil {
		return m.listTenantsFn(ctx)
	}
	return nil, nil
}

func (m *mockQuerier) SoftDeleteCustomer(ctx context.Context, arg dbgen.SoftDeleteCustomerParams) error {
	if m.softDeleteCustomerFn != nil {
		return m.softDeleteCustomerFn(ctx, arg)
	}
	return nil
}

func (m *mockQuerier) UpdateCustomer(ctx context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
	if m.updateCustomerFn != nil {
		return m.updateCustomerFn(ctx, arg)
	}
	return dbgen.Customer{}, nil
}

func (m *mockQuerier) HardDeleteCustomer(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error {
	if m.hardDeleteCustomerFn != nil {
		return m.hardDeleteCustomerFn(ctx, arg)
	}
	return nil
}

// newTestService creates a CustomerService backed by the given mock querier.
func newTestService(mock *mockQuerier) *CustomerService {
	return NewCustomerService(&database.DB{Queries: mock})
}

// --- toDomainCustomer tests ---

func TestToDomainCustomer_AllFieldsPopulated(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	phone := "123456789"
	meta := json.RawMessage(`{"key":"value"}`)
	id := uuid.New()
	tenantID := uuid.New()

	row := dbgen.Customer{
		ID:           id,
		TenantID:     tenantID,
		Name:         "Alice",
		Email:        "alice@example.com",
		Phone:        sql.NullString{String: phone, Valid: true},
		Document:     "12345678900",
		DocumentType: "cpf",
		Status:       sql.NullString{String: "active", Valid: true},
		Metadata:     pqtype.NullRawMessage{RawMessage: meta, Valid: true},
		CreatedAt:    sql.NullTime{Time: now, Valid: true},
		UpdatedAt:    sql.NullTime{Time: now, Valid: true},
	}

	c := toDomainCustomer(row)

	if c.ID != id {
		t.Fatalf("expected ID %v, got %v", id, c.ID)
	}
	if c.TenantID != tenantID {
		t.Fatalf("expected TenantID %v, got %v", tenantID, c.TenantID)
	}
	if c.Name != "Alice" {
		t.Fatalf("expected Name Alice, got %s", c.Name)
	}
	if c.Email != "alice@example.com" {
		t.Fatalf("expected Email alice@example.com, got %s", c.Email)
	}
	if c.Phone == nil || *c.Phone != phone {
		t.Fatalf("expected Phone %s, got %v", phone, c.Phone)
	}
	if c.Document != "12345678900" {
		t.Fatalf("expected Document 12345678900, got %s", c.Document)
	}
	if c.DocumentType != "cpf" {
		t.Fatalf("expected DocumentType cpf, got %s", c.DocumentType)
	}
	if c.Status != "active" {
		t.Fatalf("expected Status active, got %s", c.Status)
	}
	if string(c.Metadata) != `{"key":"value"}` {
		t.Fatalf("expected Metadata {\"key\":\"value\"}, got %s", string(c.Metadata))
	}
	if !c.CreatedAt.Equal(now) {
		t.Fatalf("expected CreatedAt %v, got %v", now, c.CreatedAt)
	}
	if !c.UpdatedAt.Equal(now) {
		t.Fatalf("expected UpdatedAt %v, got %v", now, c.UpdatedAt)
	}
}

func TestToDomainCustomer_NullableFieldsEmpty(t *testing.T) {
	row := dbgen.Customer{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Bob",
		Email:        "bob@example.com",
		Document:     "98765432100",
		DocumentType: "cnpj",
		Phone:        sql.NullString{Valid: false},
		Status:       sql.NullString{Valid: false},
		Metadata:     pqtype.NullRawMessage{Valid: false},
		CreatedAt:    sql.NullTime{Valid: false},
		UpdatedAt:    sql.NullTime{Valid: false},
	}

	c := toDomainCustomer(row)

	if c.Phone != nil {
		t.Fatalf("expected Phone nil, got %v", c.Phone)
	}
	if c.Status != "" {
		t.Fatalf("expected Status empty, got %s", c.Status)
	}
	if c.Metadata != nil {
		t.Fatalf("expected Metadata nil, got %v", c.Metadata)
	}
	if !c.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt zero, got %v", c.CreatedAt)
	}
	if !c.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt zero, got %v", c.UpdatedAt)
	}
}

// --- sql.ErrNoRows → domain.ErrCustomerNotFound tests ---

func TestGetByID_NotFound_ReturnsDomainError(t *testing.T) {
	mock := &mockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}
	svc := newTestService(mock)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected domain.ErrCustomerNotFound, got %v", err)
	}
}

func TestUpdate_NotFound_ReturnsDomainError(t *testing.T) {
	mock := &mockQuerier{
		updateCustomerFn: func(_ context.Context, _ dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}
	svc := newTestService(mock)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), domain.UpdateCustomerInput{
		Name:         "Test",
		Email:        "test@example.com",
		Document:     "123",
		DocumentType: "cpf",
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected domain.ErrCustomerNotFound, got %v", err)
	}
}

// --- Unique constraint violation tests ---

func TestHandleWriteError_EmailUniqueViolation(t *testing.T) {
	svc := &CustomerService{}

	pqErr := &pq.Error{Code: "23505", Constraint: "unique_customer_email_per_tenant"}
	err := svc.handleWriteError(pqErr)

	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apierr.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected status 409, got %d", appErr.Status)
	}
	if appErr.Message != "customer with this email already exists" {
		t.Fatalf("expected message about email, got %q", appErr.Message)
	}
}

func TestHandleWriteError_DocumentUniqueViolation(t *testing.T) {
	svc := &CustomerService{}

	pqErr := &pq.Error{Code: "23505", Constraint: "unique_customer_doc_per_tenant"}
	err := svc.handleWriteError(pqErr)

	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apierr.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected status 409, got %d", appErr.Status)
	}
	if appErr.Message != "customer with this document already exists" {
		t.Fatalf("expected message about document, got %q", appErr.Message)
	}
}

func TestHandleWriteError_UnknownConstraint_ReturnsInternal(t *testing.T) {
	svc := &CustomerService{}

	pqErr := &pq.Error{Code: "23505", Constraint: "some_other_constraint"}
	err := svc.handleWriteError(pqErr)

	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apierr.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected status 500, got %d", appErr.Status)
	}
}

func TestHandleWriteError_NonPqError_ReturnsInternal(t *testing.T) {
	svc := &CustomerService{}

	err := svc.handleWriteError(errors.New("some db error"))

	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apierr.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected status 500, got %d", appErr.Status)
	}
}

func TestCreate_UniqueEmailViolation_ReturnsConflict(t *testing.T) {
	mock := &mockQuerier{
		createCustomerFn: func(_ context.Context, _ dbgen.CreateCustomerParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, &pq.Error{Code: "23505", Constraint: "unique_customer_email_per_tenant"}
		},
	}
	svc := newTestService(mock)

	_, err := svc.Create(context.Background(), uuid.New(), domain.CreateCustomerInput{
		Name:         "Test",
		Email:        "dup@example.com",
		Document:     "123",
		DocumentType: "cpf",
	})

	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apierr.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected status 409, got %d", appErr.Status)
	}
}

// --- Property-based tests ---

// **Validates: Requirements 1.1, 1.7, 1.8, 1.10, 3.1, 6.2**
func TestProperty_CreateAndGetByID_RoundTrip(t *testing.T) {
	// Property 1: Round-trip of creation and read.
	// For any valid creation input, creating a customer and then reading it
	// by ID should return the same data with a UUID v4, status "active",
	// and the correct tenant_id.

	f := func(seed uint64) bool {
		// Use seed to generate deterministic but varied inputs.
		r := seed

		// Simple pseudo-random string generator from seed.
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
		emailLocal := randStr("", 6)
		email := emailLocal + "@test.com"
		document := randStr("doc-", 10)
		documentType := randStr("dt-", 4)
		phoneStr := randStr("", 10)

		// Randomly decide if phone and metadata are present.
		var phone *string
		if seed%2 == 0 {
			phone = &phoneStr
		}

		var metadata json.RawMessage
		if seed%3 == 0 {
			metadata = json.RawMessage(`{"key":"` + randStr("v-", 4) + `"}`)
		}

		tenantID := uuid.New()
		generatedID := uuid.New()
		now := time.Now().Truncate(time.Second)

		// This variable will hold the customer "stored" by CreateCustomer.
		var stored dbgen.Customer

		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				stored = dbgen.Customer{
					ID:           generatedID,
					TenantID:     arg.TenantID,
					Name:         arg.Name,
					Email:        arg.Email,
					Phone:        arg.Phone,
					Document:     arg.Document,
					DocumentType: arg.DocumentType,
					Status:       sql.NullString{String: "active", Valid: true},
					Metadata:     arg.Metadata,
					CreatedAt:    sql.NullTime{Time: now, Valid: true},
					UpdatedAt:    sql.NullTime{Time: now, Valid: true},
				}
				return stored, nil
			},
			getCustomerByIDFn: func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				if arg.ID == stored.ID && arg.TenantID == stored.TenantID {
					return stored, nil
				}
				return dbgen.Customer{}, sql.ErrNoRows
			},
		}

		svc := newTestService(mock)
		ctx := context.Background()

		input := domain.CreateCustomerInput{
			Name:         name,
			Email:        email,
			Phone:        phone,
			Document:     document,
			DocumentType: documentType,
			Metadata:     metadata,
		}

		created, err := svc.Create(ctx, tenantID, input)
		if err != nil {
			t.Logf("Create failed: %v", err)
			return false
		}

		got, err := svc.GetByID(ctx, tenantID, created.ID)
		if err != nil {
			t.Logf("GetByID failed: %v", err)
			return false
		}

		// Verify UUID v4 format (version nibble = 4, variant bits = 10xx).
		idBytes := got.ID
		if (idBytes[6] >> 4) != 4 {
			t.Logf("ID version is not 4: got %d", idBytes[6]>>4)
			return false
		}
		if (idBytes[8] >> 6) != 2 {
			t.Logf("ID variant is not RFC 4122: got %d", idBytes[8]>>6)
			return false
		}

		// Verify status is "active".
		if got.Status != "active" {
			t.Logf("expected status 'active', got %q", got.Status)
			return false
		}

		// Verify tenant_id matches.
		if got.TenantID != tenantID {
			t.Logf("expected TenantID %v, got %v", tenantID, got.TenantID)
			return false
		}

		// Verify all input fields are preserved.
		if got.Name != name {
			t.Logf("expected Name %q, got %q", name, got.Name)
			return false
		}
		if got.Email != email {
			t.Logf("expected Email %q, got %q", email, got.Email)
			return false
		}
		if got.Document != document {
			t.Logf("expected Document %q, got %q", document, got.Document)
			return false
		}
		if got.DocumentType != documentType {
			t.Logf("expected DocumentType %q, got %q", documentType, got.DocumentType)
			return false
		}

		// Verify phone round-trip.
		if phone == nil {
			if got.Phone != nil {
				t.Logf("expected Phone nil, got %v", got.Phone)
				return false
			}
		} else {
			if got.Phone == nil || *got.Phone != *phone {
				t.Logf("expected Phone %q, got %v", *phone, got.Phone)
				return false
			}
		}

		// Verify metadata round-trip.
		if metadata == nil {
			if got.Metadata != nil {
				t.Logf("expected Metadata nil, got %v", got.Metadata)
				return false
			}
		} else {
			if string(got.Metadata) != string(metadata) {
				t.Logf("expected Metadata %s, got %s", string(metadata), string(got.Metadata))
				return false
			}
		}

		return true
	}

	// Run the property with 100 iterations using testing/quick.
	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("Property 1 failed: %v", err)
	}
}

// **Validates: Requirements 2.1, 6.1, 6.3**
func TestProperty_MultiTenantIsolation(t *testing.T) {
	// Property 5: Multi-tenant isolation.
	// Customers from tenant A should never be visible to tenant B
	// in any operation (List, GetByID, Update, Delete).

	f := func(seed uint64) bool {
		r := seed

		// Simple pseudo-random string generator from seed.
		randStr := func(prefix string, n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				r = r*6364136223846793005 + 1
				buf[i] = chars[r%uint64(len(chars))]
			}
			return prefix + string(buf)
		}

		// Generate two distinct tenant UUIDs.
		tenantA := uuid.New()
		tenantB := uuid.New()

		// Generate random customer input.
		name := randStr("name-", 8)
		email := randStr("", 6) + "@test.com"
		document := randStr("doc-", 10)
		documentType := randStr("dt-", 4)

		customerID := uuid.New()
		now := time.Now().Truncate(time.Second)

		// In-memory tenant-scoped storage: maps tenantID -> []dbgen.Customer
		store := make(map[uuid.UUID][]dbgen.Customer)

		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				c := dbgen.Customer{
					ID:           customerID,
					TenantID:     arg.TenantID,
					Name:         arg.Name,
					Email:        arg.Email,
					Phone:        arg.Phone,
					Document:     arg.Document,
					DocumentType: arg.DocumentType,
					Status:       sql.NullString{String: "active", Valid: true},
					Metadata:     arg.Metadata,
					CreatedAt:    sql.NullTime{Time: now, Valid: true},
					UpdatedAt:    sql.NullTime{Time: now, Valid: true},
				}
				store[arg.TenantID] = append(store[arg.TenantID], c)
				return c, nil
			},
			listCustomersByTenantFn: func(_ context.Context, tid uuid.UUID) ([]dbgen.Customer, error) {
				return store[tid], nil
			},
			getCustomerByIDFn: func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				for _, c := range store[arg.TenantID] {
					if c.ID == arg.ID {
						return c, nil
					}
				}
				return dbgen.Customer{}, sql.ErrNoRows
			},
			updateCustomerFn: func(_ context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
				for i, c := range store[arg.TenantID] {
					if c.ID == arg.ID {
						store[arg.TenantID][i].Name = arg.Name
						store[arg.TenantID][i].Email = arg.Email
						store[arg.TenantID][i].UpdatedAt = sql.NullTime{Time: time.Now(), Valid: true}
						return store[arg.TenantID][i], nil
					}
				}
				return dbgen.Customer{}, sql.ErrNoRows
			},
			softDeleteCustomerFn: func(_ context.Context, arg dbgen.SoftDeleteCustomerParams) error {
				for _, c := range store[arg.TenantID] {
					if c.ID == arg.ID {
						return nil
					}
				}
				return nil // :exec doesn't return ErrNoRows
			},
		}

		svc := newTestService(mock)
		ctx := context.Background()

		// Create a customer for tenant A.
		input := domain.CreateCustomerInput{
			Name:         name,
			Email:        email,
			Document:     document,
			DocumentType: documentType,
		}
		created, err := svc.Create(ctx, tenantA, input)
		if err != nil {
			t.Logf("Create for tenant A failed: %v", err)
			return false
		}

		// --- Tenant B should NOT see tenant A's customer ---

		// List: tenant B should get empty list.
		listB, err := svc.List(ctx, tenantB)
		if err != nil {
			t.Logf("List for tenant B failed: %v", err)
			return false
		}
		if len(listB) != 0 {
			t.Logf("tenant B List returned %d customers, expected 0", len(listB))
			return false
		}

		// GetByID: tenant B should get ErrCustomerNotFound.
		_, err = svc.GetByID(ctx, tenantB, created.ID)
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			t.Logf("tenant B GetByID: expected ErrCustomerNotFound, got %v", err)
			return false
		}

		// Update: tenant B should get ErrCustomerNotFound.
		_, err = svc.Update(ctx, tenantB, created.ID, domain.UpdateCustomerInput{
			Name:         "hacked",
			Email:        "hacked@evil.com",
			Document:     document,
			DocumentType: documentType,
		})
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			t.Logf("tenant B Update: expected ErrCustomerNotFound, got %v", err)
			return false
		}

		// Delete: tenant B should get ErrCustomerNotFound.
		// (Delete first calls GetByID internally, which returns ErrNoRows for wrong tenant)
		err = svc.Delete(ctx, tenantB, created.ID)
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			t.Logf("tenant B Delete: expected ErrCustomerNotFound, got %v", err)
			return false
		}

		// --- Tenant A should still see the customer ---

		// List: tenant A should see exactly 1 customer.
		listA, err := svc.List(ctx, tenantA)
		if err != nil {
			t.Logf("List for tenant A failed: %v", err)
			return false
		}
		if len(listA) != 1 {
			t.Logf("tenant A List returned %d customers, expected 1", len(listA))
			return false
		}
		if listA[0].ID != created.ID {
			t.Logf("tenant A List: expected ID %v, got %v", created.ID, listA[0].ID)
			return false
		}

		// GetByID: tenant A should find the customer.
		gotA, err := svc.GetByID(ctx, tenantA, created.ID)
		if err != nil {
			t.Logf("tenant A GetByID failed: %v", err)
			return false
		}
		if gotA.ID != created.ID {
			t.Logf("tenant A GetByID: expected ID %v, got %v", created.ID, gotA.ID)
			return false
		}
		if gotA.TenantID != tenantA {
			t.Logf("tenant A GetByID: expected TenantID %v, got %v", tenantA, gotA.TenantID)
			return false
		}

		// Update: tenant A should succeed.
		updatedA, err := svc.Update(ctx, tenantA, created.ID, domain.UpdateCustomerInput{
			Name:         "updated-name",
			Email:        email,
			Document:     document,
			DocumentType: documentType,
		})
		if err != nil {
			t.Logf("tenant A Update failed: %v", err)
			return false
		}
		if updatedA.Name != "updated-name" {
			t.Logf("tenant A Update: expected Name 'updated-name', got %q", updatedA.Name)
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("Property 5 (Multi-tenant isolation) failed: %v", err)
	}
}

// --- HardDelete tests ---

func TestHardDelete_Success(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	deleteCalled := false
	mock := &mockQuerier{
		hardDeleteCustomerFn: func(_ context.Context, arg dbgen.HardDeleteCustomerParams) error {
			if arg.ID == customerID && arg.TenantID == tenantID {
				deleteCalled = true
				return nil
			}
			return nil
		},
	}

	svc := newTestService(mock)

	err := svc.HardDelete(context.Background(), tenantID, customerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !deleteCalled {
		t.Fatal("expected HardDeleteCustomer to be called")
	}
}

func TestHardDelete_DatabaseError(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	mock := &mockQuerier{
		hardDeleteCustomerFn: func(_ context.Context, _ dbgen.HardDeleteCustomerParams) error {
			return errors.New("database connection failed")
		},
	}

	svc := newTestService(mock)

	err := svc.HardDelete(context.Background(), tenantID, customerID)

	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apierr.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected status 500, got %d", appErr.Status)
	}
}
