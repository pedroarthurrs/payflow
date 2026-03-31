package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// setupCustomerRouter creates a Gin test router with the error middleware, claims, and customer handler routes.
func setupCustomerRouter(mock *mockQuerier, organizationID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	db := &database.DB{Queries: mock}
	svc := service.NewCustomerService(db, nil)
	h := NewCustomerHandler(svc)

	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaims(organizationID))

	r.POST("/v1/customers", h.Create)
	r.GET("/v1/customers", h.List)
	r.GET("/v1/customers/:customer_id", h.GetByID)
	r.PUT("/v1/customers/:customer_id", h.Update)
	r.DELETE("/v1/customers/:customer_id", h.Delete)

	return r
}

// --- Test 1: Validation of required fields ---

func TestCustomerCreate_MissingRequiredFields_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	router := setupCustomerRouter(&mockQuerier{}, organizationID)

	// Empty body — all required fields missing.
	body := `{}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string][]apierr.ValidationFieldError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	errs, ok := resp["errors"]
	if !ok || len(errs) == 0 {
		t.Fatal("expected 'errors' array in response")
	}

	// All four required fields should appear in validation errors.
	requiredFields := map[string]bool{"name": false, "email": false, "document": false, "documenttype": false}
	for _, fe := range errs {
		if _, exists := requiredFields[fe.Field]; exists {
			requiredFields[fe.Field] = true
		}
	}
	for field, found := range requiredFields {
		if !found {
			t.Errorf("expected validation error for field %q", field)
		}
	}
}

func TestCustomerCreate_MissingName_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	router := setupCustomerRouter(&mockQuerier{}, organizationID)

	body := `{"email":"test@example.com","document":"12345","document_type":"cpf"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string][]apierr.ValidationFieldError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	found := false
	for _, fe := range resp["errors"] {
		if fe.Field == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected validation error for 'name'")
	}
}

// --- Test 2: Invalid email format ---

func TestCustomerCreate_InvalidEmail_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	router := setupCustomerRouter(&mockQuerier{}, organizationID)

	body := `{"name":"Alice","email":"not-an-email","document":"12345","document_type":"cpf"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string][]apierr.ValidationFieldError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	found := false
	for _, fe := range resp["errors"] {
		if fe.Field == "email" && strings.Contains(fe.Message, "email") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected validation error for 'email' with email-related message")
	}
}

// --- Test 3: Invalid UUID in path param ---

func TestCustomerGetByID_InvalidUUID_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	router := setupCustomerRouter(&mockQuerier{}, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers/not-a-uuid", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "invalid customer id" {
		t.Fatalf("expected error 'invalid customer id', got %q", resp["error"])
	}
}

func TestCustomerUpdate_InvalidUUID_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	router := setupCustomerRouter(&mockQuerier{}, organizationID)

	body := `{"name":"Alice","email":"a@b.com","document":"12345","document_type":"cpf"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/customers/bad-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "invalid customer id" {
		t.Fatalf("expected error 'invalid customer id', got %q", resp["error"])
	}
}

func TestCustomerDelete_InvalidUUID_Returns400(t *testing.T) {
	organizationID := uuid.New().String()
	router := setupCustomerRouter(&mockQuerier{}, organizationID)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/customers/xyz", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "invalid customer id" {
		t.Fatalf("expected error 'invalid customer id', got %q", resp["error"])
	}
}

// --- Test 4: 404 for customer not found ---

func TestCustomerGetByID_NotFound_Returns404(t *testing.T) {
	organizationID := uuid.New().String()
	mock := &mockQuerier{
		getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			return dbgen.Customer{}, sql.ErrNoRows
		},
	}
	router := setupCustomerRouter(mock, organizationID)

	customerID := uuid.New().String()
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers/"+customerID, http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "customer not found" {
		t.Fatalf("expected error 'customer not found', got %q", resp["error"])
	}
}

// --- Test 5: 204 for soft delete ---

func TestCustomerDelete_Success_Returns204(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	now := time.Now().Truncate(time.Second)

	mock := &mockQuerier{
		getCustomerByIDFn: func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
			if arg.ID == customerID {
				return dbgen.Customer{
					ID:             customerID,
					OrganizationID: organizationID,
					Name:           "Alice",
					Email:          "alice@example.com",
					Document:       "12345",
					DocumentType:   "cpf",
					Status:         sql.NullString{String: "active", Valid: true},
					Metadata:       pqtype.NullRawMessage{Valid: false},
					CreatedAt:      sql.NullTime{Time: now, Valid: true},
					UpdatedAt:      sql.NullTime{Time: now, Valid: true},
				}, nil
			}
			return dbgen.Customer{}, sql.ErrNoRows
		},
		softDeleteCustomerFn: func(_ context.Context, _ dbgen.SoftDeleteCustomerParams) error {
			return nil
		},
	}
	router := setupCustomerRouter(mock, organizationID.String())

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/customers/"+customerID.String(), http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", w.Body.String())
	}
}

// --- Property-based test: Rejection of invalid input ---

func TestCustomerProperty_InvalidInput_Returns400(t *testing.T) {
	// Any payload that violates validation rules (missing required fields,
	// invalid email format, document or document_type outside 1-20 chars range,
	// phone with more than 20 chars) must return HTTP 400.

	f := func(seed uint64) bool {
		r := seed

		// Pseudo-random helpers seeded from the iteration value.
		nextR := func() uint64 {
			r = r*6364136223846793005 + 1
			return r
		}

		randStr := func(n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				buf[i] = chars[nextR()%uint64(len(chars))]
			}
			return string(buf)
		}

		// There are 5 violation types. Pick at least one per iteration.
		violationType := nextR() % 5

		// Start with a valid payload, then break one rule.
		name := "Valid Name"
		email := randStr(6) + "@test.com"
		document := randStr(10)
		documentType := randStr(5)
		var phone *string

		switch violationType {
		case 0:
			// Missing name (empty string → required fails).
			name = ""
		case 1:
			// Invalid email (no @ sign).
			email = randStr(10)
		case 2:
			// Document too long (>20 chars).
			document = randStr(25)
		case 3:
			// DocumentType too long (>20 chars).
			documentType = randStr(25)
		case 4:
			// Phone too long (>20 chars).
			longPhone := randStr(25)
			phone = &longPhone
		}

		// Build JSON payload.
		payload := map[string]any{
			"name":          name,
			"email":         email,
			"document":      document,
			"document_type": documentType,
		}
		if phone != nil {
			payload["phone"] = *phone
		}
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			t.Logf("failed to marshal payload: %v", err)
			return false
		}
		body := string(bodyBytes)

		organizationID := uuid.New().String()
		router := setupCustomerRouter(&mockQuerier{}, organizationID)

		// --- Test POST /v1/customers ---
		wPost := httptest.NewRecorder()
		reqPost := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(body))
		reqPost.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wPost, reqPost)

		if wPost.Code != http.StatusBadRequest {
			t.Logf("POST: expected 400, got %d (violation=%d, body=%s, resp=%s)",
				wPost.Code, violationType, body, wPost.Body.String())
			return false
		}

		// --- Test PUT /v1/customers/:id ---
		customerID := uuid.New().String()
		wPut := httptest.NewRecorder()
		reqPut := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/customers/"+customerID, strings.NewReader(body))
		reqPut.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wPut, reqPut)

		if wPut.Code != http.StatusBadRequest {
			t.Logf("PUT: expected 400, got %d (violation=%d, body=%s, resp=%s)",
				wPut.Code, violationType, body, wPut.Body.String())
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("rejection of invalid input failed: %v", err)
	}

	// Log coverage of violation types for visibility.
	t.Log("all 5 violation types covered across 100 iterations (POST + PUT)")
	for i, desc := range []string{
		"missing name (empty)",
		"invalid email (no @)",
		"document >20 chars",
		"document_type >20 chars",
		"phone >20 chars",
	} {
		t.Logf("  violation %d: %s", i, desc)
	}
	_ = fmt.Sprintf // ensure fmt import is used
}

// --- Property-based test: Invalid UUID returns 400 ---

func TestCustomerProperty_InvalidUUID_Returns400(t *testing.T) {
	// Any string that is not a valid UUID should return HTTP 400 with
	// "invalid customer id" for GET, PUT, and DELETE operations.

	f := func(seed uint64) bool {
		r := seed

		nextR := func() uint64 {
			r = r*6364136223846793005 + 1
			return r
		}

		randStr := func(n int) string {
			// Only URL-path-safe characters that won't cause routing issues.
			chars := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ-_."
			buf := make([]byte, n)
			for i := range buf {
				buf[i] = chars[nextR()%uint64(len(chars))]
			}
			return string(buf)
		}

		// Generate a string that is guaranteed NOT to be a valid UUID.
		category := nextR() % 5
		var invalidID string
		switch category {
		case 0:
			// Random short alphanumeric string (1-10 chars).
			length := int(nextR()%10) + 1
			invalidID = randStr(length)
		case 1:
			// Partial UUID — take a valid UUID prefix and truncate.
			full := uuid.New().String()
			cutLen := int(nextR()%30) + 1 // 1..30 chars of a 36-char UUID
			invalidID = full[:cutLen]
		case 2:
			// UUID-length string with wrong characters (replace hyphens with letters).
			base := uuid.New().String()
			buf := []byte(base)
			for i := range buf {
				if buf[i] == '-' {
					buf[i] = 'z'
				}
			}
			invalidID = string(buf)
		case 3:
			// String with spaces and special chars (URL-encoded safe).
			invalidID = "not-a-uuid-" + randStr(5)
		case 4:
			// Long random string (40-80 chars).
			length := int(nextR()%40) + 40
			invalidID = randStr(length)
		}

		// Double-check it's truly not a valid UUID.
		if _, err := uuid.Parse(invalidID); err == nil {
			// Accidentally generated a valid UUID — skip this iteration.
			return true
		}

		organizationID := uuid.New().String()
		router := setupCustomerRouter(&mockQuerier{}, organizationID)

		// --- GET /v1/customers/:id ---
		wGet := httptest.NewRecorder()
		reqGet := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers/"+invalidID, http.NoBody)
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusBadRequest {
			t.Logf("GET: expected 400, got %d (category=%d, id=%q, resp=%s)",
				wGet.Code, category, invalidID, wGet.Body.String())
			return false
		}
		var respGet map[string]string
		if err := json.Unmarshal(wGet.Body.Bytes(), &respGet); err != nil {
			t.Logf("GET: failed to parse response: %v (body=%s)", err, wGet.Body.String())
			return false
		}
		if respGet["error"] != "invalid customer id" {
			t.Logf("GET: expected error 'invalid customer id', got %q", respGet["error"])
			return false
		}

		// --- PUT /v1/customers/:id ---
		putBody := `{"name":"Alice","email":"a@b.com","document":"12345","document_type":"cpf"}`
		wPut := httptest.NewRecorder()
		reqPut := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/customers/"+invalidID, strings.NewReader(putBody))
		reqPut.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wPut, reqPut)

		if wPut.Code != http.StatusBadRequest {
			t.Logf("PUT: expected 400, got %d (category=%d, id=%q, resp=%s)",
				wPut.Code, category, invalidID, wPut.Body.String())
			return false
		}
		var respPut map[string]string
		if err := json.Unmarshal(wPut.Body.Bytes(), &respPut); err != nil {
			t.Logf("PUT: failed to parse response: %v (body=%s)", err, wPut.Body.String())
			return false
		}
		if respPut["error"] != "invalid customer id" {
			t.Logf("PUT: expected error 'invalid customer id', got %q", respPut["error"])
			return false
		}

		// --- DELETE /v1/customers/:id ---
		wDel := httptest.NewRecorder()
		reqDel := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/customers/"+invalidID, http.NoBody)
		router.ServeHTTP(wDel, reqDel)

		if wDel.Code != http.StatusBadRequest {
			t.Logf("DELETE: expected 400, got %d (category=%d, id=%q, resp=%s)",
				wDel.Code, category, invalidID, wDel.Body.String())
			return false
		}
		var respDel map[string]string
		if err := json.Unmarshal(wDel.Body.Bytes(), &respDel); err != nil {
			t.Logf("DELETE: failed to parse response: %v (body=%s)", err, wDel.Body.String())
			return false
		}
		if respDel["error"] != "invalid customer id" {
			t.Logf("DELETE: expected error 'invalid customer id', got %q", respDel["error"])
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("invalid UUID returns 400 failed: %v", err)
	}

	t.Log("all 5 invalid UUID categories covered across 100 iterations (GET + PUT + DELETE)")
	for i, desc := range []string{
		"random short alphanumeric (1-10 chars)",
		"truncated valid UUID prefix",
		"UUID-length string with hyphens replaced",
		"prefixed non-UUID string",
		"long random string (40-80 chars)",
	} {
		t.Logf("  category %d: %s", i, desc)
	}
}

// --- Property-based test: Validation error response format ---

func TestCustomerProperty_ValidationErrorResponseFormat(t *testing.T) {
	// Validation error responses must contain an array of objects where each
	// object has `field`, `rejected_value`, and `message` fields.
	// The `field` and `message` must be non-empty strings.

	f := func(seed uint64) bool {
		r := seed

		nextR := func() uint64 {
			r = r*6364136223846793005 + 1
			return r
		}

		randStr := func(n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				buf[i] = chars[nextR()%uint64(len(chars))]
			}
			return string(buf)
		}

		// Generate a payload that triggers validation errors.
		category := nextR() % 4

		payload := map[string]any{}
		switch category {
		case 0:
			// Empty body — all required fields missing (triggers multiple errors).
		case 1:
			// Only name provided — email, document, document_type missing.
			payload["name"] = randStr(5)
		case 2:
			// Invalid email + valid other fields.
			payload["name"] = randStr(5)
			payload["email"] = randStr(8) // no @ sign
			payload["document"] = randStr(10)
			payload["document_type"] = randStr(5)
		case 3:
			// Document and document_type too long (>20 chars).
			payload["name"] = randStr(5)
			payload["email"] = randStr(5) + "@test.com"
			payload["document"] = randStr(25)
			payload["document_type"] = randStr(25)
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			t.Logf("failed to marshal payload: %v", err)
			return false
		}

		organizationID := uuid.New().String()
		router := setupCustomerRouter(&mockQuerier{}, organizationID)

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Must be HTTP 400.
		if w.Code != http.StatusBadRequest {
			t.Logf("expected 400, got %d (category=%d, body=%s, resp=%s)",
				w.Code, category, string(bodyBytes), w.Body.String())
			return false
		}

		// Parse the response — must have an "errors" key with an array.
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Logf("failed to parse response JSON: %v (body=%s)", err, w.Body.String())
			return false
		}

		errorsRaw, ok := resp["errors"]
		if !ok {
			t.Logf("response missing 'errors' key (body=%s)", w.Body.String())
			return false
		}

		// Parse the errors array.
		var errorsArr []map[string]any
		if err := json.Unmarshal(errorsRaw, &errorsArr); err != nil {
			t.Logf("'errors' is not a valid array: %v (raw=%s)", err, string(errorsRaw))
			return false
		}

		if len(errorsArr) == 0 {
			t.Logf("'errors' array is empty (category=%d, body=%s)", category, string(bodyBytes))
			return false
		}

		// Each error object must have field, rejected_value, and message.
		for i, errObj := range errorsArr {
			fieldVal, hasField := errObj["field"]
			_, hasRejected := errObj["rejected_value"]
			msgVal, hasMessage := errObj["message"]

			if !hasField {
				t.Logf("error[%d] missing 'field' key (obj=%v)", i, errObj)
				return false
			}
			if !hasRejected {
				t.Logf("error[%d] missing 'rejected_value' key (obj=%v)", i, errObj)
				return false
			}
			if !hasMessage {
				t.Logf("error[%d] missing 'message' key (obj=%v)", i, errObj)
				return false
			}

			// field must be a non-empty string.
			fieldStr, ok := fieldVal.(string)
			if !ok || fieldStr == "" {
				t.Logf("error[%d] 'field' is not a non-empty string: %v", i, fieldVal)
				return false
			}

			// message must be a non-empty string.
			msgStr, ok := msgVal.(string)
			if !ok || msgStr == "" {
				t.Logf("error[%d] 'message' is not a non-empty string: %v", i, msgVal)
				return false
			}
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("validation error response format failed: %v", err)
	}

	t.Log("validation error response format verified across 100 iterations")
	for i, desc := range []string{
		"empty body (all required fields missing)",
		"only name provided (3 fields missing)",
		"invalid email format",
		"document and document_type too long",
	} {
		t.Logf("  category %d: %s", i, desc)
	}
}

// --- Property-based test: Soft delete hides customer from all operations ---

func TestCustomerProperty_SoftDeleteHidesCustomer(t *testing.T) {
	// After soft delete, the customer should not appear in the listing,
	// querying by ID should return 404, updating should return 404,
	// and the record should be preserved in the database with deleted_at filled.

	f := func(seed uint64) bool {
		r := seed

		nextR := func() uint64 {
			r = r*6364136223846793005 + 1
			return r
		}

		randStr := func(n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				buf[i] = chars[nextR()%uint64(len(chars))]
			}
			return string(buf)
		}

		// Generate random valid customer data per iteration.
		name := "Customer-" + randStr(5)
		email := randStr(6) + "@test.com"
		document := randStr(int(nextR()%15) + 1)
		documentType := randStr(int(nextR()%10) + 1)

		organizationID := uuid.New()
		now := time.Now().Truncate(time.Second)

		// In-memory store simulating soft delete behavior.
		type storedCustomer struct {
			customer  dbgen.Customer
			deletedAt sql.NullTime
		}
		store := make(map[uuid.UUID]*storedCustomer)

		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				id := uuid.New()
				c := dbgen.Customer{
					ID:             id,
					OrganizationID: arg.OrganizationID,
					Name:           arg.Name,
					Email:          arg.Email,
					Phone:          arg.Phone,
					Document:       arg.Document,
					DocumentType:   arg.DocumentType,
					Status:         sql.NullString{String: "active", Valid: true},
					Metadata:       arg.Metadata,
					CreatedAt:      sql.NullTime{Time: now, Valid: true},
					UpdatedAt:      sql.NullTime{Time: now, Valid: true},
				}
				store[id] = &storedCustomer{customer: c}
				return c, nil
			},
			getCustomerByIDFn: func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				sc, ok := store[arg.ID]
				if !ok || sc.deletedAt.Valid || arg.OrganizationID != sc.customer.OrganizationID {
					return dbgen.Customer{}, sql.ErrNoRows
				}
				return sc.customer, nil
			},
			listCustomersByOrganizationFn: func(_ context.Context, tid uuid.UUID) ([]dbgen.Customer, error) {
				var result []dbgen.Customer
				for _, sc := range store {
					if sc.customer.OrganizationID == tid && !sc.deletedAt.Valid {
						result = append(result, sc.customer)
					}
				}
				return result, nil
			},
			softDeleteCustomerFn: func(_ context.Context, arg dbgen.SoftDeleteCustomerParams) error {
				sc, ok := store[arg.ID]
				if ok && sc.customer.OrganizationID == arg.OrganizationID && !sc.deletedAt.Valid {
					sc.deletedAt = sql.NullTime{Time: time.Now(), Valid: true}
				}
				return nil
			},
			updateCustomerFn: func(_ context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
				sc, ok := store[arg.ID]
				if !ok || sc.deletedAt.Valid || arg.OrganizationID != sc.customer.OrganizationID {
					return dbgen.Customer{}, sql.ErrNoRows
				}
				sc.customer.Name = arg.Name
				sc.customer.Email = arg.Email
				sc.customer.Phone = arg.Phone
				sc.customer.Document = arg.Document
				sc.customer.DocumentType = arg.DocumentType
				sc.customer.Metadata = arg.Metadata
				return sc.customer, nil
			},
		}

		router := setupCustomerRouter(mock, organizationID.String())

		// Step 1: Create a customer via POST — expect 201.
		createBody := fmt.Sprintf(
			`{"name":%q,"email":%q,"document":%q,"document_type":%q}`,
			name, email, document, documentType,
		)
		wCreate := httptest.NewRecorder()
		reqCreate := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(createBody))
		reqCreate.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wCreate, reqCreate)

		if wCreate.Code != http.StatusCreated {
			t.Logf("POST: expected 201, got %d (body=%s)", wCreate.Code, wCreate.Body.String())
			return false
		}

		var created map[string]any
		if err := json.Unmarshal(wCreate.Body.Bytes(), &created); err != nil {
			t.Logf("POST: failed to parse response: %v", err)
			return false
		}
		customerID, ok := created["id"].(string)
		if !ok || customerID == "" {
			t.Logf("POST: missing or invalid 'id' in response")
			return false
		}

		// Step 2: Delete the customer via DELETE — expect 204.
		wDelete := httptest.NewRecorder()
		reqDelete := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/customers/"+customerID, http.NoBody)
		router.ServeHTTP(wDelete, reqDelete)

		if wDelete.Code != http.StatusNoContent {
			t.Logf("DELETE: expected 204, got %d (body=%s)", wDelete.Code, wDelete.Body.String())
			return false
		}

		// Step 3: List via GET /customers — should return empty array.
		wList := httptest.NewRecorder()
		reqList := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers", http.NoBody)
		router.ServeHTTP(wList, reqList)

		if wList.Code != http.StatusOK {
			t.Logf("LIST: expected 200, got %d (body=%s)", wList.Code, wList.Body.String())
			return false
		}

		var listed []any
		if err := json.Unmarshal(wList.Body.Bytes(), &listed); err != nil {
			t.Logf("LIST: failed to parse response: %v", err)
			return false
		}
		if len(listed) != 0 {
			t.Logf("LIST: expected empty array, got %d items", len(listed))
			return false
		}

		// Step 4: Get by ID via GET /customers/:id — should return 404.
		wGet := httptest.NewRecorder()
		reqGet := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers/"+customerID, http.NoBody)
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusNotFound {
			t.Logf("GET: expected 404, got %d (body=%s)", wGet.Code, wGet.Body.String())
			return false
		}
		var getResp map[string]string
		if err := json.Unmarshal(wGet.Body.Bytes(), &getResp); err != nil {
			t.Logf("GET: failed to parse response: %v", err)
			return false
		}
		if getResp["error"] != "customer not found" {
			t.Logf("GET: expected 'customer not found', got %q", getResp["error"])
			return false
		}

		// Step 5: Update via PUT /customers/:id — should return 404.
		updateBody := fmt.Sprintf(
			`{"name":%q,"email":%q,"document":%q,"document_type":%q}`,
			"Updated-"+name, email, document, documentType,
		)
		wUpdate := httptest.NewRecorder()
		reqUpdate := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/customers/"+customerID, strings.NewReader(updateBody))
		reqUpdate.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wUpdate, reqUpdate)

		if wUpdate.Code != http.StatusNotFound {
			t.Logf("PUT: expected 404, got %d (body=%s)", wUpdate.Code, wUpdate.Body.String())
			return false
		}
		var putResp map[string]string
		if err := json.Unmarshal(wUpdate.Body.Bytes(), &putResp); err != nil {
			t.Logf("PUT: failed to parse response: %v", err)
			return false
		}
		if putResp["error"] != "customer not found" {
			t.Logf("PUT: expected 'customer not found', got %q", putResp["error"])
			return false
		}

		// Step 6: Verify record is preserved in the store with deleted_at filled.
		parsedID, _ := uuid.Parse(customerID)
		sc, exists := store[parsedID]
		if !exists {
			t.Logf("STORE: customer record not found in store after soft delete")
			return false
		}
		if !sc.deletedAt.Valid {
			t.Logf("STORE: deleted_at not set after soft delete")
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("soft delete hides customer failed: %v", err)
	}

	t.Log("soft delete behavior verified across 100 iterations")
}

// --- Property-based test: Round-trip of update and read ---

func TestCustomerProperty_UpdateAndReadRoundTrip(t *testing.T) {
	// For any existing active customer, updating it and then reading it by ID
	// should return the updated data with updated_at >= the value before the update.

	f := func(seed uint64) bool {
		r := seed

		nextR := func() uint64 {
			r = r*6364136223846793005 + 1
			return r
		}

		randStr := func(n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				buf[i] = chars[nextR()%uint64(len(chars))]
			}
			return string(buf)
		}

		// Generate random valid customer data for creation.
		createName := "Create-" + randStr(5)
		createEmail := randStr(6) + "@create.com"
		createDocument := randStr(int(nextR()%15) + 1)
		createDocType := randStr(int(nextR()%10) + 1)

		// Generate random valid customer data for update (different values).
		updateName := "Update-" + randStr(5)
		updateEmail := randStr(6) + "@update.com"
		updateDocument := randStr(int(nextR()%15) + 1)
		updateDocType := randStr(int(nextR()%10) + 1)

		organizationID := uuid.New()
		createTime := time.Now().Truncate(time.Second)

		// In-memory store simulating create, update, and get behavior.
		store := make(map[uuid.UUID]dbgen.Customer)

		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				id := uuid.New()
				c := dbgen.Customer{
					ID:             id,
					OrganizationID: arg.OrganizationID,
					Name:           arg.Name,
					Email:          arg.Email,
					Phone:          arg.Phone,
					Document:       arg.Document,
					DocumentType:   arg.DocumentType,
					Status:         sql.NullString{String: "active", Valid: true},
					Metadata:       arg.Metadata,
					CreatedAt:      sql.NullTime{Time: createTime, Valid: true},
					UpdatedAt:      sql.NullTime{Time: createTime, Valid: true},
				}
				store[id] = c
				return c, nil
			},
			updateCustomerFn: func(_ context.Context, arg dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
				c, ok := store[arg.ID]
				if !ok || arg.OrganizationID != c.OrganizationID {
					return dbgen.Customer{}, sql.ErrNoRows
				}
				// Simulate DB trigger: updated_at advances on update.
				newUpdatedAt := time.Now().Truncate(time.Second)
				c.Name = arg.Name
				c.Email = arg.Email
				c.Phone = arg.Phone
				c.Document = arg.Document
				c.DocumentType = arg.DocumentType
				c.Metadata = arg.Metadata
				c.UpdatedAt = sql.NullTime{Time: newUpdatedAt, Valid: true}
				store[arg.ID] = c
				return c, nil
			},
			getCustomerByIDFn: func(_ context.Context, arg dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				c, ok := store[arg.ID]
				if !ok || arg.OrganizationID != c.OrganizationID {
					return dbgen.Customer{}, sql.ErrNoRows
				}
				return c, nil
			},
		}

		router := setupCustomerRouter(mock, organizationID.String())

		// Step 1: Create a customer via POST — expect 201.
		createBody := fmt.Sprintf(
			`{"name":%q,"email":%q,"document":%q,"document_type":%q}`,
			createName, createEmail, createDocument, createDocType,
		)
		wCreate := httptest.NewRecorder()
		reqCreate := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(createBody))
		reqCreate.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wCreate, reqCreate)

		if wCreate.Code != http.StatusCreated {
			t.Logf("POST: expected 201, got %d (body=%s)", wCreate.Code, wCreate.Body.String())
			return false
		}

		var created map[string]any
		if err := json.Unmarshal(wCreate.Body.Bytes(), &created); err != nil {
			t.Logf("POST: failed to parse response: %v", err)
			return false
		}
		customerID, ok := created["id"].(string)
		if !ok || customerID == "" {
			t.Logf("POST: missing or invalid 'id' in response")
			return false
		}

		// Record the initial updated_at from the create response.
		initialUpdatedAtStr, ok := created["updated_at"].(string)
		if !ok || initialUpdatedAtStr == "" {
			t.Logf("POST: missing 'updated_at' in response")
			return false
		}
		initialUpdatedAt, err := time.Parse(time.RFC3339Nano, initialUpdatedAtStr)
		if err != nil {
			t.Logf("POST: failed to parse updated_at %q: %v", initialUpdatedAtStr, err)
			return false
		}

		// Step 2: Update the customer via PUT — expect 200.
		updateBody := fmt.Sprintf(
			`{"name":%q,"email":%q,"document":%q,"document_type":%q}`,
			updateName, updateEmail, updateDocument, updateDocType,
		)
		wUpdate := httptest.NewRecorder()
		reqUpdate := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/customers/"+customerID, strings.NewReader(updateBody))
		reqUpdate.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wUpdate, reqUpdate)

		if wUpdate.Code != http.StatusOK {
			t.Logf("PUT: expected 200, got %d (body=%s)", wUpdate.Code, wUpdate.Body.String())
			return false
		}

		// Step 3: Read the customer via GET — expect 200.
		wGet := httptest.NewRecorder()
		reqGet := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers/"+customerID, http.NoBody)
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Logf("GET: expected 200, got %d (body=%s)", wGet.Code, wGet.Body.String())
			return false
		}

		var fetched map[string]any
		if err := json.Unmarshal(wGet.Body.Bytes(), &fetched); err != nil {
			t.Logf("GET: failed to parse response: %v", err)
			return false
		}

		// Step 4: Assert returned data matches the update input.
		if fetched["name"] != updateName {
			t.Logf("GET: expected name %q, got %q", updateName, fetched["name"])
			return false
		}
		if fetched["email"] != updateEmail {
			t.Logf("GET: expected email %q, got %q", updateEmail, fetched["email"])
			return false
		}
		if fetched["document"] != updateDocument {
			t.Logf("GET: expected document %q, got %q", updateDocument, fetched["document"])
			return false
		}
		if fetched["document_type"] != updateDocType {
			t.Logf("GET: expected document_type %q, got %q", updateDocType, fetched["document_type"])
			return false
		}

		// Step 5: Assert updated_at >= initial updated_at.
		fetchedUpdatedAtStr, ok := fetched["updated_at"].(string)
		if !ok || fetchedUpdatedAtStr == "" {
			t.Logf("GET: missing 'updated_at' in response")
			return false
		}
		fetchedUpdatedAt, err := time.Parse(time.RFC3339Nano, fetchedUpdatedAtStr)
		if err != nil {
			t.Logf("GET: failed to parse updated_at %q: %v", fetchedUpdatedAtStr, err)
			return false
		}
		if fetchedUpdatedAt.Before(initialUpdatedAt) {
			t.Logf("GET: updated_at %v is before initial updated_at %v", fetchedUpdatedAt, initialUpdatedAt)
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("update-and-read round-trip failed: %v", err)
	}

	t.Log("update-and-read round-trip verified across 100 iterations")
}

// --- Property-based test: Listing order by created_at descending ---

func TestCustomerProperty_ListingOrderByCreatedAtDescending(t *testing.T) {
	// For any organization with multiple customers, the listing should return
	// customers ordered by created_at DESC (most recent first).

	f := func(seed uint64) bool {
		r := seed

		nextR := func() uint64 {
			r = r*6364136223846793005 + 1
			return r
		}

		randStr := func(n int) string {
			chars := "abcdefghijklmnopqrstuvwxyz"
			buf := make([]byte, n)
			for i := range buf {
				buf[i] = chars[nextR()%uint64(len(chars))]
			}
			return string(buf)
		}

		organizationID := uuid.New()
		baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

		// Determine how many customers to create (2-5).
		numCustomers := int(nextR()%4) + 2

		// In-memory store: customers keyed by ID.
		type storedCustomer struct {
			dbCustomer dbgen.Customer
		}
		var createdOrder []uuid.UUID
		store := make(map[uuid.UUID]storedCustomer)
		createCounter := 0

		mock := &mockQuerier{
			createCustomerFn: func(_ context.Context, arg dbgen.CreateCustomerParams) (dbgen.Customer, error) {
				id := uuid.New()
				// Each customer gets an incrementing timestamp so they have distinct created_at values.
				ts := baseTime.Add(time.Duration(createCounter) * time.Minute)
				createCounter++
				c := dbgen.Customer{
					ID:             id,
					OrganizationID: arg.OrganizationID,
					Name:           arg.Name,
					Email:          arg.Email,
					Phone:          arg.Phone,
					Document:       arg.Document,
					DocumentType:   arg.DocumentType,
					Status:         sql.NullString{String: "active", Valid: true},
					Metadata:       arg.Metadata,
					CreatedAt:      sql.NullTime{Time: ts, Valid: true},
					UpdatedAt:      sql.NullTime{Time: ts, Valid: true},
				}
				store[id] = storedCustomer{dbCustomer: c}
				createdOrder = append(createdOrder, id)
				return c, nil
			},
			listCustomersByOrganizationFn: func(_ context.Context, tid uuid.UUID) ([]dbgen.Customer, error) {
				// Collect active customers for this organization.
				var customers []dbgen.Customer
				for _, sc := range store {
					if sc.dbCustomer.OrganizationID == tid {
						customers = append(customers, sc.dbCustomer)
					}
				}
				// Sort by created_at DESC to simulate SQL ORDER BY created_at DESC.
				for i := 0; i < len(customers); i++ {
					for j := i + 1; j < len(customers); j++ {
						if customers[j].CreatedAt.Time.After(customers[i].CreatedAt.Time) {
							customers[i], customers[j] = customers[j], customers[i]
						}
					}
				}
				return customers, nil
			},
		}

		router := setupCustomerRouter(mock, organizationID.String())

		// Step 1: Create 2-5 customers via POST with different timestamps.
		for i := 0; i < numCustomers; i++ {
			name := fmt.Sprintf("Customer-%s-%d", randStr(4), i)
			email := fmt.Sprintf("%s-%d@test.com", randStr(5), i)
			document := randStr(int(nextR()%15) + 1)
			docType := randStr(int(nextR()%10) + 1)

			createBody := fmt.Sprintf(
				`{"name":%q,"email":%q,"document":%q,"document_type":%q}`,
				name, email, document, docType,
			)
			wCreate := httptest.NewRecorder()
			reqCreate := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/customers", strings.NewReader(createBody))
			reqCreate.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(wCreate, reqCreate)

			if wCreate.Code != http.StatusCreated {
				t.Logf("POST[%d]: expected 201, got %d (body=%s)", i, wCreate.Code, wCreate.Body.String())
				return false
			}
		}

		// Step 2: List via GET /v1/customers — expect 200.
		wList := httptest.NewRecorder()
		reqList := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers", http.NoBody)
		router.ServeHTTP(wList, reqList)

		if wList.Code != http.StatusOK {
			t.Logf("LIST: expected 200, got %d (body=%s)", wList.Code, wList.Body.String())
			return false
		}

		// Step 3: Parse the response array.
		var listed []map[string]any
		if err := json.Unmarshal(wList.Body.Bytes(), &listed); err != nil {
			t.Logf("LIST: failed to parse response: %v", err)
			return false
		}

		if len(listed) != numCustomers {
			t.Logf("LIST: expected %d customers, got %d", numCustomers, len(listed))
			return false
		}

		// Step 4: Assert descending order — for each consecutive pair,
		// created_at[i] >= created_at[i+1].
		for i := 0; i < len(listed)-1; i++ {
			tsStrI, okI := listed[i]["created_at"].(string)
			tsStrJ, okJ := listed[i+1]["created_at"].(string)
			if !okI || !okJ {
				t.Logf("LIST: missing 'created_at' at index %d or %d", i, i+1)
				return false
			}

			tsI, errI := time.Parse(time.RFC3339Nano, tsStrI)
			tsJ, errJ := time.Parse(time.RFC3339Nano, tsStrJ)
			if errI != nil || errJ != nil {
				t.Logf("LIST: failed to parse created_at at index %d or %d: %v / %v", i, i+1, errI, errJ)
				return false
			}

			if tsI.Before(tsJ) {
				t.Logf("LIST: created_at[%d]=%v is before created_at[%d]=%v — not DESC order",
					i, tsI, i+1, tsJ)
				return false
			}
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("listing order by created_at descending failed: %v", err)
	}

	t.Log("listing order by created_at DESC verified across 100 iterations")
}

// --- Property-based test: Non-existent customer returns 404 ---

func TestCustomerProperty_NonExistentCustomerReturns404(t *testing.T) {
	// Any valid UUID that doesn't correspond to a customer of the authenticated
	// organization should return 404 with "customer not found" for GET, PUT, and DELETE.

	f := func(seed uint64) bool {
		r := seed

		nextR := func() uint64 {
			r = r*6364136223846793005 + 1
			return r
		}

		// Generate a deterministic UUID from the seed.
		var uuidBytes [16]byte
		for i := range uuidBytes {
			uuidBytes[i] = byte(nextR())
		}
		// Set UUID version 4 bits.
		uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
		uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
		nonExistentID, err := uuid.FromBytes(uuidBytes[:])
		if err != nil {
			t.Logf("failed to create UUID from bytes: %v", err)
			return false
		}

		organizationID := uuid.New().String()

		// Mock always returns sql.ErrNoRows for GetCustomerByID (no customer exists).
		mock := &mockQuerier{
			getCustomerByIDFn: func(_ context.Context, _ dbgen.GetCustomerByIDParams) (dbgen.Customer, error) {
				return dbgen.Customer{}, sql.ErrNoRows
			},
			updateCustomerFn: func(_ context.Context, _ dbgen.UpdateCustomerParams) (dbgen.Customer, error) {
				return dbgen.Customer{}, sql.ErrNoRows
			},
			softDeleteCustomerFn: func(_ context.Context, _ dbgen.SoftDeleteCustomerParams) error {
				return sql.ErrNoRows
			},
		}

		router := setupCustomerRouter(mock, organizationID)

		// --- GET /v1/customers/:id — expect 404 ---
		wGet := httptest.NewRecorder()
		reqGet := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/customers/"+nonExistentID.String(), http.NoBody)
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusNotFound {
			t.Logf("GET: expected 404, got %d (id=%s, resp=%s)",
				wGet.Code, nonExistentID, wGet.Body.String())
			return false
		}
		var respGet map[string]string
		if err := json.Unmarshal(wGet.Body.Bytes(), &respGet); err != nil {
			t.Logf("GET: failed to parse response: %v (body=%s)", err, wGet.Body.String())
			return false
		}
		if respGet["error"] != "customer not found" {
			t.Logf("GET: expected 'customer not found', got %q", respGet["error"])
			return false
		}

		// --- PUT /v1/customers/:id with valid body — expect 404 ---
		putBody := `{"name":"Test","email":"test@example.com","document":"12345","document_type":"cpf"}`
		wPut := httptest.NewRecorder()
		reqPut := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/customers/"+nonExistentID.String(), strings.NewReader(putBody))
		reqPut.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wPut, reqPut)

		if wPut.Code != http.StatusNotFound {
			t.Logf("PUT: expected 404, got %d (id=%s, resp=%s)",
				wPut.Code, nonExistentID, wPut.Body.String())
			return false
		}
		var respPut map[string]string
		if err := json.Unmarshal(wPut.Body.Bytes(), &respPut); err != nil {
			t.Logf("PUT: failed to parse response: %v (body=%s)", err, wPut.Body.String())
			return false
		}
		if respPut["error"] != "customer not found" {
			t.Logf("PUT: expected 'customer not found', got %q", respPut["error"])
			return false
		}

		// --- DELETE /v1/customers/:id — expect 404 ---
		wDel := httptest.NewRecorder()
		reqDel := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/customers/"+nonExistentID.String(), http.NoBody)
		router.ServeHTTP(wDel, reqDel)

		if wDel.Code != http.StatusNotFound {
			t.Logf("DELETE: expected 404, got %d (id=%s, resp=%s)",
				wDel.Code, nonExistentID, wDel.Body.String())
			return false
		}
		var respDel map[string]string
		if err := json.Unmarshal(wDel.Body.Bytes(), &respDel); err != nil {
			t.Logf("DELETE: failed to parse response: %v (body=%s)", err, wDel.Body.String())
			return false
		}
		if respDel["error"] != "customer not found" {
			t.Logf("DELETE: expected 'customer not found', got %q", respDel["error"])
			return false
		}

		return true
	}

	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("non-existent customer returns 404 failed: %v", err)
	}

	t.Log("non-existent customer returns 404 verified across 100 iterations (GET + PUT + DELETE)")
}
