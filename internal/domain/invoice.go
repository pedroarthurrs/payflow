package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// InvoiceStatus represents the lifecycle state of an invoice.
type InvoiceStatus string

// Invoice status constants.
const (
	InvStatusPending  InvoiceStatus = "pending"
	InvStatusSent     InvoiceStatus = "sent"
	InvStatusPaid     InvoiceStatus = "paid"
	InvStatusError    InvoiceStatus = "error"
	InvStatusCanceled InvoiceStatus = "canceled"
)

// InvoiceType represents the fiscal modality of an invoice.
type InvoiceType string

// Invoice type constants.
const (
	InvTypeNational      InvoiceType = "national"
	InvTypeInternational InvoiceType = "international"
)

// Invoice represents a fiscal document issued by an organization.
type Invoice struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	CustomerID       uuid.UUID
	SubscriptionID   *uuid.UUID
	InvoiceType      InvoiceType
	OperationType    int32
	AmountCents      int32
	Currency         string
	Status           InvoiceStatus
	ERPReference     *string
	CustomerSnapshot json.RawMessage
	ItemsSnapshot    json.RawMessage
	LastSyncLog      json.RawMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Items            []InvoiceItem
}

// InvoiceItem represents a line item within an invoice.
type InvoiceItem struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	InvoiceID       uuid.UUID
	Description     string
	Quantity        string
	UnitPriceCents  int32
	TotalPriceCents int32
	Metadata        json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// InvoiceLog represents an immutable audit record of ERP operations.
type InvoiceLog struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	InvoiceID       uuid.UUID
	EventType       string
	InvoiceStatus   *string
	ERPReference    *string
	ERPErrorCode    *string
	ERPErrorMessage *string
	HTTPStatus      *int32
	RequestURL      *string
	RequestPayload  json.RawMessage
	ResponseBody    json.RawMessage
	AttemptNumber   int32
	Success         bool
	Message         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// InvoiceSetting holds fiscal configuration per organization and invoice type.
type InvoiceSetting struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	Type                InvoiceType
	ERPApiKey           *string
	ERPApiSecret        *string
	ERPAccountID        *string
	NfseServiceCode     *string
	NfseItemListService *string
	TaxRate             *string
	InvoiceSeries       *string
	IsoCurrency         *string
	ExportRegime        *string
	IssueOn             string
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// --- Input structs ---

// CreateInvoiceInput carries the data required to create an invoice.
type CreateInvoiceInput struct {
	CustomerID     uuid.UUID
	SubscriptionID *uuid.UUID
	InvoiceType    InvoiceType
	OperationType  int32
	AmountCents    int32
	Currency       string
	Items          []CreateInvoiceItemInput
}

// CreateInvoiceItemInput carries the data for a single invoice item.
type CreateInvoiceItemInput struct {
	Description     string
	Quantity        string
	UnitPriceCents  int32
	TotalPriceCents int32
	Metadata        json.RawMessage
}

// CreateInvoiceLogInput carries the data for creating an invoice log entry.
type CreateInvoiceLogInput struct {
	InvoiceID       uuid.UUID
	EventType       string
	InvoiceStatus   *string
	ERPReference    *string
	ERPErrorCode    *string
	ERPErrorMessage *string
	HTTPStatus      *int32
	RequestURL      *string
	RequestPayload  json.RawMessage
	ResponseBody    json.RawMessage
	AttemptNumber   int32
	Success         bool
	Message         string
}

// UpsertInvoiceSettingInput carries the data for creating or updating an invoice setting.
type UpsertInvoiceSettingInput struct {
	Type                InvoiceType
	ERPApiKey           *string
	ERPApiSecret        *string
	ERPAccountID        *string
	NfseServiceCode     *string
	NfseItemListService *string
	TaxRate             *string
	InvoiceSeries       *string
	IsoCurrency         *string
	ExportRegime        *string
	IssueOn             string
	IsActive            bool
}

// InvoiceListParams carries filter and pagination parameters for listing invoices.
type InvoiceListParams struct {
	ListParams
	CustomerID     *uuid.UUID
	SubscriptionID *uuid.UUID
	Status         *InvoiceStatus
	InvoiceType    *InvoiceType
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
}

// --- Repository interfaces ---

// InvoiceRepository defines persistence operations for invoices.
type InvoiceRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CreateInvoiceInput) (*Invoice, error)
	GetByID(ctx context.Context, organizationID, invoiceID uuid.UUID) (*Invoice, error)
	List(ctx context.Context, organizationID uuid.UUID, params InvoiceListParams) ([]Invoice, error)
	Count(ctx context.Context, organizationID uuid.UUID, params InvoiceListParams) (int64, error)
	UpdateStatus(ctx context.Context, organizationID, invoiceID uuid.UUID, newStatus InvoiceStatus) (*Invoice, error)
	UpdateERPReference(ctx context.Context, organizationID, invoiceID uuid.UUID, erpReference string) error
	UpdateLastSyncLog(ctx context.Context, organizationID, invoiceID uuid.UUID, syncLog json.RawMessage) error
}

// InvoiceItemRepository defines persistence operations for invoice items.
type InvoiceItemRepository interface {
	Create(ctx context.Context, organizationID, invoiceID uuid.UUID, input CreateInvoiceItemInput) (*InvoiceItem, error)
	ListByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID) ([]InvoiceItem, error)
	DeleteByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID) error
}

// InvoiceLogRepository defines persistence operations for invoice logs.
type InvoiceLogRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CreateInvoiceLogInput) (*InvoiceLog, error)
	ListByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID, params ListParams) ([]InvoiceLog, error)
	CountByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID) (int64, error)
}

// InvoiceSettingRepository defines persistence operations for invoice settings.
type InvoiceSettingRepository interface {
	Upsert(ctx context.Context, organizationID uuid.UUID, input UpsertInvoiceSettingInput) (*InvoiceSetting, error)
	GetByOrgAndType(ctx context.Context, organizationID uuid.UUID, invoiceType InvoiceType) (*InvoiceSetting, error)
	List(ctx context.Context, organizationID uuid.UUID) ([]InvoiceSetting, error)
	Delete(ctx context.Context, organizationID, settingID uuid.UUID) error
}

// --- Invoice State Machine ---

// InvoiceStateMachine is the pure FSM for invoices — no infrastructure dependencies.
type InvoiceStateMachine struct{}

// invoiceTransitionRules is the immutable graph of valid transitions.
var invoiceTransitionRules = map[InvoiceStatus][]InvoiceStatus{
	InvStatusPending:  {InvStatusSent, InvStatusCanceled},
	InvStatusSent:     {InvStatusPaid, InvStatusError, InvStatusCanceled},
	InvStatusError:    {InvStatusSent, InvStatusCanceled},
	InvStatusPaid:     {},
	InvStatusCanceled: {},
}

// invoiceTerminalStates is the set of states from which no transition is allowed.
var invoiceTerminalStates = map[InvoiceStatus]bool{
	InvStatusPaid:     true,
	InvStatusCanceled: true,
}

// NewInvoiceStateMachine creates an InvoiceStateMachine.
func NewInvoiceStateMachine() *InvoiceStateMachine {
	return &InvoiceStateMachine{}
}

// Validate returns nil if the transition from → to is valid, or an error otherwise.
func (sm *InvoiceStateMachine) Validate(from, to InvoiceStatus) error {
	if from == to {
		return ErrSameStateTransition
	}
	if invoiceTerminalStates[from] {
		if from == InvStatusCanceled {
			return ErrInvoiceAlreadyCanceled
		}
		return ErrTerminalState
	}
	for _, allowed := range invoiceTransitionRules[from] {
		if allowed == to {
			return nil
		}
	}
	return fmt.Errorf("%w: %s → %s", ErrInvalidInvoiceTransition, from, to)
}

// AllowedTransitions returns the destination states allowed from the given status.
// Returns an empty slice for terminal states.
func (sm *InvoiceStateMachine) AllowedTransitions(from InvoiceStatus) []InvoiceStatus {
	return invoiceTransitionRules[from]
}

// IsTerminal returns true if status is a terminal state (paid or canceled).
func (sm *InvoiceStateMachine) IsTerminal(status InvoiceStatus) bool {
	return invoiceTerminalStates[status]
}
