package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// WebhookDispatcher abstracts webhook event dispatching so the service
// does not depend on the concrete WebhookScheduler from the worker package.
type WebhookDispatcher interface {
	Dispatch(ctx context.Context, orgID uuid.UUID, eventType, environment string, payload []byte) error
}

// InvoiceService handles invoice business logic, including CRUD operations,
// FSM-driven state transitions, automatic generation from subscriptions,
// and invoice settings management.
type InvoiceService struct {
	invoices   domain.InvoiceRepository
	items      domain.InvoiceItemRepository
	logs       domain.InvoiceLogRepository
	settings   domain.InvoiceSettingRepository
	customers  domain.CustomerRepository
	subs       domain.SubscriptionRepository
	subItems   domain.SubscriptionItemRepository
	plans      domain.PlanRepository
	fsm        *domain.InvoiceStateMachine
	txMgr      domain.TxManager
	dispatcher WebhookDispatcher
}

// NewInvoiceService creates an InvoiceService.
func NewInvoiceService(
	invoices domain.InvoiceRepository,
	items domain.InvoiceItemRepository,
	logs domain.InvoiceLogRepository,
	settings domain.InvoiceSettingRepository,
	customers domain.CustomerRepository,
	subs domain.SubscriptionRepository,
	subItems domain.SubscriptionItemRepository,
	plans domain.PlanRepository,
	txMgr domain.TxManager,
	dispatcher WebhookDispatcher,
) *InvoiceService {
	return &InvoiceService{
		invoices:   invoices,
		items:      items,
		logs:       logs,
		settings:   settings,
		customers:  customers,
		subs:       subs,
		subItems:   subItems,
		plans:      plans,
		fsm:        domain.NewInvoiceStateMachine(),
		txMgr:      txMgr,
		dispatcher: dispatcher,
	}
}

// Create creates a new invoice with its items atomically inside a transaction.
func (s *InvoiceService) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateInvoiceInput) (*domain.Invoice, error) {
	if len(input.Items) == 0 {
		return nil, domain.ErrInvoiceItemsRequired
	}

	// Validate customer exists.
	if _, err := s.customers.GetByID(ctx, organizationID, input.CustomerID); err != nil {
		return nil, err
	}

	// Calculate amount_cents = sum of all items' total_price_cents.
	var amountCents int32
	for _, item := range input.Items {
		amountCents += item.TotalPriceCents
	}
	input.AmountCents = amountCents

	// Default currency.
	if input.Currency == "" {
		input.Currency = "BRL"
	}
	// Default operation type.
	if input.OperationType == 0 {
		input.OperationType = 3
	}

	var invoice *domain.Invoice
	err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		invoiceRepo := tx.InvoiceRepo()
		itemRepo := tx.InvoiceItemRepo()

		created, err := invoiceRepo.Create(ctx, organizationID, input)
		if err != nil {
			return err
		}

		createdItems := make([]domain.InvoiceItem, 0, len(input.Items))
		for _, itemInput := range input.Items {
			item, err := itemRepo.Create(ctx, organizationID, created.ID, itemInput)
			if err != nil {
				return err
			}
			createdItems = append(createdItems, *item)
		}
		created.Items = createdItems
		invoice = created
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Dispatch webhook — non-fatal.
	s.dispatchWebhook(ctx, organizationID, domain.EventInvoiceCreated, invoice)

	return invoice, nil
}

// GetByID retrieves an invoice by ID with its items loaded.
func (s *InvoiceService) GetByID(ctx context.Context, organizationID, invoiceID uuid.UUID) (*domain.Invoice, error) {
	invoice, err := s.invoices.GetByID(ctx, organizationID, invoiceID)
	if err != nil {
		return nil, err
	}
	items, err := s.items.ListByInvoice(ctx, organizationID, invoiceID)
	if err != nil {
		return nil, err
	}
	invoice.Items = items
	return invoice, nil
}

// List returns invoices with offset-based pagination and filters.
func (s *InvoiceService) List(ctx context.Context, organizationID uuid.UUID, params domain.InvoiceListParams) ([]domain.Invoice, int64, error) {
	invoices, err := s.invoices.List(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.invoices.Count(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	return invoices, total, nil
}

// Transition applies an invoice status transition, creates a log entry,
// and dispatches the appropriate webhook event.
func (s *InvoiceService) Transition(
	ctx context.Context,
	organizationID, invoiceID uuid.UUID,
	newStatus domain.InvoiceStatus,
	message string,
) (*domain.Invoice, error) {
	// Load current invoice.
	invoice, err := s.invoices.GetByID(ctx, organizationID, invoiceID)
	if err != nil {
		return nil, err
	}

	// Validate FSM transition.
	if err := s.fsm.Validate(invoice.Status, newStatus); err != nil {
		return nil, err
	}

	// Update status.
	updated, err := s.invoices.UpdateStatus(ctx, organizationID, invoiceID, newStatus)
	if err != nil {
		return nil, err
	}

	// Create transition log.
	oldStatus := string(invoice.Status)
	newStatusStr := string(newStatus)
	if _, err := s.logs.Create(ctx, organizationID, domain.CreateInvoiceLogInput{
		InvoiceID:     invoiceID,
		EventType:     fmt.Sprintf("status_transition:%s→%s", oldStatus, newStatusStr),
		InvoiceStatus: &newStatusStr,
		AttemptNumber: 1,
		Success:       true,
		Message:       message,
	}); err != nil {
		slog.Warn("failed to create invoice transition log", "invoice_id", invoiceID, "error", err)
	}

	// Load items for the response and webhook payload.
	items, err := s.items.ListByInvoice(ctx, organizationID, invoiceID)
	if err != nil {
		slog.Warn("failed to load invoice items after transition", "invoice_id", invoiceID, "error", err)
	} else {
		updated.Items = items
	}

	// Dispatch webhook based on new status.
	switch newStatus { //nolint:exhaustive // only paid and error trigger webhooks
	case domain.InvStatusPaid:
		s.dispatchWebhook(ctx, organizationID, domain.EventInvoicePaid, updated)
	case domain.InvStatusError:
		s.dispatchWebhook(ctx, organizationID, domain.EventInvoicePaymentFailed, updated)
	}

	return updated, nil
}

// Cancel cancels an invoice (shortcut to Transition → canceled).
func (s *InvoiceService) Cancel(ctx context.Context, organizationID, invoiceID uuid.UUID) (*domain.Invoice, error) {
	return s.Transition(ctx, organizationID, invoiceID, domain.InvStatusCanceled, "canceled by user")
}

// GenerateFromSubscription creates an invoice from a subscription's items.
// Returns nil without error if no InvoiceSetting is configured (non-fatal skip).
func (s *InvoiceService) GenerateFromSubscription(
	ctx context.Context,
	organizationID, subscriptionID uuid.UUID,
	paymentID *uuid.UUID,
) (*domain.Invoice, error) {
	// Load subscription.
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("loading subscription: %w", err)
	}

	// Load subscription items.
	subItems, err := s.subItems.ListBySubscription(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("loading subscription items: %w", err)
	}

	// Try national first, then international.
	setting, err := s.findActiveSetting(ctx, organizationID)
	if err != nil {
		slog.Warn("no invoice setting found, skipping invoice generation",
			"organization_id", organizationID,
			"subscription_id", subscriptionID,
		)
		return nil, nil //nolint:nilnil,nilerr // non-fatal skip by design
	}

	// Build invoice items from subscription items + plans.
	invoiceItems := make([]domain.CreateInvoiceItemInput, 0, len(subItems))
	var totalAmountCents int32

	for _, si := range subItems {
		plan, err := s.plans.GetByID(ctx, organizationID, si.PlanID)
		if err != nil {
			return nil, fmt.Errorf("loading plan %s: %w", si.PlanID, err)
		}

		itemTotal := plan.AmountCents * si.Quantity
		totalAmountCents += itemTotal

		invoiceItems = append(invoiceItems, domain.CreateInvoiceItemInput{
			Description:     plan.Name,
			Quantity:        fmt.Sprintf("%d", si.Quantity),
			UnitPriceCents:  plan.AmountCents,
			TotalPriceCents: itemTotal,
		})
	}

	input := domain.CreateInvoiceInput{
		CustomerID:     sub.CustomerID,
		SubscriptionID: &subscriptionID,
		InvoiceType:    setting.Type,
		OperationType:  3,
		AmountCents:    totalAmountCents,
		Currency:       "BRL",
		Items:          invoiceItems,
	}

	return s.Create(ctx, organizationID, input)
}

// --- Invoice Logs ---

// ListLogs returns invoice logs with offset-based pagination.
func (s *InvoiceService) ListLogs(ctx context.Context, organizationID, invoiceID uuid.UUID, params domain.ListParams) ([]domain.InvoiceLog, error) {
	return s.logs.ListByInvoice(ctx, organizationID, invoiceID, params)
}

// CountLogs returns the total number of logs for an invoice.
func (s *InvoiceService) CountLogs(ctx context.Context, organizationID, invoiceID uuid.UUID) (int64, error) {
	return s.logs.CountByInvoice(ctx, organizationID, invoiceID)
}

// --- Invoice Settings CRUD ---

// UpsertSetting creates or updates an invoice setting.
func (s *InvoiceService) UpsertSetting(ctx context.Context, organizationID uuid.UUID, input domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error) {
	return s.settings.Upsert(ctx, organizationID, input)
}

// GetSettingByOrgAndType returns the invoice setting for the given type.
func (s *InvoiceService) GetSettingByOrgAndType(ctx context.Context, organizationID uuid.UUID, invoiceType domain.InvoiceType) (*domain.InvoiceSetting, error) {
	return s.settings.GetByOrgAndType(ctx, organizationID, invoiceType)
}

// ListSettings returns all invoice settings for the organization.
func (s *InvoiceService) ListSettings(ctx context.Context, organizationID uuid.UUID) ([]domain.InvoiceSetting, error) {
	return s.settings.List(ctx, organizationID)
}

// DeleteSetting removes an invoice setting.
func (s *InvoiceService) DeleteSetting(ctx context.Context, organizationID, settingID uuid.UUID) error {
	return s.settings.Delete(ctx, organizationID, settingID)
}

// --- Internal helpers ---

// findActiveSetting tries to find an active invoice setting for the organization.
// It checks national first, then international.
func (s *InvoiceService) findActiveSetting(ctx context.Context, organizationID uuid.UUID) (*domain.InvoiceSetting, error) {
	setting, err := s.settings.GetByOrgAndType(ctx, organizationID, domain.InvTypeNational)
	if err == nil && setting.IsActive {
		return setting, nil
	}
	setting, err = s.settings.GetByOrgAndType(ctx, organizationID, domain.InvTypeInternational)
	if err == nil && setting.IsActive {
		return setting, nil
	}
	return nil, domain.ErrInvoiceSettingNotFound
}

// dispatchWebhook marshals the invoice and dispatches the webhook event.
// Failures are logged and do not affect the main operation.
func (s *InvoiceService) dispatchWebhook(ctx context.Context, organizationID uuid.UUID, eventType string, invoice *domain.Invoice) {
	if s.dispatcher == nil {
		return
	}
	payload, err := json.Marshal(invoice)
	if err != nil {
		slog.Warn("failed to marshal invoice for webhook", "event_type", eventType, "error", err)
		return
	}
	if err := s.dispatcher.Dispatch(ctx, organizationID, eventType, "production", payload); err != nil {
		slog.Warn("failed to dispatch invoice webhook", "event_type", eventType, "error", err)
	}
}
