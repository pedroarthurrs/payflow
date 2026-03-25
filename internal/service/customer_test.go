package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"sync"
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
	"pgregory.net/rapid"
)

// ============================================================
// Mock Querier
// ============================================================

type mockQuerier struct {
	createCustomerFn        func(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error)
	getCustomerByIDFn       func(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error)
	getTenantFn             func(ctx context.Context, id uuid.UUID) (dbgen.Tenant, error)
	listCustomersByTenantFn func(ctx context.Context, tenantID uuid.UUID) ([]dbgen.Customer, error)
	listTenantsFn           func(ctx context.Context) ([]dbgen.Tenant, error)
	softDeleteCustomerFn    func(ctx context.Context, arg dbgen.SoftDeleteCustomerParams) error
	updateCustomerFn        func(ctx context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error)
	hardDeleteCustomerFn    func(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error
	createAddressFn         func(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error)
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
func (m *mockQuerier) CreateAddress(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
	if m.createAddressFn != nil {
		return m.createAddressFn(ctx, arg)
	}
	return dbgen.Address{}, nil
}
func (m *mockQuerier) DeleteAddress(ctx context.Context, arg dbgen.DeleteAddressParams) error {
	return nil
}
func (m *mockQuerier) GetAddressByID(ctx context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}
func (m *mockQuerier) ListAddressesByCustomer(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
	return nil, nil
}
func (m *mockQuerier) UpdateAddress(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
	return dbgen.Address{}, nil
}

// Stubs for user/api-key methods added by rbac-dynamic-auth feature.
func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) GetUserByEmailAndTenant(ctx context.Context, arg dbgen.GetUserByEmailAndTenantParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) GetUserByID(ctx context.Context, arg dbgen.GetUserByIDParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) ListUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbgen.User, error) {
	return nil, nil
}
func (m *mockQuerier) SoftDeleteUser(ctx context.Context, arg dbgen.SoftDeleteUserParams) error {
	return nil
}
func (m *mockQuerier) UpdateUser(ctx context.Context, arg dbgen.UpdateUserParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) GetAPIKeyByHash(ctx context.Context, keyHash string) (dbgen.ApiKey, error) {
	return dbgen.ApiKey{}, nil
}
func (m *mockQuerier) UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func newTestService(mock *mockQuerier) *CustomerService {
	return NewCustomerService(&database.DB{Queries: mock}, nil)
}

// ============================================================
// Fake SQL driver for transaction testing
// ============================================================

// txTestStore holds data that the fake driver returns during transaction tests.
// Tests set these before calling Create/Update with address.
var txTestStore struct {
	mu                sync.Mutex
	customerRow       []driver.Value // 12 columns for CreateCustomer RETURNING
	addressRow        []driver.Value // 14 columns for CreateAddress RETURNING
	addressShouldFail bool
	customerCreated   bool
	addressCreated    bool
}

func resetTxTestStore() {
	txTestStore.mu.Lock()
	defer txTestStore.mu.Unlock()
	txTestStore.customerRow = nil
	txTestStore.addressRow = nil
	txTestStore.addressShouldFail = false
	txTestStore.customerCreated = false
	txTestStore.addressCreated = false
}

type txFakeDriver struct{}

func (d *txFakeDriver) Open(_ string) (driver.Conn, error) { return &txFakeConn{}, nil }

type txFakeConn struct{ queryCount int }

func (c *txFakeConn) Prepare(_ string) (driver.Stmt, error) { return &txFakeStmt{conn: c}, nil }
func (c *txFakeConn) Close() error                          { return nil }
func (c *txFakeConn) Begin() (driver.Tx, error)             { return &txFakeTx{}, nil }

type txFakeTx struct{}

func (t *txFakeTx) Commit() error   { return nil }
func (t *txFakeTx) Rollback() error { return nil }

type txFakeStmt struct{ conn *txFakeConn }

func (s *txFakeStmt) Close() error                                 { return nil }
func (s *txFakeStmt) NumInput() int                                { return -1 }
func (s *txFakeStmt) Exec(_ []driver.Value) (driver.Result, error) { return nil, nil }
func (s *txFakeStmt) Query(_ []driver.Value) (driver.Rows, error) {
	txTestStore.mu.Lock()
	defer txTestStore.mu.Unlock()
	s.conn.queryCount++
	// First query = CreateCustomer, second = CreateAddress
	if s.conn.queryCount == 1 {
		txTestStore.customerCreated = true
		return &txFakeResultRows{row: txTestStore.customerRow, cols: customerColumns()}, nil
	}
	if txTestStore.addressShouldFail {
		return nil, errors.New("address creation failed")
	}
	txTestStore.addressCreated = true
	return &txFakeResultRows{row: txTestStore.addressRow, cols: addressColumns()}, nil
}

func customerColumns() []string {
	return []string{"id", "tenant_id", "name", "email", "phone", "document", "document_type", "status", "metadata", "created_at", "updated_at", "deleted_at"}
}

func addressColumns() []string {
	return []string{"id", "tenant_id", "customer_id", "type", "postal_code", "street", "number", "complement", "neighborhood", "city", "state", "country", "is_main", "created_at"}
}

type txFakeResultRows struct {
	row  []driver.Value
	cols []string
	done bool
}

func (r *txFakeResultRows) Columns() []string { return r.cols }
func (r *txFakeResultRows) Close() error      { return nil }
func (r *txFakeResultRows) Next(dest []driver.Value) error {
	if r.done || r.row == nil {
		return io.EOF
	}
	r.done = true
	copy(dest, r.row)
	return nil
}

var registerTxOnce sync.Once

func openTxFakeDB() *sql.DB {
	registerTxOnce.Do(func() { sql.Register("txfakecustomerdriver", &txFakeDriver{}) })
	db, _ := sql.Open("txfakecustomerdriver", "")
	return db
}

func newTestServiceWithTx() *CustomerService {
	fakeDB := openTxFakeDB()
	realQueries := dbgen.New(fakeDB)
	db := &database.DB{DB: fakeDB, Queries: realQueries}
	addrDB := &database.DB{DB: fakeDB, Queries: realQueries}
	addrSvc := NewAddressService(addrDB)
	return NewCustomerService(db, addrSvc)
}

func makeCustomerRow(id, tenantID uuid.UUID, name, email, document, documentType string) []driver.Value {
	now := time.Now().Truncate(time.Second)
	return []driver.Value{
		id.String(),       // id
		tenantID.String(), // tenant_id
		name,              // name
		email,             // email
		nil,               // phone (NULL)
		document,          // document
		documentType,      // document_type
		"active",          // status
		nil,               // metadata (NULL)
		now,               // created_at
		now,               // updated_at
		nil,               // deleted_at (NULL)
	}
}

func makeAddressRow(addrID, tenantID, customerID uuid.UUID) []driver.Value {
	now := time.Now().Truncate(time.Second)
	return []driver.Value{
		addrID.String(),     // id
		tenantID.String(),   // tenant_id
		customerID.String(), // customer_id
		"billing",           // type
		"12345-678",         // postal_code
		"Main St",           // street
		nil,                 // number (NULL)
		nil,                 // complement (NULL)
		nil,                 // neighborhood (NULL)
		"Springfield",       // city
		"IL",                // state
		"BR",                // country
		true,                // is_main
		now,                 // created_at
	}
}

// ============================================================
// Existing unit tests
// ============================================================

func TestToDomainCustomer_AllFieldsPopulated(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	phone := "123456789"
	meta := json.RawMessage(`{"key":"value"}`)
	id := uuid.New()
	tid := uuid.New()
	row := dbgen.Customer{
		ID: id, TenantID: tid, Name: "Alice", Email: "alice@example.com",
		Phone: sql.NullString{String: phone, Valid: true}, Document: "12345678900", DocumentType: "cpf",
		Status:    sql.NullString{String: "active", Valid: true},
		Metadata:  pqtype.NullRawMessage{RawMessage: meta, Valid: true},
		CreatedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
	}
	c := toDomainCustomer(row)
	if c.ID != id {
		t.Fatalf("expected ID %v, got %v", id, c.ID)
	}
	if c.TenantID != tid {
		t.Fatalf("expected TenantID %v, got %v", tid, c.TenantID)
	}
	if c.Name != "Alice" {
		t.Fatalf("expected Name Alice, got %s", c.Name)
	}
	if c.Email != "alice@example.com" {
		t.Fatalf("got %s", c.Email)
	}
	if c.Phone == nil || *c.Phone != phone {
		t.Fatalf("expected Phone %s, got %v", phone, c.Phone)
	}
	if c.Document != "12345678900" {
		t.Fatalf("got %s", c.Document)
	}
	if c.DocumentType != "cpf" {
		t.Fatalf("got %s", c.DocumentType)
	}
	if c.Status != "active" {
		t.Fatalf("got %s", c.Status)
	}
	if string(c.Metadata) != `{"key":"value"}` {
		t.Fatalf("got %s", string(c.Metadata))
	}
	if !c.CreatedAt.Equal(now) {
		t.Fatalf("got %v", c.CreatedAt)
	}
	if !c.UpdatedAt.Equal(now) {
		t.Fatalf("got %v", c.UpdatedAt)
	}
}

func TestToDomainCustomer_NullableFieldsEmpty(t *testing.T) {
	row := dbgen.Customer{
		ID: uuid.New(), TenantID: uuid.New(), Name: "Bob", Email: "bob@example.com",
		Document: "98765432100", DocumentType: "cnpj",
		Phone: sql.NullString{Valid: false}, Status: sql.NullString{Valid: false},
		Metadata:  pqtype.NullRawMessage{Valid: false},
		CreatedAt: sql.NullTime{Valid: false}, UpdatedAt: sql.NullTime{Valid: false},
	}
	c := toDomainCustomer(row)
	if c.Phone != nil {
		t.Fatalf("expected Phone nil")
	}
	if c.Status != "" {
		t.Fatalf("expected Status empty")
	}
	if c.Metadata != nil {
		t.Fatalf("expected Metadata nil")
	}
	if !c.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt zero")
	}
	if !c.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt zero")
	}
}

func TestGetByID_NotFound_ReturnsDomainError(t *testing.T) {
	mock := &mockQuerier{getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
		return dbgen.Customer{}, sql.ErrNoRows
	}}
	svc := newTestService(mock)
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestUpdate_NotFound_ReturnsDomainError(t *testing.T) {
	mock := &mockQuerier{updateCustomerFn: func(_ context.Context, _ dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
		return dbgen.Customer{}, sql.ErrNoRows
	}}
	svc := newTestService(mock)
	_, _, err := svc.Update(context.Background(), uuid.New(), uuid.New(), domain.UpdateCustomerInput{
		Name: "Test", Email: "test@example.com", Document: "123", DocumentType: "cpf",
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestHandleWriteError_EmailUniqueViolation(t *testing.T) {
	svc := &CustomerService{}
	err := svc.handleWriteError(&pq.Error{Code: "23505", Constraint: "unique_customer_email_per_tenant"})
	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected 409, got %d", appErr.Status)
	}
}

func TestHandleWriteError_DocumentUniqueViolation(t *testing.T) {
	svc := &CustomerService{}
	err := svc.handleWriteError(&pq.Error{Code: "23505", Constraint: "unique_customer_doc_per_tenant"})
	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected 409, got %d", appErr.Status)
	}
}

func TestHandleWriteError_UnknownConstraint_ReturnsInternal(t *testing.T) {
	svc := &CustomerService{}
	err := svc.handleWriteError(&pq.Error{Code: "23505", Constraint: "other"})
	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected 500, got %d", appErr.Status)
	}
}

func TestHandleWriteError_NonPqError_ReturnsInternal(t *testing.T) {
	svc := &CustomerService{}
	err := svc.handleWriteError(errors.New("some db error"))
	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected 500, got %d", appErr.Status)
	}
}

func TestCreate_UniqueEmailViolation_ReturnsConflict(t *testing.T) {
	mock := &mockQuerier{createCustomerFn: func(_ context.Context, _ dbgen.CreateCustomerParams) (dbgen.Customer, error) {
		return dbgen.Customer{}, &pq.Error{Code: "23505", Constraint: "unique_customer_email_per_tenant"}
	}}
	svc := newTestService(mock)
	_, _, err := svc.Create(context.Background(), uuid.New(), domain.CreateCustomerInput{
		Name: "Test", Email: "dup@example.com", Document: "123", DocumentType: "cpf",
	})
	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected 409, got %d", appErr.Status)
	}
}

func TestHardDelete_Success(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	deleteCalled := false
	mock := &mockQuerier{hardDeleteCustomerFn: func(_ context.Context, arg dbgen.HardDeleteCustomerParams) error {
		if arg.ID == customerID && arg.TenantID == tenantID {
			deleteCalled = true
		}
		return nil
	}}
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
	mock := &mockQuerier{hardDeleteCustomerFn: func(_ context.Context, _ dbgen.HardDeleteCustomerParams) error {
		return errors.New("database connection failed")
	}}
	svc := newTestService(mock)
	err := svc.HardDelete(context.Background(), uuid.New(), uuid.New())
	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != 500 {
		t.Fatalf("expected 500, got %d", appErr.Status)
	}
}

// ============================================================
// Existing property-based tests
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
		tenantID := uuid.New()
		generatedID := uuid.New()
		now := time.Now().Truncate(time.Second)
		var stored dbgen.Customer
		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				stored = dbgen.Customer{
					ID: generatedID, TenantID: arg.TenantID, Name: arg.Name, Email: arg.Email,
					Phone: arg.Phone, Document: arg.Document, DocumentType: arg.DocumentType,
					Status: sql.NullString{String: "active", Valid: true}, Metadata: arg.Metadata,
					CreatedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
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
		created, _, err := svc.Create(ctx, tenantID, domain.CreateCustomerInput{
			Name: name, Email: email, Phone: phone, Document: document, DocumentType: documentType, Metadata: metadata,
		})
		if err != nil {
			t.Logf("Create failed: %v", err)
			return false
		}
		got, err := svc.GetByID(ctx, tenantID, created.ID)
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
		if got.TenantID != tenantID {
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

func TestProperty_MultiTenantIsolation(t *testing.T) {
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
		tenantA := uuid.New()
		tenantB := uuid.New()
		name := randStr("name-", 8)
		email := randStr("", 6) + "@test.com"
		document := randStr("doc-", 10)
		documentType := randStr("dt-", 4)
		customerID := uuid.New()
		now := time.Now().Truncate(time.Second)
		store := make(map[uuid.UUID][]dbgen.Customer)
		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				c := dbgen.Customer{
					ID: customerID, TenantID: arg.TenantID, Name: arg.Name, Email: arg.Email,
					Phone: arg.Phone, Document: arg.Document, DocumentType: arg.DocumentType,
					Status: sql.NullString{String: "active", Valid: true}, Metadata: arg.Metadata,
					CreatedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
				}
				store[arg.TenantID] = append(store[arg.TenantID], c)
				return c, nil
			},
			listCustomersByTenantFn: func(_ context.Context, tid uuid.UUID) ([]dbgen.Customer, error) { return store[tid], nil },
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
						return store[arg.TenantID][i], nil
					}
				}
				return dbgen.Customer{}, sql.ErrNoRows
			},
			softDeleteCustomerFn: func(_ context.Context, _ dbgen.SoftDeleteCustomerParams) error { return nil },
		}
		svc := newTestService(mock)
		ctx := context.Background()
		created, _, err := svc.Create(ctx, tenantA, domain.CreateCustomerInput{Name: name, Email: email, Document: document, DocumentType: documentType})
		if err != nil {
			return false
		}
		listB, _ := svc.List(ctx, tenantB)
		if len(listB) != 0 {
			return false
		}
		_, err = svc.GetByID(ctx, tenantB, created.ID)
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			return false
		}
		_, _, err = svc.Update(ctx, tenantB, created.ID, domain.UpdateCustomerInput{Name: "hacked", Email: "h@e.com", Document: document, DocumentType: documentType})
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			return false
		}
		err = svc.Delete(ctx, tenantB, created.ID)
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			return false
		}
		listA, _ := svc.List(ctx, tenantA)
		if len(listA) != 1 || listA[0].ID != created.ID {
			return false
		}
		updatedA, _, err := svc.Update(ctx, tenantA, created.ID, domain.UpdateCustomerInput{Name: "updated", Email: email, Document: document, DocumentType: documentType})
		if err != nil || updatedA.Name != "updated" {
			return false
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatalf("multi-tenant isolation failed: %v", err)
	}
}

func TestCreate_WithAddress_BothCreated(t *testing.T) {
	resetTxTestStore()
	tenantID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()

	txTestStore.customerRow = makeCustomerRow(customerID, tenantID, "Alice", "alice@test.com", "12345", "cpf")
	txTestStore.addressRow = makeAddressRow(addressID, tenantID, customerID)

	svc := newTestServiceWithTx()
	ctx := context.Background()

	cust, addr, err := svc.Create(ctx, tenantID, domain.CreateCustomerInput{
		Name: "Alice", Email: "alice@test.com", Document: "12345", DocumentType: "cpf",
		Address: &domain.CreateAddressInput{
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
	if !txTestStore.customerCreated {
		t.Fatal("customer was not created in tx")
	}
	if !txTestStore.addressCreated {
		t.Fatal("address was not created in tx")
	}
}

func TestCreate_WithoutAddress_OnlyCustomerCreated(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mock := &mockQuerier{
		createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
			return dbgen.Customer{
				ID: uuid.New(), TenantID: arg.TenantID, Name: arg.Name, Email: arg.Email,
				Document: arg.Document, DocumentType: arg.DocumentType,
				Status:    sql.NullString{String: "active", Valid: true},
				CreatedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
	}
	svc := newTestService(mock)
	cust, addr, err := svc.Create(context.Background(), uuid.New(), domain.CreateCustomerInput{
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
	resetTxTestStore()
	tenantID := uuid.New()
	customerID := uuid.New()

	txTestStore.customerRow = makeCustomerRow(customerID, tenantID, "Charlie", "charlie@test.com", "11111", "cpf")
	txTestStore.addressShouldFail = true

	svc := newTestServiceWithTx()
	ctx := context.Background()

	cust, addr, err := svc.Create(ctx, tenantID, domain.CreateCustomerInput{
		Name: "Charlie", Email: "charlie@test.com", Document: "11111", DocumentType: "cpf",
		Address: &domain.CreateAddressInput{
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
	if txTestStore.customerCreated && txTestStore.addressCreated {
		t.Fatal("address should not have been created")
	}
}

func TestUpdate_WithoutAddress_OnlyCustomerUpdated(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mock := &mockQuerier{
		updateCustomerFn: func(_ context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
			return dbgen.Customer{
				ID: arg.ID, TenantID: arg.TenantID, Name: arg.Name, Email: arg.Email,
				Document: arg.Document, DocumentType: arg.DocumentType,
				Status:    sql.NullString{String: "active", Valid: true},
				CreatedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
	}
	svc := newTestService(mock)
	cust, addr, err := svc.Update(context.Background(), uuid.New(), uuid.New(), domain.UpdateCustomerInput{
		Name: "Updated", Email: "updated@test.com", Document: "123", DocumentType: "cpf",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cust == nil {
		t.Fatal("expected customer")
	}
	if addr != nil {
		t.Fatalf("expected nil address, got %v", addr)
	}
	if cust.Name != "Updated" {
		t.Fatalf("expected name Updated, got %s", cust.Name)
	}
}

func TestProperty_AtomicCustomerWithAddress(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		resetTxTestStore()

		tenantID := uuid.New()
		customerID := uuid.New()
		addressID := uuid.New()

		name := rapid.StringMatching(`[A-Za-z ]{3,30}`).Draw(t, "name")
		email := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "email_local") + "@test.com"
		document := rapid.StringMatching(`[0-9]{5,15}`).Draw(t, "document")
		documentType := rapid.SampledFrom([]string{"cpf", "cnpj"}).Draw(t, "doc_type")

		postalCode := rapid.StringMatching(`[0-9]{5}-[0-9]{3}`).Draw(t, "postal_code")
		street := rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "street")
		city := rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "city")
		state := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "state")

		txTestStore.customerRow = makeCustomerRow(customerID, tenantID, name, email, document, documentType)
		txTestStore.addressRow = makeAddressRow(addressID, tenantID, customerID)

		svc := newTestServiceWithTx()
		ctx := context.Background()

		cust, addr, err := svc.Create(ctx, tenantID, domain.CreateCustomerInput{
			Name: name, Email: email, Document: document, DocumentType: documentType,
			Address: &domain.CreateAddressInput{
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

		// Both customer and address were created in the transaction
		if !txTestStore.customerCreated {
			t.Fatal("customer not created")
		}
		if !txTestStore.addressCreated {
			t.Fatal("address not created")
		}

		// Customer has correct tenant
		if cust.TenantID != tenantID {
			t.Fatalf("expected tenant %v, got %v", tenantID, cust.TenantID)
		}

		// Address has correct tenant and customer
		if addr.TenantID != tenantID {
			t.Fatalf("addr tenant mismatch")
		}
		if addr.CustomerID != customerID {
			t.Fatalf("addr customer mismatch")
		}
	})
}
func TestProperty_CreateWithoutAddress_NoAddressCreated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		generatedID := uuid.New()
		now := time.Now().Truncate(time.Second)

		name := rapid.StringMatching(`[A-Za-z ]{3,30}`).Draw(t, "name")
		email := rapid.StringMatching(`[a-z]{5,10}`).Draw(t, "email_local") + "@test.com"
		document := rapid.StringMatching(`[0-9]{5,15}`).Draw(t, "document")
		documentType := rapid.SampledFrom([]string{"cpf", "cnpj"}).Draw(t, "doc_type")

		addressListCalled := false

		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				return dbgen.Customer{
					ID: generatedID, TenantID: arg.TenantID, Name: arg.Name, Email: arg.Email,
					Phone: arg.Phone, Document: arg.Document, DocumentType: arg.DocumentType,
					Status:    sql.NullString{String: "active", Valid: true},
					CreatedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: sql.NullTime{Time: now, Valid: true},
				}, nil
			},
		}

		// Create an address service with a mock that tracks list calls
		addrMock := &addressMockQuerier{
			getCustomerByIDFn: func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				return dbgen.Customer{ID: arg.ID, TenantID: arg.TenantID}, nil
			},
			listAddressesByCustomerFn: func(_ context.Context, _ dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
				addressListCalled = true
				return []dbgen.Address{}, nil
			},
		}

		db := &database.DB{Queries: mock}
		addrDB := &database.DB{Queries: addrMock}
		addrSvc := NewAddressService(addrDB)
		svc := NewCustomerService(db, addrSvc)
		ctx := context.Background()

		cust, addr, err := svc.Create(ctx, tenantID, domain.CreateCustomerInput{
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

		// Verify no addresses exist by listing
		addresses, err := addrSvc.List(ctx, tenantID, generatedID)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if !addressListCalled {
			t.Fatal("expected list to be called")
		}
		if len(addresses) != 0 {
			t.Fatalf("expected 0 addresses, got %d", len(addresses))
		}
	})
}
