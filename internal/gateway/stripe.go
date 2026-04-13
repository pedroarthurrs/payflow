package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"i9pay/internal/domain"
)

const stripeBaseURL = "https://api.stripe.com/v1"

// StripeAdapter implements domain.Gateway using the Stripe REST API.
type StripeAdapter struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewStripeAdapter creates a StripeAdapter with the given API key.
func NewStripeAdapter(apiKey string) *StripeAdapter {
	return &StripeAdapter{
		apiKey:  apiKey,
		baseURL: stripeBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// compile-time assertion
var _ domain.Gateway = (*StripeAdapter)(nil)

// stripeError represents a Stripe API error response.
type stripeError struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// stripePaymentIntent represents a Stripe PaymentIntent response.
type stripePaymentIntent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// stripeRefund represents a Stripe Refund response.
type stripeRefund struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Charge creates a Stripe PaymentIntent and confirms it immediately.
func (a *StripeAdapter) Charge(ctx context.Context, req domain.GatewayRequest) (*domain.ChargeResult, error) {
	params := url.Values{}
	params.Set("amount", fmt.Sprintf("%d", req.AmountCents))
	params.Set("currency", strings.ToLower(req.Currency))
	params.Set("payment_method", req.PaymentMethod)
	params.Set("confirm", "true")
	params.Set("metadata[payment_id]", req.PaymentID.String())
	params.Set("metadata[customer_id]", req.CustomerID.String())

	body, statusCode, err := a.post(ctx, "/payment_intents", params)
	if err != nil {
		return nil, domain.ErrGatewayUnavailable
	}

	if statusCode >= 400 {
		msg := extractStripeError(body)
		return nil, fmt.Errorf("%w: %s", domain.ErrGatewayChargeFailed, msg)
	}

	var pi stripePaymentIntent
	if err := json.Unmarshal(body, &pi); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response", domain.ErrGatewayChargeFailed)
	}

	status := stripeIntentStatusToDomain(pi.Status)
	return &domain.ChargeResult{
		GatewayReference: pi.ID,
		GatewayName:      "stripe",
		Status:           status,
	}, nil
}

// Refund creates a Stripe Refund for the given PaymentIntent.
func (a *StripeAdapter) Refund(ctx context.Context, req domain.GatewayRequest) (*domain.RefundResult, error) {
	params := url.Values{}
	params.Set("payment_intent", req.GatewayReference)

	body, statusCode, err := a.post(ctx, "/refunds", params)
	if err != nil {
		return nil, domain.ErrGatewayUnavailable
	}

	if statusCode >= 400 {
		msg := extractStripeError(body)
		return nil, fmt.Errorf("%w: %s", domain.ErrGatewayRefundFailed, msg)
	}

	var refund stripeRefund
	if err := json.Unmarshal(body, &refund); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response", domain.ErrGatewayRefundFailed)
	}

	return &domain.RefundResult{
		GatewayReference: refund.ID,
		Status:           domain.StatusRefunded,
	}, nil
}

// Cancel cancels a Stripe PaymentIntent.
func (a *StripeAdapter) Cancel(ctx context.Context, req domain.GatewayRequest) (*domain.ChargeResult, error) {
	path := fmt.Sprintf("/payment_intents/%s/cancel", req.GatewayReference)

	body, statusCode, err := a.post(ctx, path, url.Values{})
	if err != nil {
		return nil, domain.ErrGatewayUnavailable
	}

	if statusCode >= 400 {
		msg := extractStripeError(body)
		return nil, fmt.Errorf("%w: %s", domain.ErrGatewayCancelFailed, msg)
	}

	var pi stripePaymentIntent
	if err := json.Unmarshal(body, &pi); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response", domain.ErrGatewayCancelFailed)
	}

	return &domain.ChargeResult{
		GatewayReference: pi.ID,
		GatewayName:      "stripe",
		Status:           domain.StatusCanceled,
	}, nil
}

// post sends a POST request to the Stripe API and returns the raw body and status code.
func (a *StripeAdapter) post(ctx context.Context, path string, params url.Values) (body []byte, statusCode int, err error) {
	reqURL := a.baseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.SetBasicAuth(a.apiKey, "")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

// extractStripeError pulls the error message from a Stripe error response body.
func extractStripeError(body []byte) string {
	var se stripeError
	if err := json.Unmarshal(body, &se); err == nil && se.Error.Message != "" {
		return se.Error.Message
	}
	return "unknown error"
}

// stripeIntentStatusToDomain maps Stripe PaymentIntent statuses to domain statuses.
func stripeIntentStatusToDomain(status string) domain.PaymentStatus {
	switch status {
	case "requires_capture":
		return domain.StatusAuthorized
	case "succeeded":
		return domain.StatusSucceeded
	default:
		return domain.StatusAuthorized
	}
}
