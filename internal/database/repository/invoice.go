package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	invoicedb "payflow/internal/database/dbgen/invoice"
	"payflow/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// InvoiceRepo
// =============================================================================

// InvoiceRepo implements domain.InvoiceRepository.
type InvoiceRepo struct {
	q invoicedb.Querier
}

// NewInvoiceRepo creates an InvoiceRepo.
func NewInvoiceRepo(q invoicedb.Querier) *InvoiceRepo {
	return &InvoiceRepo{q: q}
}

// Create inserts a new invoice.
func (r *InvoiceRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateInvoiceInput) (*domain.Invoice, error) {
	row, err := r.q.CreateInvoice(ctx, invoicedb.CreateInvoiceParams{
		OrganizationID: organizationID,
		CustomerID:     input.CustomerID,
		SubscriptionID: toNullUUID(input.SubscriptionID),
		InvoiceType:    string(input.InvoiceType),
		OperationType:  input.OperationType,
		AmountCents:    input.AmountCents,
		Currency:       sql.NullString{String: input.Currency, Valid: input.Currency != ""},
	})
	if err != nil {
		slog.Error("failed to create invoice", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	inv := toDomainInvoice(row)
	return &inv, nil
}

// GetByID retrieves an invoice by ID scoped to an organization.
func (r *InvoiceRepo) GetByID(ctx context.Context, organizationID, invoiceID uuid.UUID) (*domain.Invoice, error) {
	row, err := r.q.GetInvoiceByID(ctx, invoicedb.GetInvoiceByIDParams{
		OrganizationID: organizationID,
		ID:             invoiceID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		slog.Error("failed to get invoice", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return nil, domain.ErrInternal
	}
	inv := toDomainInvoice(row)
	return &inv, nil
}

// List returns invoices for an organization with optional filters.
func (r *InvoiceRepo) List(ctx context.Context, organizationID uuid.UUID, params domain.InvoiceListParams) ([]domain.Invoice, error) {
	rows, err := r.q.ListInvoices(ctx, invoicedb.ListInvoicesParams{
		OrganizationID: organizationID,
		Limit:          params.Limit,
		Offset:         params.Offset,
		CustomerID:     toNullUUID(params.CustomerID),
		SubscriptionID: toNullUUID(params.SubscriptionID),
		Status:         toNullStringFromInvoiceStatus(params.Status),
		InvoiceType:    toNullStringFromInvoiceType(params.InvoiceType),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
	})
	if err != nil {
		slog.Error("failed to list invoices", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	invoices := make([]domain.Invoice, len(rows))
	for i := range rows {
		invoices[i] = toDomainInvoice(rows[i])
	}
	return invoices, nil
}

// Count returns the total count of invoices matching the filters.
func (r *InvoiceRepo) Count(ctx context.Context, organizationID uuid.UUID, params domain.InvoiceListParams) (int64, error) {
	count, err := r.q.CountInvoices(ctx, invoicedb.CountInvoicesParams{
		OrganizationID: organizationID,
		CustomerID:     toNullUUID(params.CustomerID),
		SubscriptionID: toNullUUID(params.SubscriptionID),
		Status:         toNullStringFromInvoiceStatus(params.Status),
		InvoiceType:    toNullStringFromInvoiceType(params.InvoiceType),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
	})
	if err != nil {
		slog.Error("failed to count invoices", "organization_id", organizationID, "error", err)
		return 0, domain.ErrInternal
	}
	return count, nil
}

// UpdateStatus transitions the invoice to a new status.
func (r *InvoiceRepo) UpdateStatus(ctx context.Context, organizationID, invoiceID uuid.UUID, newStatus domain.InvoiceStatus) (*domain.Invoice, error) {
	row, err := r.q.UpdateInvoiceStatus(ctx, invoicedb.UpdateInvoiceStatusParams{
		OrganizationID: organizationID,
		ID:             invoiceID,
		Status:         sql.NullString{String: string(newStatus), Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		slog.Error("failed to update invoice status", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return nil, domain.ErrInternal
	}
	inv := toDomainInvoice(row)
	return &inv, nil
}

// UpdateERPReference sets the ERP reference on an invoice.
func (r *InvoiceRepo) UpdateERPReference(ctx context.Context, organizationID, invoiceID uuid.UUID, erpReference string) error {
	err := r.q.UpdateInvoiceERPReference(ctx, invoicedb.UpdateInvoiceERPReferenceParams{
		OrganizationID: organizationID,
		ID:             invoiceID,
		ErpReference:   sql.NullString{String: erpReference, Valid: true},
	})
	if err != nil {
		slog.Error("failed to update invoice erp reference", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// UpdateLastSyncLog sets the last sync log on an invoice.
func (r *InvoiceRepo) UpdateLastSyncLog(ctx context.Context, organizationID, invoiceID uuid.UUID, syncLog json.RawMessage) error {
	err := r.q.UpdateInvoiceLastSyncLog(ctx, invoicedb.UpdateInvoiceLastSyncLogParams{
		OrganizationID: organizationID,
		ID:             invoiceID,
		LastSyncLog:    toNullRawMessage(syncLog),
	})
	if err != nil {
		slog.Error("failed to update invoice last sync log", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// =============================================================================
// InvoiceItemRepo
// =============================================================================

// InvoiceItemRepo implements domain.InvoiceItemRepository.
type InvoiceItemRepo struct {
	q invoicedb.Querier
}

// NewInvoiceItemRepo creates an InvoiceItemRepo.
func NewInvoiceItemRepo(q invoicedb.Querier) *InvoiceItemRepo {
	return &InvoiceItemRepo{q: q}
}

// Create inserts a new invoice item.
func (r *InvoiceItemRepo) Create(ctx context.Context, organizationID, invoiceID uuid.UUID, input domain.CreateInvoiceItemInput) (*domain.InvoiceItem, error) {
	row, err := r.q.CreateInvoiceItem(ctx, invoicedb.CreateInvoiceItemParams{
		OrganizationID:  organizationID,
		InvoiceID:       invoiceID,
		Description:     input.Description,
		Quantity:        input.Quantity,
		UnitPriceCents:  input.UnitPriceCents,
		TotalPriceCents: input.TotalPriceCents,
		Metadata:        toNullRawMessage(input.Metadata),
	})
	if err != nil {
		slog.Error("failed to create invoice item", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return nil, domain.ErrInternal
	}
	item := toDomainInvoiceItem(row)
	return &item, nil
}

// ListByInvoice returns all items for an invoice.
func (r *InvoiceItemRepo) ListByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID) ([]domain.InvoiceItem, error) {
	rows, err := r.q.ListInvoiceItemsByInvoice(ctx, invoicedb.ListInvoiceItemsByInvoiceParams{
		OrganizationID: organizationID,
		InvoiceID:      invoiceID,
	})
	if err != nil {
		slog.Error("failed to list invoice items", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return nil, domain.ErrInternal
	}
	items := make([]domain.InvoiceItem, len(rows))
	for i := range rows {
		items[i] = toDomainInvoiceItem(rows[i])
	}
	return items, nil
}

// DeleteByInvoice removes all items for an invoice.
func (r *InvoiceItemRepo) DeleteByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID) error {
	err := r.q.DeleteInvoiceItemsByInvoice(ctx, invoicedb.DeleteInvoiceItemsByInvoiceParams{
		OrganizationID: organizationID,
		InvoiceID:      invoiceID,
	})
	if err != nil {
		slog.Error("failed to delete invoice items", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// =============================================================================
// InvoiceLogRepo
// =============================================================================

// InvoiceLogRepo implements domain.InvoiceLogRepository.
type InvoiceLogRepo struct {
	q invoicedb.Querier
}

// NewInvoiceLogRepo creates an InvoiceLogRepo.
func NewInvoiceLogRepo(q invoicedb.Querier) *InvoiceLogRepo {
	return &InvoiceLogRepo{q: q}
}

// Create inserts a new invoice log entry.
func (r *InvoiceLogRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateInvoiceLogInput) (*domain.InvoiceLog, error) {
	row, err := r.q.CreateInvoiceLog(ctx, invoicedb.CreateInvoiceLogParams{
		OrganizationID:  organizationID,
		InvoiceID:       input.InvoiceID,
		EventType:       input.EventType,
		InvoiceStatus:   toNullString(input.InvoiceStatus),
		ErpReference:    toNullString(input.ERPReference),
		ErpErrorCode:    toNullString(input.ERPErrorCode),
		ErpErrorMessage: toNullString(input.ERPErrorMessage),
		HttpStatus:      toNullInt32(input.HTTPStatus),
		RequestUrl:      toNullString(input.RequestURL),
		RequestPayload:  toNullRawMessage(input.RequestPayload),
		ResponseBody:    toNullRawMessage(input.ResponseBody),
		AttemptNumber:   input.AttemptNumber,
		Success:         input.Success,
		Message:         input.Message,
	})
	if err != nil {
		slog.Error("failed to create invoice log", "organization_id", organizationID, "invoice_id", input.InvoiceID, "error", err)
		return nil, domain.ErrInternal
	}
	l := toDomainInvoiceLog(row)
	return &l, nil
}

// ListByInvoice returns paginated logs for an invoice.
func (r *InvoiceLogRepo) ListByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID, params domain.ListParams) ([]domain.InvoiceLog, error) {
	rows, err := r.q.ListInvoiceLogsByInvoice(ctx, invoicedb.ListInvoiceLogsByInvoiceParams{
		OrganizationID: organizationID,
		InvoiceID:      invoiceID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list invoice logs", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return nil, domain.ErrInternal
	}
	logs := make([]domain.InvoiceLog, len(rows))
	for i := range rows {
		logs[i] = toDomainInvoiceLog(rows[i])
	}
	return logs, nil
}

// CountByInvoice returns the total count of logs for an invoice.
func (r *InvoiceLogRepo) CountByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID) (int64, error) {
	count, err := r.q.CountInvoiceLogsByInvoice(ctx, invoicedb.CountInvoiceLogsByInvoiceParams{
		OrganizationID: organizationID,
		InvoiceID:      invoiceID,
	})
	if err != nil {
		slog.Error("failed to count invoice logs", "organization_id", organizationID, "invoice_id", invoiceID, "error", err)
		return 0, domain.ErrInternal
	}
	return count, nil
}

// =============================================================================
// InvoiceSettingRepo
// =============================================================================

// InvoiceSettingRepo implements domain.InvoiceSettingRepository.
type InvoiceSettingRepo struct {
	q invoicedb.Querier
}

// NewInvoiceSettingRepo creates an InvoiceSettingRepo.
func NewInvoiceSettingRepo(q invoicedb.Querier) *InvoiceSettingRepo {
	return &InvoiceSettingRepo{q: q}
}

// Upsert creates or updates an invoice setting.
func (r *InvoiceSettingRepo) Upsert(ctx context.Context, organizationID uuid.UUID, input domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error) {
	row, err := r.q.UpsertInvoiceSetting(ctx, invoicedb.UpsertInvoiceSettingParams{
		OrganizationID:      organizationID,
		Type:                string(input.Type),
		ErpApiKey:           toNullString(input.ERPApiKey),
		ErpApiSecret:        toNullString(input.ERPApiSecret),
		ErpAccountID:        toNullString(input.ERPAccountID),
		NfseServiceCode:     toNullString(input.NfseServiceCode),
		NfseItemListService: toNullString(input.NfseItemListService),
		TaxRate:             toNullString(input.TaxRate),
		InvoiceSeries:       toNullString(input.InvoiceSeries),
		IsoCurrency:         toNullString(input.IsoCurrency),
		ExportRegime:        toNullString(input.ExportRegime),
		IssueOn:             sql.NullString{String: input.IssueOn, Valid: input.IssueOn != ""},
		IsActive:            sql.NullBool{Bool: input.IsActive, Valid: true},
	})
	if err != nil {
		slog.Error("failed to upsert invoice setting", "organization_id", organizationID, "type", input.Type, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainInvoiceSetting(row)
	return &s, nil
}

// GetByOrgAndType retrieves an invoice setting by organization and type.
func (r *InvoiceSettingRepo) GetByOrgAndType(ctx context.Context, organizationID uuid.UUID, invoiceType domain.InvoiceType) (*domain.InvoiceSetting, error) {
	row, err := r.q.GetInvoiceSettingByOrgAndType(ctx, invoicedb.GetInvoiceSettingByOrgAndTypeParams{
		OrganizationID: organizationID,
		Type:           string(invoiceType),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvoiceSettingNotFound
		}
		slog.Error("failed to get invoice setting", "organization_id", organizationID, "type", invoiceType, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainInvoiceSetting(row)
	return &s, nil
}

// List returns all invoice settings for an organization.
func (r *InvoiceSettingRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.InvoiceSetting, error) {
	rows, err := r.q.ListInvoiceSettings(ctx, organizationID)
	if err != nil {
		slog.Error("failed to list invoice settings", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	settings := make([]domain.InvoiceSetting, len(rows))
	for i := range rows {
		settings[i] = toDomainInvoiceSetting(rows[i])
	}
	return settings, nil
}

// Delete removes an invoice setting.
func (r *InvoiceSettingRepo) Delete(ctx context.Context, organizationID, settingID uuid.UUID) error {
	err := r.q.DeleteInvoiceSetting(ctx, invoicedb.DeleteInvoiceSettingParams{
		OrganizationID: organizationID,
		ID:             settingID,
	})
	if err != nil {
		slog.Error("failed to delete invoice setting", "organization_id", organizationID, "setting_id", settingID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// =============================================================================
// Mapping helpers
// =============================================================================

func toDomainInvoice(row invoicedb.Invoice) domain.Invoice {
	inv := domain.Invoice{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		CustomerID:     row.CustomerID,
		InvoiceType:    domain.InvoiceType(row.InvoiceType),
		OperationType:  row.OperationType,
		AmountCents:    row.AmountCents,
	}
	if row.SubscriptionID.Valid {
		inv.SubscriptionID = &row.SubscriptionID.UUID
	}
	if row.Currency.Valid {
		inv.Currency = row.Currency.String
	}
	if row.Status.Valid {
		inv.Status = domain.InvoiceStatus(row.Status.String)
	}
	if row.ErpReference.Valid {
		inv.ERPReference = &row.ErpReference.String
	}
	if row.CustomerSnapshot.Valid {
		inv.CustomerSnapshot = row.CustomerSnapshot.RawMessage
	}
	if row.ItemsSnapshot.Valid {
		inv.ItemsSnapshot = row.ItemsSnapshot.RawMessage
	}
	if row.LastSyncLog.Valid {
		inv.LastSyncLog = row.LastSyncLog.RawMessage
	}
	if row.CreatedAt.Valid {
		inv.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		inv.UpdatedAt = row.UpdatedAt.Time
	}
	return inv
}

func toDomainInvoiceItem(row invoicedb.InvoiceItem) domain.InvoiceItem {
	item := domain.InvoiceItem{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		InvoiceID:       row.InvoiceID,
		Description:     row.Description,
		Quantity:        row.Quantity,
		UnitPriceCents:  row.UnitPriceCents,
		TotalPriceCents: row.TotalPriceCents,
	}
	if row.Metadata.Valid {
		item.Metadata = row.Metadata.RawMessage
	}
	if row.CreatedAt.Valid {
		item.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	return item
}

func toDomainInvoiceLog(row invoicedb.InvoiceLog) domain.InvoiceLog {
	l := domain.InvoiceLog{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		InvoiceID:      row.InvoiceID,
		EventType:      row.EventType,
		AttemptNumber:  row.AttemptNumber,
		Success:        row.Success,
		Message:        row.Message,
	}
	if row.InvoiceStatus.Valid {
		l.InvoiceStatus = &row.InvoiceStatus.String
	}
	if row.ErpReference.Valid {
		l.ERPReference = &row.ErpReference.String
	}
	if row.ErpErrorCode.Valid {
		l.ERPErrorCode = &row.ErpErrorCode.String
	}
	if row.ErpErrorMessage.Valid {
		l.ERPErrorMessage = &row.ErpErrorMessage.String
	}
	if row.HttpStatus.Valid {
		l.HTTPStatus = &row.HttpStatus.Int32
	}
	if row.RequestUrl.Valid {
		l.RequestURL = &row.RequestUrl.String
	}
	if row.RequestPayload.Valid {
		l.RequestPayload = row.RequestPayload.RawMessage
	}
	if row.ResponseBody.Valid {
		l.ResponseBody = row.ResponseBody.RawMessage
	}
	if row.CreatedAt.Valid {
		l.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		l.UpdatedAt = row.UpdatedAt.Time
	}
	return l
}

func toDomainInvoiceSetting(row invoicedb.InvoiceSetting) domain.InvoiceSetting {
	s := domain.InvoiceSetting{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Type:           domain.InvoiceType(row.Type),
	}
	if row.ErpApiKey.Valid {
		s.ERPApiKey = &row.ErpApiKey.String
	}
	if row.ErpApiSecret.Valid {
		s.ERPApiSecret = &row.ErpApiSecret.String
	}
	if row.ErpAccountID.Valid {
		s.ERPAccountID = &row.ErpAccountID.String
	}
	if row.NfseServiceCode.Valid {
		s.NfseServiceCode = &row.NfseServiceCode.String
	}
	if row.NfseItemListService.Valid {
		s.NfseItemListService = &row.NfseItemListService.String
	}
	if row.TaxRate.Valid {
		s.TaxRate = &row.TaxRate.String
	}
	if row.InvoiceSeries.Valid {
		s.InvoiceSeries = &row.InvoiceSeries.String
	}
	if row.IsoCurrency.Valid {
		s.IsoCurrency = &row.IsoCurrency.String
	}
	if row.ExportRegime.Valid {
		s.ExportRegime = &row.ExportRegime.String
	}
	if row.IssueOn.Valid {
		s.IssueOn = row.IssueOn.String
	}
	if row.IsActive.Valid {
		s.IsActive = row.IsActive.Bool
	}
	if row.CreatedAt.Valid {
		s.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		s.UpdatedAt = row.UpdatedAt.Time
	}
	return s
}

// toNullInt32 converts a *int32 to sql.NullInt32.
func toNullInt32(v *int32) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *v, Valid: true}
}

// toNullStringFromInvoiceStatus converts a *domain.InvoiceStatus to sql.NullString.
func toNullStringFromInvoiceStatus(s *domain.InvoiceStatus) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(*s), Valid: true}
}

// toNullStringFromInvoiceType converts a *domain.InvoiceType to sql.NullString.
func toNullStringFromInvoiceType(t *domain.InvoiceType) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(*t), Valid: true}
}

// Ensure interfaces are satisfied at compile time.
var _ domain.InvoiceRepository = (*InvoiceRepo)(nil)
var _ domain.InvoiceItemRepository = (*InvoiceItemRepo)(nil)
var _ domain.InvoiceLogRepository = (*InvoiceLogRepo)(nil)
var _ domain.InvoiceSettingRepository = (*InvoiceSettingRepo)(nil)
