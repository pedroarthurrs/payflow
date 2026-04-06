package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/database/repository"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// mockQuerier implements dbgen.Querier for handler tests.
type mockQuerier struct {
	createAddressFn               func(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error)
	deleteAddressFn               func(ctx context.Context, arg dbgen.DeleteAddressParams) error
	getAddressByIDFn              func(ctx context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error)
	listAddressesByCustomerFn     func(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error)
	updateAddressFn               func(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error)
	getCustomerByIDFn             func(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error)
	createCustomerFn              func(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error)
	getOrganizationFn             func(ctx context.Context, id uuid.UUID) (dbgen.Organization, error)
	listCustomersByOrganizationFn func(ctx context.Context, organizationID uuid.UUID) ([]dbgen.Customer, error)
	softDeleteCustomerFn          func(ctx context.Context, arg dbgen.SoftDeleteCustomerParams) error
	updateCustomerFn              func(ctx context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error)
	hardDeleteCustomerFn          func(ctx context.Context, arg dbgen.HardDeleteCustomerParams) error
}

func (m *mockQuerier) CreateAddress(ctx context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
	if m.createAddressFn != nil {
		return m.createAddressFn(ctx, arg)
	}
	return dbgen.Address{}, nil
}
func (m *mockQuerier) DeleteAddress(ctx context.Context, arg dbgen.DeleteAddressParams) error {
	if m.deleteAddressFn != nil {
		return m.deleteAddressFn(ctx, arg)
	}
	return nil
}
func (m *mockQuerier) GetAddressByID(ctx context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
	if m.getAddressByIDFn != nil {
		return m.getAddressByIDFn(ctx, arg)
	}
	return dbgen.Address{}, sql.ErrNoRows
}
func (m *mockQuerier) ListAddressesByCustomer(ctx context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
	if m.listAddressesByCustomerFn != nil {
		return m.listAddressesByCustomerFn(ctx, arg)
	}
	return nil, nil
}
func (m *mockQuerier) UpdateAddress(ctx context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
	if m.updateAddressFn != nil {
		return m.updateAddressFn(ctx, arg)
	}
	return dbgen.Address{}, sql.ErrNoRows
}
func (m *mockQuerier) GetCustomerByID(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
	if m.getCustomerByIDFn != nil {
		return m.getCustomerByIDFn(ctx, arg)
	}
	return dbgen.Customer{}, nil
}

// Stub/delegating methods to satisfy dbgen.Querier interface.
func (m *mockQuerier) CreateCustomer(ctx context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
	if m.createCustomerFn != nil {
		return m.createCustomerFn(ctx, arg)
	}
	return dbgen.Customer{}, nil
}
func (m *mockQuerier) GetOrganization(ctx context.Context, id uuid.UUID) (dbgen.Organization, error) {
	if m.getOrganizationFn != nil {
		return m.getOrganizationFn(ctx, id)
	}
	return dbgen.Organization{}, nil
}
func (m *mockQuerier) ListCustomersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]dbgen.Customer, error) {
	if m.listCustomersByOrganizationFn != nil {
		return m.listCustomersByOrganizationFn(ctx, organizationID)
	}
	return nil, nil
}
func (m *mockQuerier) ListOrganizations(ctx context.Context) ([]dbgen.Organization, error) {
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
func (m *mockQuerier) CreateUser(ctx context.Context, arg dbgen.CreateUserParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) GetUserByEmailAndOrganization(ctx context.Context, arg dbgen.GetUserByEmailAndOrganizationParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) GetUserByID(ctx context.Context, arg dbgen.GetUserByIDParams) (dbgen.User, error) {
	return dbgen.User{}, nil
}
func (m *mockQuerier) ListUsersByOrganization(ctx context.Context, organizationID uuid.UUID) ([]dbgen.User, error) {
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
func (m *mockQuerier) CreateProduct(ctx context.Context, arg dbgen.CreateProductParams) (dbgen.Product, error) {
	return dbgen.Product{}, nil
}
func (m *mockQuerier) GetProductByID(ctx context.Context, arg dbgen.GetProductByIDParams) (dbgen.Product, error) {
	return dbgen.Product{}, nil
}
func (m *mockQuerier) ListProductsByOrganization(ctx context.Context, organizationID uuid.UUID) ([]dbgen.Product, error) {
	return nil, nil
}
func (m *mockQuerier) UpdateProduct(ctx context.Context, arg dbgen.UpdateProductParams) (dbgen.Product, error) {
	return dbgen.Product{}, nil
}
func (m *mockQuerier) SoftDeleteProduct(ctx context.Context, arg dbgen.SoftDeleteProductParams) error {
	return nil
}
func (m *mockQuerier) HardDeleteProduct(ctx context.Context, arg dbgen.HardDeleteProductParams) error {
	return nil
}

// withClaims injects JWT claims into the Gin context for testing.
func withClaims(organizationID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ClaimsKey, &token.Claims{OrganizationID: organizationID})
		c.Next()
	}
}

// setupAddressRouter creates a Gin test router with the error middleware, claims, and address handler routes.
func setupAddressRouter(mock *mockQuerier, organizationID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	customerRepo := repository.NewCustomerRepo(mock)
	addressRepo := repository.NewAddressRepo(mock)
	svc := service.NewAddressService(customerRepo, addressRepo)
	h := NewAddressHandler(svc)

	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaims(organizationID))

	r.POST("/v1/customers/:customer_id/addresses", h.Create)
	r.GET("/v1/customers/:customer_id/addresses", h.List)
	r.GET("/v1/customers/:customer_id/addresses/:id", h.GetByID)
	r.PUT("/v1/customers/:customer_id/addresses/:id", h.Update)
	r.DELETE("/v1/customers/:customer_id/addresses/:id", h.Delete)

	return r
}

// customerExistsMock returns a mock that always finds the customer.
func customerExistsMock() func(ctx context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
	return func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
		return dbgen.Customer{ID: arg.ID, OrganizationID: arg.OrganizationID}, nil
	}
}

// validAddressJSON returns a valid address JSON payload.
func validAddressJSON() string {
	return `{"postal_code":"12345","street":"Main St","city":"Springfield","state":"IL"}`
}

// basePath returns the address base path for a given customer.
func basePath(customerID string) string {
	return "/v1/customers/" + customerID + "/addresses"
}

// ============================================================
// Property-Based Tests
// ============================================================

func TestProperty_MissingRequiredFields_Returns400(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New().String()
		customerID := uuid.New().String()
		mock := &mockQuerier{getCustomerByIDFn: customerExistsMock()}
		router := setupAddressRouter(mock, organizationID)

		requiredFields := []string{"postal_code", "street", "city", "state"}
		removeCount := rapid.IntRange(1, 4).Draw(t, "remove_count")
		perm := rapid.Permutation([]int{0, 1, 2, 3}).Draw(t, "perm")
		toRemove := make(map[string]bool)
		for i := 0; i < removeCount; i++ {
			toRemove[requiredFields[perm[i]]] = true
		}

		payload := map[string]any{}
		if !toRemove["postal_code"] {
			payload["postal_code"] = rapid.StringMatching(`[0-9]{5}`).Draw(t, "pc")
		}
		if !toRemove["street"] {
			payload["street"] = rapid.StringMatching(`[A-Za-z ]{5,20}`).Draw(t, "st")
		}
		if !toRemove["city"] {
			payload["city"] = rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "ci")
		}
		if !toRemove["state"] {
			payload["state"] = rapid.StringMatching(`[A-Z]{2}`).Draw(t, "sta")
		}
		bodyBytes, _ := json.Marshal(payload)
		body := string(bodyBytes)

		// Test POST
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, basePath(customerID), strings.NewReader(body))
		r1.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w1, r1)
		if w1.Code != http.StatusBadRequest {
			t.Fatalf("POST: expected 400, got %d (removed=%v)", w1.Code, toRemove)
		}

		// Test PUT
		addrID := uuid.New().String()
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID)+"/"+addrID, strings.NewReader(body))
		r2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w2, r2)
		if w2.Code != http.StatusBadRequest {
			t.Fatalf("PUT: expected 400, got %d (removed=%v)", w2.Code, toRemove)
		}
	})
}

func TestProperty_InvalidAddressType_Returns400(t *testing.T) {
	validTypes := map[string]bool{"billing": true, "shipping": true, "home": true}
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New().String()
		customerID := uuid.New().String()
		mock := &mockQuerier{getCustomerByIDFn: customerExistsMock()}
		router := setupAddressRouter(mock, organizationID)

		invalidType := rapid.StringMatching(`[a-z]{1,20}`).Draw(t, "bad_type")
		if validTypes[invalidType] {
			return
		}

		payload := map[string]any{
			"type": invalidType, "postal_code": "12345",
			"street": "Main St", "city": "Springfield", "state": "IL",
		}
		bodyBytes, _ := json.Marshal(payload)
		body := string(bodyBytes)

		// Test POST
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, basePath(customerID), strings.NewReader(body))
		r1.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w1, r1)
		if w1.Code != http.StatusBadRequest {
			t.Fatalf("POST: expected 400, got %d (type=%q)", w1.Code, invalidType)
		}

		// Test PUT
		addrID := uuid.New().String()
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID)+"/"+addrID, strings.NewReader(body))
		r2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w2, r2)
		if w2.Code != http.StatusBadRequest {
			t.Fatalf("PUT: expected 400, got %d (type=%q)", w2.Code, invalidType)
		}
	})
}

func TestProperty_CustomerNotFound_Returns404(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New().String()
		customerID := uuid.New().String()

		mock := &mockQuerier{
			getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				return dbgen.Customer{}, sql.ErrNoRows
			},
		}
		router := setupAddressRouter(mock, organizationID)

		body := validAddressJSON()
		addrID := uuid.New().String()

		// POST
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, basePath(customerID), strings.NewReader(body))
		r1.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w1, r1)
		if w1.Code != http.StatusNotFound {
			t.Fatalf("POST: expected 404, got %d", w1.Code)
		}
		assertErrorMessage(t, w1, "customer not found")

		// LIST
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID), http.NoBody)
		router.ServeHTTP(w2, r2)
		if w2.Code != http.StatusNotFound {
			t.Fatalf("LIST: expected 404, got %d", w2.Code)
		}
		assertErrorMessage(t, w2, "customer not found")

		// GET
		w3 := httptest.NewRecorder()
		r3 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID)+"/"+addrID, http.NoBody)
		router.ServeHTTP(w3, r3)
		if w3.Code != http.StatusNotFound {
			t.Fatalf("GET: expected 404, got %d", w3.Code)
		}
		assertErrorMessage(t, w3, "customer not found")

		// PUT
		w4 := httptest.NewRecorder()
		r4 := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID)+"/"+addrID, strings.NewReader(body))
		r4.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w4, r4)
		if w4.Code != http.StatusNotFound {
			t.Fatalf("PUT: expected 404, got %d", w4.Code)
		}
		assertErrorMessage(t, w4, "customer not found")

		// DELETE
		w5 := httptest.NewRecorder()
		r5 := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, basePath(customerID)+"/"+addrID, http.NoBody)
		router.ServeHTTP(w5, r5)
		if w5.Code != http.StatusNotFound {
			t.Fatalf("DELETE: expected 404, got %d", w5.Code)
		}
		assertErrorMessage(t, w5, "customer not found")
	})
}

func assertErrorMessage(t *rapid.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v (body=%s)", err, w.Body.String())
	}
	if resp["error"] != expected {
		t.Fatalf("expected error %q, got %q", expected, resp["error"])
	}
}

func TestProperty_ValidationErrorResponseFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New().String()
		customerID := uuid.New().String()
		mock := &mockQuerier{getCustomerByIDFn: customerExistsMock()}
		router := setupAddressRouter(mock, organizationID)

		category := rapid.IntRange(0, 3).Draw(t, "category")
		payload := map[string]any{}
		switch category {
		case 0:
			// Empty body — all required fields missing
		case 1:
			payload["postal_code"] = rapid.StringMatching(`[0-9]{5}`).Draw(t, "pc")
		case 2:
			badType := rapid.StringMatching(`[a-z]{4,10}`).Draw(t, "bad_type")
			if badType == "billing" || badType == "shipping" || badType == "home" {
				badType = "invalid"
			}
			payload["type"] = badType
			payload["postal_code"] = "12345"
			payload["street"] = "St"
			payload["city"] = "City"
			payload["state"] = "XX"
		case 3:
			payload["postal_code"] = "12345"
			payload["street"] = "St"
			payload["city"] = "City"
			payload["state"] = rapid.StringMatching(`[A-Z]{3,5}`).Draw(t, "bad_state")
		}

		bodyBytes, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, basePath(customerID), strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d (category=%d, body=%s)", w.Code, category, string(bodyBytes))
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		errorsRaw, ok := resp["errors"]
		if !ok {
			t.Fatalf("response missing 'errors' key")
		}
		var errorsArr []map[string]any
		if err := json.Unmarshal(errorsRaw, &errorsArr); err != nil {
			t.Fatalf("'errors' is not a valid array: %v", err)
		}
		if len(errorsArr) == 0 {
			t.Fatalf("'errors' array is empty (category=%d)", category)
		}
		for i, errObj := range errorsArr {
			fieldVal, hasField := errObj["field"]
			_, hasRejected := errObj["rejected_value"]
			msgVal, hasMessage := errObj["message"]
			if !hasField {
				t.Fatalf("error[%d] missing 'field'", i)
			}
			if !hasRejected {
				t.Fatalf("error[%d] missing 'rejected_value'", i)
			}
			if !hasMessage {
				t.Fatalf("error[%d] missing 'message'", i)
			}
			if s, ok := fieldVal.(string); !ok || s == "" {
				t.Fatalf("error[%d] 'field' is not a non-empty string: %v", i, fieldVal)
			}
			if s, ok := msgVal.(string); !ok || s == "" {
				t.Fatalf("error[%d] 'message' is not a non-empty string: %v", i, msgVal)
			}
		}
	})
}

// ============================================================
// Unit Tests
// ============================================================

func TestCreate_ValidPayload_Returns201(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addressID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		createAddressFn: func(_ context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
			return dbgen.Address{
				ID: addressID, OrganizationID: organizationID, CustomerID: customerID,
				Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
				PostalCode: arg.PostalCode, Street: arg.Street, City: arg.City, State: arg.State,
				Country:   sql.NullString{String: "BR", Valid: true},
				IsMain:    sql.NullBool{Bool: true, Valid: true},
				CreatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
	}
	router := setupAddressRouter(mock, organizationID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, basePath(customerID.String()), strings.NewReader(validAddressJSON()))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}

	var resp dto.AddressResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.ID != addressID {
		t.Fatalf("expected ID %v, got %v", addressID, resp.ID)
	}
	if resp.CustomerID != customerID {
		t.Fatalf("expected CustomerID %v, got %v", customerID, resp.CustomerID)
	}
}

func TestCreate_InvalidPayload_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	mock := &mockQuerier{getCustomerByIDFn: customerExistsMock()}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, basePath(customerID), strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string][]apierr.ValidationFieldError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp["errors"]) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestList_Empty_Returns200(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		listAddressesByCustomerFn: func(_ context.Context, _ dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
			return []dbgen.Address{}, nil
		},
	}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []*dto.AddressResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %d items", len(resp))
	}
}

func TestList_WithAddresses_Returns200(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		listAddressesByCustomerFn: func(_ context.Context, _ dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
			return []dbgen.Address{
				{ID: uuid.New(), OrganizationID: organizationID, CustomerID: customerID,
					Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
					PostalCode: "11111", Street: "St A", City: "City A", State: "AA",
					Country:   sql.NullString{String: "BR", Valid: true},
					IsMain:    sql.NullBool{Bool: true, Valid: true},
					CreatedAt: sql.NullTime{Time: now, Valid: true}},
				{ID: uuid.New(), OrganizationID: organizationID, CustomerID: customerID,
					Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeShipping, Valid: true},
					PostalCode: "22222", Street: "St B", City: "City B", State: "BB",
					Country:   sql.NullString{String: "US", Valid: true},
					IsMain:    sql.NullBool{Bool: false, Valid: true},
					CreatedAt: sql.NullTime{Time: now.Add(-time.Hour), Valid: true}},
			}, nil
		},
	}
	router := setupAddressRouter(mock, organizationID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String()), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []*dto.AddressResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(resp))
	}
}

func TestGetByID_Found_Returns200(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addrID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{
				ID: addrID, OrganizationID: organizationID, CustomerID: customerID,
				Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
				PostalCode: "12345", Street: "Main St", City: "City", State: "ST",
				Country:   sql.NullString{String: "BR", Valid: true},
				IsMain:    sql.NullBool{Bool: true, Valid: true},
				CreatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
	}
	router := setupAddressRouter(mock, organizationID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp dto.AddressResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.ID != addrID {
		t.Fatalf("expected ID %v, got %v", addrID, resp.ID)
	}
}

func TestGetByID_NotFound_Returns404(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	addrID := uuid.New().String()

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{}, sql.ErrNoRows
		},
	}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID)+"/"+addrID, http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "address not found" {
		t.Fatalf("expected 'address not found', got %q", resp["error"])
	}
}

func TestGetByID_InvalidUUID_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	mock := &mockQuerier{getCustomerByIDFn: customerExistsMock()}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID)+"/not-a-uuid", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "invalid address id" {
		t.Fatalf("expected 'invalid address id', got %q", resp["error"])
	}
}

func TestUpdate_ValidPayload_Returns200(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addrID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		updateAddressFn: func(_ context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
			return dbgen.Address{
				ID: addrID, OrganizationID: organizationID, CustomerID: customerID,
				Type: arg.Type, PostalCode: arg.PostalCode, Street: arg.Street,
				City: arg.City, State: arg.State,
				Country: arg.Country, IsMain: arg.IsMain,
				CreatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
	}
	router := setupAddressRouter(mock, organizationID.String())

	body := `{"postal_code":"99999","street":"Updated St","city":"New City","state":"NC"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID.String())+"/"+addrID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestUpdate_InvalidPayload_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	addrID := uuid.New().String()
	mock := &mockQuerier{getCustomerByIDFn: customerExistsMock()}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID)+"/"+addrID, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_NotFound_Returns404(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	addrID := uuid.New().String()

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		updateAddressFn: func(_ context.Context, _ dbgen.UpdateAddressParams) (dbgen.Address, error) {
			return dbgen.Address{}, sql.ErrNoRows
		},
	}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID)+"/"+addrID, strings.NewReader(validAddressJSON()))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "address not found" {
		t.Fatalf("expected 'address not found', got %q", resp["error"])
	}
}

func TestDelete_Found_Returns204(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	addrID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{
				ID: addrID, OrganizationID: organizationID, CustomerID: customerID,
				PostalCode: "12345", Street: "St", City: "C", State: "ST",
				CreatedAt: sql.NullTime{Time: now, Valid: true},
			}, nil
		},
		deleteAddressFn: func(_ context.Context, _ dbgen.DeleteAddressParams) error {
			return nil
		},
	}
	router := setupAddressRouter(mock, organizationID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestDelete_NotFound_Returns404(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	addrID := uuid.New().String()

	mock := &mockQuerier{
		getCustomerByIDFn: customerExistsMock(),
		getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
			return dbgen.Address{}, sql.ErrNoRows
		},
	}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, basePath(customerID)+"/"+addrID, http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "address not found" {
		t.Fatalf("expected 'address not found', got %q", resp["error"])
	}
}

func TestDelete_InvalidUUID_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()
	mock := &mockQuerier{getCustomerByIDFn: customerExistsMock()}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, basePath(customerID)+"/not-a-uuid", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "invalid address id" {
		t.Fatalf("expected 'invalid address id', got %q", resp["error"])
	}
}

func TestCreate_CustomerNotFound_Returns404(t *testing.T) {
	organizationID := uuid.New().String()
	customerID := uuid.New().String()

	mock := &mockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}
	router := setupAddressRouter(mock, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, basePath(customerID), strings.NewReader(validAddressJSON()))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "customer not found" {
		t.Fatalf("expected 'customer not found', got %q", resp["error"])
	}
}

// ============================================================
// Property-Based Tests
// ============================================================

func TestProperty_ListingOrderedByCreatedAtDesc(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		customerID := uuid.New()

		n := rapid.IntRange(1, 10).Draw(t, "n")
		baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		addresses := make([]dbgen.Address, n)
		for i := 0; i < n; i++ {
			addresses[i] = dbgen.Address{
				ID: uuid.New(), OrganizationID: organizationID, CustomerID: customerID,
				Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
				PostalCode: rapid.StringMatching(`[0-9]{5}`).Draw(t, "pc"),
				Street:     rapid.StringMatching(`[A-Za-z ]{5,20}`).Draw(t, "st"),
				City:       rapid.StringMatching(`[A-Za-z]{3,15}`).Draw(t, "ci"),
				State:      rapid.StringMatching(`[A-Z]{2}`).Draw(t, "sta"),
				Country:    sql.NullString{String: "BR", Valid: true},
				IsMain:     sql.NullBool{Bool: i == 0, Valid: true},
				CreatedAt:  sql.NullTime{Time: baseTime.Add(time.Duration(i) * time.Hour), Valid: true},
			}
		}

		// Shuffle to simulate DB returning in DESC order after we sort
		shuffled := make([]dbgen.Address, n)
		perm := rapid.Permutation(make([]int, n)).Draw(t, "perm")
		for i, p := range perm {
			shuffled[i] = addresses[p]
		}

		// Sort DESC by created_at (mock simulates DB ORDER BY)
		sorted := make([]dbgen.Address, n)
		copy(sorted, addresses)
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		}

		mock := &mockQuerier{
			getCustomerByIDFn: customerExistsMock(),
			listAddressesByCustomerFn: func(_ context.Context, _ dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
				return sorted, nil
			},
		}
		router := setupAddressRouter(mock, organizationID.String())

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String()), http.NoBody)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp []*dto.AddressResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(resp) != n {
			t.Fatalf("expected %d addresses, got %d", n, len(resp))
		}

		// Verify DESC order
		for i := 1; i < len(resp); i++ {
			if resp[i].CreatedAt.After(resp[i-1].CreatedAt) {
				t.Fatalf("addresses not in DESC order: index %d (%v) > index %d (%v)",
					i, resp[i].CreatedAt, i-1, resp[i-1].CreatedAt)
			}
		}
	})
}

func TestProperty_UpdateReflectsNewValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		customerID := uuid.New()
		addrID := uuid.New()
		now := time.Now().Truncate(time.Second)

		newPC := rapid.StringMatching(`[0-9]{5}`).Draw(t, "pc")
		newStreet := rapid.StringMatching(`[A-Za-z ]{5,30}`).Draw(t, "street")
		newCity := rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "city")
		newState := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "state")
		types := []string{"billing", "shipping", "home"}
		newType := types[rapid.IntRange(0, 2).Draw(t, "type_idx")]

		var updatedRow dbgen.Address

		mock := &mockQuerier{
			getCustomerByIDFn: customerExistsMock(),
			updateAddressFn: func(_ context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
				updatedRow = dbgen.Address{
					ID: addrID, OrganizationID: organizationID, CustomerID: customerID,
					Type:       arg.Type,
					PostalCode: arg.PostalCode, Street: arg.Street,
					City: arg.City, State: arg.State,
					Country:   arg.Country,
					IsMain:    arg.IsMain,
					CreatedAt: sql.NullTime{Time: now, Valid: true},
				}
				return updatedRow, nil
			},
			getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
				return updatedRow, nil
			},
		}
		router := setupAddressRouter(mock, organizationID.String())

		payload := map[string]any{
			"type": newType, "postal_code": newPC,
			"street": newStreet, "city": newCity, "state": newState,
		}
		bodyBytes, _ := json.Marshal(payload)

		// PUT to update
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID.String())+"/"+addrID.String(), strings.NewReader(string(bodyBytes)))
		r1.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w1, r1)

		if w1.Code != http.StatusOK {
			t.Fatalf("PUT: expected 200, got %d (body=%s)", w1.Code, w1.Body.String())
		}

		// GET to verify
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
		router.ServeHTTP(w2, r2)

		if w2.Code != http.StatusOK {
			t.Fatalf("GET: expected 200, got %d", w2.Code)
		}

		var resp dto.AddressResponse
		if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse GET response: %v", err)
		}

		if resp.PostalCode != newPC {
			t.Fatalf("postal_code: expected %q, got %q", newPC, resp.PostalCode)
		}
		if resp.Street != newStreet {
			t.Fatalf("street: expected %q, got %q", newStreet, resp.Street)
		}
		if resp.City != newCity {
			t.Fatalf("city: expected %q, got %q", newCity, resp.City)
		}
		if resp.State != newState {
			t.Fatalf("state: expected %q, got %q", newState, resp.State)
		}
		if resp.Type != newType {
			t.Fatalf("type: expected %q, got %q", newType, resp.Type)
		}
	})
}

func TestProperty_DeleteRemovesAddress(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationID := uuid.New()
		customerID := uuid.New()
		addrID := uuid.New()
		now := time.Now().Truncate(time.Second)

		deleted := false

		mock := &mockQuerier{
			getCustomerByIDFn: customerExistsMock(),
			getAddressByIDFn: func(_ context.Context, _ dbgen.GetAddressByIDParams) (dbgen.Address, error) {
				if deleted {
					return dbgen.Address{}, sql.ErrNoRows
				}
				return dbgen.Address{
					ID: addrID, OrganizationID: organizationID, CustomerID: customerID,
					Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
					PostalCode: "12345", Street: "St", City: "City", State: "ST",
					Country:   sql.NullString{String: "BR", Valid: true},
					IsMain:    sql.NullBool{Bool: true, Valid: true},
					CreatedAt: sql.NullTime{Time: now, Valid: true},
				}, nil
			},
			deleteAddressFn: func(_ context.Context, _ dbgen.DeleteAddressParams) error {
				deleted = true
				return nil
			},
		}
		router := setupAddressRouter(mock, organizationID.String())

		// Verify address exists first
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
		router.ServeHTTP(w1, r1)
		if w1.Code != http.StatusOK {
			t.Fatalf("GET before delete: expected 200, got %d", w1.Code)
		}

		// DELETE
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
		router.ServeHTTP(w2, r2)
		if w2.Code != http.StatusNoContent {
			t.Fatalf("DELETE: expected 204, got %d", w2.Code)
		}

		// GET after delete should return 404
		w3 := httptest.NewRecorder()
		r3 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
		router.ServeHTTP(w3, r3)
		if w3.Code != http.StatusNotFound {
			t.Fatalf("GET after delete: expected 404, got %d", w3.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(w3.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["error"] != "address not found" {
			t.Fatalf("expected 'address not found', got %q", resp["error"])
		}
	})
}

func TestProperty_MultiOrganizationIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		organizationA := uuid.New()
		organizationB := uuid.New()
		customerID := uuid.New()
		addrID := uuid.New()
		now := time.Now().Truncate(time.Second)

		mock := &mockQuerier{
			getCustomerByIDFn: func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				// Customer only exists for organization A
				if arg.OrganizationID == organizationA {
					return dbgen.Customer{ID: arg.ID, OrganizationID: organizationA}, nil
				}
				return dbgen.Customer{}, sql.ErrNoRows
			},
			createAddressFn: func(_ context.Context, arg dbgen.CreateAddressParams) (dbgen.Address, error) {
				return dbgen.Address{
					ID: addrID, OrganizationID: arg.OrganizationID, CustomerID: arg.CustomerID,
					Type:       arg.Type,
					PostalCode: arg.PostalCode, Street: arg.Street, City: arg.City, State: arg.State,
					Country:   arg.Country,
					IsMain:    arg.IsMain,
					CreatedAt: sql.NullTime{Time: now, Valid: true},
				}, nil
			},
			getAddressByIDFn: func(_ context.Context, arg dbgen.GetAddressByIDParams) (dbgen.Address, error) {
				// Address only visible to organization A
				if arg.OrganizationID == organizationA {
					return dbgen.Address{
						ID: addrID, OrganizationID: organizationA, CustomerID: customerID,
						Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
						PostalCode: "12345", Street: "St", City: "City", State: "ST",
						Country:   sql.NullString{String: "BR", Valid: true},
						IsMain:    sql.NullBool{Bool: true, Valid: true},
						CreatedAt: sql.NullTime{Time: now, Valid: true},
					}, nil
				}
				return dbgen.Address{}, sql.ErrNoRows
			},
			listAddressesByCustomerFn: func(_ context.Context, arg dbgen.ListAddressesByCustomerParams) ([]dbgen.Address, error) {
				if arg.OrganizationID == organizationA {
					return []dbgen.Address{{
						ID: addrID, OrganizationID: organizationA, CustomerID: customerID,
						Type:       dbgen.NullAddressType{AddressType: dbgen.AddressTypeBilling, Valid: true},
						PostalCode: "12345", Street: "St", City: "City", State: "ST",
						Country:   sql.NullString{String: "BR", Valid: true},
						IsMain:    sql.NullBool{Bool: true, Valid: true},
						CreatedAt: sql.NullTime{Time: now, Valid: true},
					}}, nil
				}
				return []dbgen.Address{}, nil
			},
			updateAddressFn: func(_ context.Context, arg dbgen.UpdateAddressParams) (dbgen.Address, error) {
				if arg.OrganizationID == organizationA {
					return dbgen.Address{
						ID: addrID, OrganizationID: organizationA, CustomerID: customerID,
						Type: arg.Type, PostalCode: arg.PostalCode, Street: arg.Street,
						City: arg.City, State: arg.State, Country: arg.Country, IsMain: arg.IsMain,
						CreatedAt: sql.NullTime{Time: now, Valid: true},
					}, nil
				}
				return dbgen.Address{}, sql.ErrNoRows
			},
			deleteAddressFn: func(_ context.Context, arg dbgen.DeleteAddressParams) error {
				return nil
			},
		}

		// Organization A can access the address
		routerA := setupAddressRouter(mock, organizationA.String())
		wA := httptest.NewRecorder()
		rA := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
		routerA.ServeHTTP(wA, rA)
		if wA.Code != http.StatusOK {
			t.Fatalf("Organization A GET: expected 200, got %d", wA.Code)
		}

		// Organization B gets 404 for customer (customer doesn't exist for organization B)
		routerB := setupAddressRouter(mock, organizationB.String())

		// GET by ID
		wB1 := httptest.NewRecorder()
		rB1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
		routerB.ServeHTTP(wB1, rB1)
		if wB1.Code != http.StatusNotFound {
			t.Fatalf("Organization B GET: expected 404, got %d", wB1.Code)
		}

		// LIST
		wB2 := httptest.NewRecorder()
		rB2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, basePath(customerID.String()), http.NoBody)
		routerB.ServeHTTP(wB2, rB2)
		if wB2.Code != http.StatusNotFound {
			t.Fatalf("Organization B LIST: expected 404, got %d", wB2.Code)
		}

		// PUT
		wB3 := httptest.NewRecorder()
		rB3 := httptest.NewRequestWithContext(context.Background(), http.MethodPut, basePath(customerID.String())+"/"+addrID.String(), strings.NewReader(validAddressJSON()))
		rB3.Header.Set("Content-Type", "application/json")
		routerB.ServeHTTP(wB3, rB3)
		if wB3.Code != http.StatusNotFound {
			t.Fatalf("Organization B PUT: expected 404, got %d", wB3.Code)
		}

		// DELETE
		wB4 := httptest.NewRecorder()
		rB4 := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, basePath(customerID.String())+"/"+addrID.String(), http.NoBody)
		routerB.ServeHTTP(wB4, rB4)
		if wB4.Code != http.StatusNotFound {
			t.Fatalf("Organization B DELETE: expected 404, got %d", wB4.Code)
		}
	})
}
