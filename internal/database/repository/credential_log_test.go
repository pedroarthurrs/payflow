package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	credentiallogdb "i9pay/internal/database/dbgen/credentiallog"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// mockCredentialLogQuerier is a test double for credentiallogdb.Querier.
type mockCredentialLogQuerier struct {
	createFn  func(ctx context.Context, arg credentiallogdb.CreateCredentialLogParams) error
	getByIDFn func(ctx context.Context, arg credentiallogdb.GetCredentialLogByIDParams) (credentiallogdb.GetCredentialLogByIDRow, error)
	listFn    func(ctx context.Context, arg credentiallogdb.ListCredentialLogsParams) ([]credentiallogdb.ListCredentialLogsRow, error)
}

func (m *mockCredentialLogQuerier) CreateCredentialLog(ctx context.Context, arg credentiallogdb.CreateCredentialLogParams) error {
	if m.createFn != nil {
		return m.createFn(ctx, arg)
	}
	return nil
}

func (m *mockCredentialLogQuerier) GetCredentialLogByID(ctx context.Context, arg credentiallogdb.GetCredentialLogByIDParams) (credentiallogdb.GetCredentialLogByIDRow, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, arg)
	}
	return credentiallogdb.GetCredentialLogByIDRow{}, nil
}

func (m *mockCredentialLogQuerier) ListCredentialLogs(ctx context.Context, arg credentiallogdb.ListCredentialLogsParams) ([]credentiallogdb.ListCredentialLogsRow, error) {
	if m.listFn != nil {
		return m.listFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockCredentialLogQuerier) DeleteCredentialLogsOlderThan(_ context.Context, _ sql.NullTime) (sql.Result, error) {
	return nil, nil
}

func TestCredentialLogRepo_Create_Success(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	method := "POST"
	path := "/v1/customers"
	ip := "127.0.0.1"
	ua := "test-agent"
	status := int32(201)
	latency := int32(42)
	reqBody := []byte(`{"name":"test"}`)
	respBody := []byte(`{"id":"abc"}`)

	var captured credentiallogdb.CreateCredentialLogParams
	q := &mockCredentialLogQuerier{
		createFn: func(_ context.Context, arg credentiallogdb.CreateCredentialLogParams) error {
			captured = arg
			return nil
		},
	}

	repo := NewCredentialLogRepo(q)
	err := repo.Create(context.Background(), domain.CreateCredentialLogInput{
		OrganizationID: orgID,
		CredentialID:   &credID,
		Method:         &method,
		Path:           &path,
		RequestBody:    reqBody,
		ResponseBody:   respBody,
		StatusCode:     &status,
		LatencyMs:      &latency,
		IPAddress:      &ip,
		UserAgent:      &ua,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if captured.OrganizationID != orgID {
		t.Errorf("expected org_id %v, got %v", orgID, captured.OrganizationID)
	}
	if !captured.CredentialID.Valid || captured.CredentialID.UUID != credID {
		t.Errorf("expected credential_id %v, got %v", credID, captured.CredentialID)
	}
	if !captured.Method.Valid || captured.Method.String != method {
		t.Errorf("expected method %q, got %v", method, captured.Method)
	}
	if !captured.Path.Valid || captured.Path.String != path {
		t.Errorf("expected path %q, got %v", path, captured.Path)
	}
	if !captured.StatusCode.Valid || captured.StatusCode.Int32 != status {
		t.Errorf("expected status_code %d, got %v", status, captured.StatusCode)
	}
	if !captured.LatencyMs.Valid || captured.LatencyMs.Int32 != latency {
		t.Errorf("expected latency_ms %d, got %v", latency, captured.LatencyMs)
	}
	if !captured.IpAddress.Valid || captured.IpAddress.String != ip {
		t.Errorf("expected ip_address %q, got %v", ip, captured.IpAddress)
	}
	if !captured.UserAgent.Valid || captured.UserAgent.String != ua {
		t.Errorf("expected user_agent %q, got %v", ua, captured.UserAgent)
	}
}

func TestCredentialLogRepo_Create_NilOptionalFields(t *testing.T) {
	orgID := uuid.New()

	var captured credentiallogdb.CreateCredentialLogParams
	q := &mockCredentialLogQuerier{
		createFn: func(_ context.Context, arg credentiallogdb.CreateCredentialLogParams) error {
			captured = arg
			return nil
		},
	}

	repo := NewCredentialLogRepo(q)
	err := repo.Create(context.Background(), domain.CreateCredentialLogInput{
		OrganizationID: orgID,
		// all optional fields nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if captured.OrganizationID != orgID {
		t.Errorf("expected org_id %v, got %v", orgID, captured.OrganizationID)
	}
	if captured.CredentialID.Valid {
		t.Error("expected credential_id to be NULL")
	}
	if captured.Method.Valid {
		t.Error("expected method to be NULL")
	}
	if captured.Path.Valid {
		t.Error("expected path to be NULL")
	}
	if captured.RequestBody.Valid {
		t.Error("expected request_body to be NULL")
	}
	if captured.ResponseBody.Valid {
		t.Error("expected response_body to be NULL")
	}
	if captured.StatusCode.Valid {
		t.Error("expected status_code to be NULL")
	}
	if captured.LatencyMs.Valid {
		t.Error("expected latency_ms to be NULL")
	}
	if captured.IpAddress.Valid {
		t.Error("expected ip_address to be NULL")
	}
	if captured.UserAgent.Valid {
		t.Error("expected user_agent to be NULL")
	}
}

func TestCredentialLogRepo_Create_DBError_ReturnsErrInternal(t *testing.T) {
	q := &mockCredentialLogQuerier{
		createFn: func(_ context.Context, _ credentiallogdb.CreateCredentialLogParams) error {
			return errors.New("constraint violation: organization_id not found")
		},
	}

	repo := NewCredentialLogRepo(q)
	err := repo.Create(context.Background(), domain.CreateCredentialLogInput{
		OrganizationID: uuid.New(),
	})

	if !errors.Is(err, domain.ErrInternal) {
		t.Errorf("expected domain.ErrInternal, got %v", err)
	}
}
