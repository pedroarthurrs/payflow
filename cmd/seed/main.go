// Package main provides a development-only database seeder.
// It generates realistic fake data using gofakeit and respects the full
// FK dependency tree of the payflow schema.
//
// Usage:
//
//	go run ./cmd/seed [flags]
//	make seed
//
// Flags:
//
//	-orgs                int   number of organizations (default 2)
//	-users-per-org       int   users per organization (default 3)
//	-customers-per-org   int   customers per organization (default 5)
//	-products-per-org    int   products per organization (default 3)
//	-plans-per-product   int   plans per product (default 2)
//	-subs-per-customer   int   subscriptions per customer (default 2)
//	-invoices-per-sub    int   invoices per subscription (default 2)
//	-payments-per-invoice int  payments per invoice (default 1)
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"payflow/internal/config"
)

// devPasswordHash is bcrypt(cost=10) of "Dev@123456".
const devPasswordHash = "$2a$10$U2ShqBynvPjAUiCnPTypnO6Hjs4IwsLU7dJAmcU.qQkUXfeWBXU1e"

// devUserID is the fixed UUID for the shared dev@payflow.local user.
var devUserID = uuid.MustParse("a0000001-0000-0000-0000-000000000001")

// seedConfig controls the volume of generated data.
type seedConfig struct {
	Orgs               int
	UsersPerOrg        int
	CustomersPerOrg    int
	ProductsPerOrg     int
	PlansPerProduct    int
	SubsPerCustomer    int
	InvoicesPerSub     int
	PaymentsPerInvoice int
}

func main() {
	cfg := config.Load()

	if cfg.IsProduction {
		slog.Error("seed must not run in production (GO_ENV=production detected)")
		os.Exit(1)
	}

	sc := &seedConfig{}
	flag.IntVar(&sc.Orgs, "orgs", 2, "number of organizations")
	flag.IntVar(&sc.UsersPerOrg, "users-per-org", 15, "users per organization")
	flag.IntVar(&sc.CustomersPerOrg, "customers-per-org", 50, "customers per organization")
	flag.IntVar(&sc.ProductsPerOrg, "products-per-org", 5, "products per organization")
	flag.IntVar(&sc.PlansPerProduct, "plans-per-product", 7, "plans per product")
	flag.IntVar(&sc.SubsPerCustomer, "subs-per-customer", 3, "subscriptions per customer")
	flag.IntVar(&sc.InvoicesPerSub, "invoices-per-sub", 6, "invoices per subscription")
	flag.IntVar(&sc.PaymentsPerInvoice, "payments-per-invoice", 3, "payments per invoice")
	flag.Parse()

	fake := gofakeit.New(0)

	db, err := openDB(cfg)
	if err != nil {
		slog.Error("db connection failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	runErr := run(ctx, db, fake, sc)

	if cerr := db.Close(); cerr != nil {
		slog.Warn("failed to close database connection", "error", cerr)
	}

	if runErr != nil {
		slog.Error("seed failed", "error", runErr)
		os.Exit(1)
	}

	slog.Info("seed complete")
}

func run(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, sc *seedConfig) error {
	// 1. Organizations
	slog.Info("seeding organizations", "count", sc.Orgs)
	orgIDs, err := seedOrganizations(ctx, db, fake, sc.Orgs)
	if err != nil {
		return fmt.Errorf("organizations: %w", err)
	}

	// 2a. Fixed dev user (dev@payflow.local) — member of all orgs
	slog.Info("seeding dev user", "email", "dev@payflow.local")
	if err := seedDevUser(ctx, db, orgIDs); err != nil {
		return fmt.Errorf("dev user: %w", err)
	}

	// 2b. Random users + OrgMembers (depends on orgs)
	slog.Info("seeding users", "per_org", sc.UsersPerOrg)
	if err := seedUsers(ctx, db, fake, orgIDs, sc.UsersPerOrg); err != nil {
		return fmt.Errorf("users: %w", err)
	}

	// 3. Customers + Addresses (depends on orgs)
	slog.Info("seeding customers", "per_org", sc.CustomersPerOrg)
	customersByOrg, err := seedCustomers(ctx, db, fake, orgIDs, sc.CustomersPerOrg)
	if err != nil {
		return fmt.Errorf("customers: %w", err)
	}

	// 4. Products + Plans (depends on orgs)
	slog.Info("seeding products", "per_org", sc.ProductsPerOrg)
	plansByOrg, err := seedProductsAndPlans(ctx, db, fake, orgIDs, sc.ProductsPerOrg, sc.PlansPerProduct)
	if err != nil {
		return fmt.Errorf("products/plans: %w", err)
	}

	// 5. Subscriptions + Items (depends on customers + plans)
	slog.Info("seeding subscriptions", "per_customer", sc.SubsPerCustomer)
	subsByOrg, err := seedSubscriptions(ctx, db, fake, orgIDs, customersByOrg, plansByOrg, sc.SubsPerCustomer)
	if err != nil {
		return fmt.Errorf("subscriptions: %w", err)
	}

	// 6. Invoices + Items (depends on subscriptions + customers)
	slog.Info("seeding invoices", "per_sub", sc.InvoicesPerSub)
	invoicesByOrg, err := seedInvoices(ctx, db, fake, orgIDs, subsByOrg, customersByOrg, sc.InvoicesPerSub)
	if err != nil {
		return fmt.Errorf("invoices: %w", err)
	}

	// 7. Payments (depends on invoices + subscriptions)
	slog.Info("seeding payments", "per_invoice", sc.PaymentsPerInvoice)
	paymentsByOrg, err := seedPayments(ctx, db, fake, orgIDs, invoicesByOrg, sc.PaymentsPerInvoice)
	if err != nil {
		return fmt.Errorf("payments: %w", err)
	}

	// 8. Wallets — org wallet + customer wallets (depends on orgs + customers)
	slog.Info("seeding wallets")
	orgWallets, customerWallets, err := seedWallets(ctx, db, orgIDs, customersByOrg)
	if err != nil {
		return fmt.Errorf("wallets: %w", err)
	}

	// 9. Ledger entries (depends on wallets + payments)
	slog.Info("seeding ledger entries")
	if err := seedLedgerEntries(ctx, db, fake, orgIDs, orgWallets, paymentsByOrg, invoicesByOrg, subsByOrg); err != nil {
		return fmt.Errorf("ledger entries: %w", err)
	}

	// 10. Commission settings (depends on customers)
	slog.Info("seeding commission settings")
	if err := seedCommissionSettings(ctx, db, fake, orgIDs, customersByOrg, customerWallets); err != nil {
		return fmt.Errorf("commission settings: %w", err)
	}

	return nil
}

// ─── Organizations ────────────────────────────────────────────────────────────

func seedOrganizations(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, n int) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, n)
	for i := 0; i < n; i++ {
		id := uuid.New()
		name := fake.Company()
		_, err := db.ExecContext(ctx, `
			INSERT INTO organizations (id, name, status)
			VALUES ($1, $2, 'active')
		`, id, name)
		if err != nil {
			return nil, fmt.Errorf("insert org: %w", err)
		}

		// Organization config (1:1)
		_, err = db.ExecContext(ctx, `
			INSERT INTO organization_configs (id, organization_id, support_email, logo_url, primary_color)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (organization_id) DO NOTHING
		`, uuid.New(), id,
			fake.Email(),
			fmt.Sprintf("https://placehold.co/200x200?text=%s", string([]rune(name)[:1])),
			fake.HexColor(),
		)
		if err != nil {
			return nil, fmt.Errorf("insert org config: %w", err)
		}

		ids = append(ids, id)
		slog.Info("  org created", "id", id, "name", name)
	}
	return ids, nil
}

// ─── Fixed dev user ─────────────────────────────────────────────────────────

// seedDevUser inserts dev@payflow.local with a fixed UUID and adds it as a
// member of every generated organization: owner of the first, admin of the rest.
func seedDevUser(ctx context.Context, db *sql.DB, orgIDs []uuid.UUID) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO "user" (id, email, name, password)
		VALUES ($1, 'dev@payflow.local', 'Dev User', $2)
		ON CONFLICT (email) DO UPDATE SET
			name     = EXCLUDED.name,
			password = EXCLUDED.password,
			updated_at = NOW(),
			deleted_at = NULL
	`, devUserID, devPasswordHash)
	if err != nil {
		return fmt.Errorf("insert dev user: %w", err)
	}

	for i, orgID := range orgIDs {
		role := "admin"
		if i == 0 {
			role = "owner"
		}
		_, err = db.ExecContext(ctx, `
			INSERT INTO organization_members (id, organization_id, user_id, role)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (organization_id, user_id) DO UPDATE SET
				role = EXCLUDED.role,
				updated_at = NOW(),
				deleted_at = NULL
		`, uuid.New(), orgID, devUserID, role)
		if err != nil {
			return fmt.Errorf("insert dev user member (org %s): %w", orgID, err)
		}
		slog.Info("  dev user linked", "org", orgID, "role", role)
	}
	return nil
}

// ─── Users + OrgMembers ───────────────────────────────────────────────────────

func seedUsers(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, perOrg int) error {
	roles := []string{"admin", "developer", "support"}
	for _, orgID := range orgIDs {
		for i := range perOrg {
			userID := uuid.New()
			name := fake.Name()
			// unique email: username+random@domain
			email := fmt.Sprintf("%s+%s@%s", fake.Username(), uuid.New().String()[:8], fake.DomainName())

			_, err := db.ExecContext(ctx, `
				INSERT INTO "user" (id, email, name, password)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (email) DO NOTHING
			`, userID, email, name, devPasswordHash)
			if err != nil {
				return fmt.Errorf("insert user: %w", err)
			}

			role := "owner"
			if i > 0 {
				role = roles[i%len(roles)]
			}

			memberID := uuid.New()
			_, err = db.ExecContext(ctx, `
				INSERT INTO organization_members (id, organization_id, user_id, role)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (organization_id, user_id) DO NOTHING
			`, memberID, orgID, userID, role)
			if err != nil {
				return fmt.Errorf("insert member: %w", err)
			}
		}
	}
	return nil
}

// ─── Customers + Addresses ───────────────────────────────────────────────────

// customersByOrg maps orgID → list of customer UUIDs.
func seedCustomers(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, perOrg int) (map[uuid.UUID][]uuid.UUID, error) {
	result := make(map[uuid.UUID][]uuid.UUID, len(orgIDs))
	docTypes := []string{"cpf", "cnpj"}
	addressTypes := []string{"billing", "shipping", "home"}

	for _, orgID := range orgIDs {
		for i := range perOrg {
			custID := uuid.New()
			isCompany := i%3 == 0
			var name, document, docType string
			if isCompany {
				name = fake.Company()
				document = fake.Numerify("##############") // 14 digits = CNPJ
				docType = docTypes[1]
			} else {
				name = fake.Name()
				document = fake.Numerify("###########") // 11 digits = CPF
				docType = docTypes[0]
			}
			email := fmt.Sprintf("%s+%s@%s", fake.Username(), uuid.New().String()[:8], fake.DomainName())

			_, err := db.ExecContext(ctx, `
				INSERT INTO customers (id, organization_id, name, email, phone, document, document_type, status)
				VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
			`, custID, orgID, name, email, fake.Phone(), document, docType)
			if err != nil {
				return nil, fmt.Errorf("insert customer: %w", err)
			}

			// 1-2 addresses per customer
			addrCount := 1 + (i % 2)
			for j := range addrCount {
				addrType := addressTypes[j%len(addressTypes)]
				isMain := j == 0
				_, err = db.ExecContext(ctx, `
					INSERT INTO addresses (id, organization_id, customer_id, type, postal_code, street, number, complement, neighborhood, city, state, country, is_main)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'BR', $12)
				`, uuid.New(), orgID, custID,
					addrType,
					fake.Zip(),
					fake.StreetName(),
					fake.StreetNumber(),
					nullableString(fake, false),
					fake.AppName(),
					fake.City(),
					fake.StateAbr(),
					isMain,
				)
				if err != nil {
					return nil, fmt.Errorf("insert address: %w", err)
				}
			}

			result[orgID] = append(result[orgID], custID)
		}
	}
	return result, nil
}

// ─── Products + Plans ────────────────────────────────────────────────────────

// plansByOrg maps orgID → list of plan UUIDs.
func seedProductsAndPlans(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, productsPerOrg, plansPerProduct int) (map[uuid.UUID][]uuid.UUID, error) {
	intervals := []string{"month", "year"}
	plansByOrg := make(map[uuid.UUID][]uuid.UUID, len(orgIDs))

	for _, orgID := range orgIDs {
		for range productsPerOrg {
			productID := uuid.New()
			productName := fmt.Sprintf("%s %s", fake.AppName(), fake.BuzzWord())

			_, err := db.ExecContext(ctx, `
				INSERT INTO products (id, organization_id, name, active)
				VALUES ($1, $2, $3, true)
			`, productID, orgID, productName)
			if err != nil {
				return nil, fmt.Errorf("insert product: %w", err)
			}

			for j := range plansPerProduct {
				planID := uuid.New()
				interval := intervals[j%len(intervals)]
				intervalCount := int32(1)
				amountCents := int32(fake.IntRange(990, 49900))
				planName := fmt.Sprintf("%s %s", productName, intervalLabel(interval))
				slug := fmt.Sprintf("%s-%s", fake.LoremIpsumWord(), fake.LoremIpsumWord())

				_, err = db.ExecContext(ctx, `
					INSERT INTO plans (id, organization_id, product_id, name, slug, amount_cents, currency, interval, interval_count, active)
					VALUES ($1, $2, $3, $4, $5, $6, 'BRL', $7, $8, true)
				`, planID, orgID, productID, planName, slug, amountCents, interval, intervalCount)
				if err != nil {
					return nil, fmt.Errorf("insert plan: %w", err)
				}

				plansByOrg[orgID] = append(plansByOrg[orgID], planID)
			}
		}
	}
	return plansByOrg, nil
}

// ─── Subscriptions + Items ───────────────────────────────────────────────────

type subscriptionRow struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	CustomerID  uuid.UUID
	PlanID      uuid.UUID
	AmountCents int32
}

// subsByOrg maps orgID → list of subscriptionRows.
func seedSubscriptions(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, customersByOrg, plansByOrg map[uuid.UUID][]uuid.UUID, perCustomer int) (map[uuid.UUID][]subscriptionRow, error) {
	statuses := []string{"active", "active", "active", "trialing", "past_due", "canceled"}
	methods := []string{"credit_card", "pix", "boleto", "bank_transfer"}
	result := make(map[uuid.UUID][]subscriptionRow)

	for _, orgID := range orgIDs {
		customers := customersByOrg[orgID]
		plans := plansByOrg[orgID]
		if len(plans) == 0 {
			continue
		}

		for _, custID := range customers {
			for range perCustomer {
				subID := uuid.New()
				planID := plans[fake.IntRange(0, len(plans)-1)]
				status := statuses[fake.IntRange(0, len(statuses)-1)]
				method := methods[fake.IntRange(0, len(methods)-1)]

				periodStart := time.Now().AddDate(0, 0, -fake.IntRange(1, 30))
				periodEnd := periodStart.AddDate(0, 1, 0)

				_, err := db.ExecContext(ctx, `
					INSERT INTO subscriptions (id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, payment_method)
					VALUES ($1, $2, $3, $4, $5, $6, false, $7)
				`, subID, orgID, custID, status, periodStart, periodEnd, method)
				if err != nil {
					return nil, fmt.Errorf("insert subscription: %w", err)
				}

				quantity := int32(1)
				_, err = db.ExecContext(ctx, `
					INSERT INTO subscription_items (id, organization_id, subscription_id, plan_id, quantity)
					VALUES ($1, $2, $3, $4, $5)
				`, uuid.New(), orgID, subID, planID, quantity)
				if err != nil {
					return nil, fmt.Errorf("insert subscription item: %w", err)
				}

				// Retrieve plan amount_cents for use in invoices/payments
				var amountCents int32
				err = db.QueryRowContext(ctx, `SELECT amount_cents FROM plans WHERE id = $1`, planID).Scan(&amountCents)
				if err != nil {
					amountCents = int32(fake.IntRange(990, 49900))
				}

				result[orgID] = append(result[orgID], subscriptionRow{
					ID: subID, OrgID: orgID, CustomerID: custID,
					PlanID: planID, AmountCents: amountCents,
				})
			}
		}
	}
	return result, nil
}

// ─── Invoices + Items ────────────────────────────────────────────────────────

type invoiceRow struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	CustomerID  uuid.UUID
	SubID       uuid.UUID
	AmountCents int32
}

func seedInvoices(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, subsByOrg map[uuid.UUID][]subscriptionRow, customersByOrg map[uuid.UUID][]uuid.UUID, perSub int) (map[uuid.UUID][]invoiceRow, error) {
	statuses := []string{"paid", "paid", "pending", "canceled"}
	result := make(map[uuid.UUID][]invoiceRow)

	for _, orgID := range orgIDs {
		subs := subsByOrg[orgID]
		for _, sub := range subs {
			for i := 0; i < perSub; i++ {
				invoiceID := uuid.New()
				status := statuses[fake.IntRange(0, len(statuses)-1)]
				amountCents := sub.AmountCents

				_, err := db.ExecContext(ctx, `
					INSERT INTO invoices (id, organization_id, customer_id, subscription_id, invoice_type, operation_type, amount_cents, currency, status)
					VALUES ($1, $2, $3, $4, 'national', 1, $5, 'BRL', $6)
				`, invoiceID, orgID, sub.CustomerID, sub.ID, amountCents, status)
				if err != nil {
					return nil, fmt.Errorf("insert invoice: %w", err)
				}

				// 1-3 line items per invoice
				itemCount := fake.IntRange(1, 3)
				for j := 0; j < itemCount; j++ {
					qty := int32(fake.IntRange(1, 5))
					unitPrice := amountCents / int32(itemCount)
					if unitPrice == 0 {
						unitPrice = 100
					}
					totalPrice := unitPrice * qty

					_, err = db.ExecContext(ctx, `
						INSERT INTO invoice_items (id, organization_id, invoice_id, description, quantity, unit_price_cents, total_price_cents)
						VALUES ($1, $2, $3, $4, $5, $6, $7)
					`, uuid.New(), orgID, invoiceID,
						fake.ProductName(),
						fmt.Sprintf("%d", qty),
						unitPrice,
						totalPrice,
					)
					if err != nil {
						return nil, fmt.Errorf("insert invoice item: %w", err)
					}
				}

				result[orgID] = append(result[orgID], invoiceRow{
					ID: invoiceID, OrgID: orgID, CustomerID: sub.CustomerID,
					SubID: sub.ID, AmountCents: amountCents,
				})
			}
		}
	}
	return result, nil
}

// ─── Payments ────────────────────────────────────────────────────────────────

type paymentRow struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	CustomerID  uuid.UUID
	InvoiceID   uuid.UUID
	AmountCents int32
}

func seedPayments(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, invoicesByOrg map[uuid.UUID][]invoiceRow, perInvoice int) (map[uuid.UUID][]paymentRow, error) {
	statuses := []string{"succeeded", "succeeded", "succeeded", "pending", "failed"}
	methods := []string{"credit_card", "pix", "boleto", "bank_transfer"}
	gateways := []string{"stripe", "pagarme", "cielo"}
	result := make(map[uuid.UUID][]paymentRow)

	for _, orgID := range orgIDs {
		invoices := invoicesByOrg[orgID]
		for _, inv := range invoices {
			for range perInvoice {
				payID := uuid.New()
				status := statuses[fake.IntRange(0, len(statuses)-1)]
				method := methods[fake.IntRange(0, len(methods)-1)]
				gateway := gateways[fake.IntRange(0, len(gateways)-1)]
				gatewayRef := fake.UUID()
				idempotencyKey := fake.UUID()

				_, err := db.ExecContext(ctx, `
					INSERT INTO payments (
						id, organization_id, customer_id, invoice_id, subscription_id,
						payment_method, amount_cents, status, currency,
						gateway_name, gateway_reference, idempotency_key
					)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'BRL', $9, $10, $11)
				`, payID, orgID, inv.CustomerID, inv.ID, inv.SubID,
					method, inv.AmountCents, status,
					gateway, gatewayRef, idempotencyKey,
				)
				if err != nil {
					return nil, fmt.Errorf("insert payment: %w", err)
				}

				// Payment history entry
				_, err = db.ExecContext(ctx, `
					INSERT INTO payment_status_history (id, organization_id, payment_id, old_status, new_status, event_source, reason)
					VALUES ($1, $2, $3, NULL, $4, 'seed', 'initial seed')
				`, uuid.New(), orgID, payID, status)
				if err != nil {
					return nil, fmt.Errorf("insert payment history: %w", err)
				}

				result[orgID] = append(result[orgID], paymentRow{
					ID: payID, OrgID: orgID, CustomerID: inv.CustomerID,
					InvoiceID: inv.ID, AmountCents: inv.AmountCents,
				})
			}
		}
	}
	return result, nil
}

// ─── Wallets ─────────────────────────────────────────────────────────────────

func seedWallets(ctx context.Context, db *sql.DB, orgIDs []uuid.UUID, customersByOrg map[uuid.UUID][]uuid.UUID) (
	orgWallets map[uuid.UUID]uuid.UUID,
	customerWallets map[uuid.UUID]uuid.UUID,
	err error,
) {
	orgWallets = make(map[uuid.UUID]uuid.UUID)
	customerWallets = make(map[uuid.UUID]uuid.UUID)

	for _, orgID := range orgIDs {
		// Organization-owned wallet
		walletID := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO wallets (id, organization_id, organization_owner_id, customer_owner_id, currency)
			VALUES ($1, $2, $3, NULL, 'BRL')
		`, walletID, orgID, orgID)
		if err != nil {
			return nil, nil, fmt.Errorf("insert org wallet: %w", err)
		}
		orgWallets[orgID] = walletID

		// Per-customer wallets
		for _, custID := range customersByOrg[orgID] {
			cwID := uuid.New()
			_, err = db.ExecContext(ctx, `
				INSERT INTO wallets (id, organization_id, organization_owner_id, customer_owner_id, currency)
				VALUES ($1, $2, NULL, $3, 'BRL')
			`, cwID, orgID, custID)
			if err != nil {
				return nil, nil, fmt.Errorf("insert customer wallet: %w", err)
			}
			customerWallets[custID] = cwID
		}
	}
	return orgWallets, customerWallets, nil
}

// ─── Ledger Entries ───────────────────────────────────────────────────────────

func seedLedgerEntries(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, orgWallets map[uuid.UUID]uuid.UUID, paymentsByOrg map[uuid.UUID][]paymentRow, invoicesByOrg map[uuid.UUID][]invoiceRow, subsByOrg map[uuid.UUID][]subscriptionRow) error {
	categories := []string{"payment_split", "commission", "manual_adjustment"}

	for _, orgID := range orgIDs {
		walletID, ok := orgWallets[orgID]
		if !ok {
			continue
		}

		balance := int32(0)
		for _, pay := range paymentsByOrg[orgID] {
			// Credit entry for each succeeded payment
			category := categories[fake.IntRange(0, len(categories)-1)]
			balance += pay.AmountCents

			var invoiceNullUUID *uuid.UUID
			if pay.InvoiceID != uuid.Nil {
				invoiceNullUUID = &pay.InvoiceID
			}

			var subID *uuid.UUID
			subs := subsByOrg[orgID]
			for _, s := range subs {
				if s.CustomerID == pay.CustomerID {
					sid := s.ID
					subID = &sid
					break
				}
			}

			_, err := db.ExecContext(ctx, `
				INSERT INTO ledger_entries (id, organization_id, wallet_id, type, amount_cents, category, balance_after, subscription_id, invoice_id, payment_id)
				VALUES ($1, $2, $3, 'credit', $4, $5, $6, $7, $8, $9)
			`, uuid.New(), orgID, walletID,
				pay.AmountCents,
				category,
				balance,
				uuidOrNil(subID),
				uuidOrNil(invoiceNullUUID),
				pay.ID,
			)
			if err != nil {
				return fmt.Errorf("insert ledger entry: %w", err)
			}
		}

		// Update wallet balance to reflect entries
		if balance > 0 {
			_, err := db.ExecContext(ctx, `
				UPDATE wallets SET balance = $1 WHERE id = $2
			`, balance, walletID)
			if err != nil {
				return fmt.Errorf("update wallet balance: %w", err)
			}
		}
	}
	return nil
}

// ─── Commission Settings ──────────────────────────────────────────────────────

func seedCommissionSettings(ctx context.Context, db *sql.DB, fake *gofakeit.Faker, orgIDs []uuid.UUID, customersByOrg map[uuid.UUID][]uuid.UUID, customerWallets map[uuid.UUID]uuid.UUID) error {
	types := []string{"percentage", "fixed"}

	for _, orgID := range orgIDs {
		customers := customersByOrg[orgID]
		// Create a commission setting for ~half the customers
		for i, custID := range customers {
			if i%2 != 0 {
				continue
			}
			commType := types[i%len(types)]
			var value string
			if commType == "percentage" {
				value = fmt.Sprintf("%.2f", fake.Float64Range(1, 15))
			} else {
				value = fmt.Sprintf("%.2f", float64(fake.IntRange(100, 5000)))
			}

			_, err := db.ExecContext(ctx, `
				INSERT INTO commission_settings (id, organization_id, target_customer_id, type, value, is_active)
				VALUES ($1, $2, $3, $4, $5, true)
				ON CONFLICT DO NOTHING
			`, uuid.New(), orgID, custID, commType, value)
			if err != nil {
				return fmt.Errorf("insert commission setting: %w", err)
			}
		}
	}
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func openDB(cfg *config.Config) (*sql.DB, error) {
	driver := cfg.DbDriver
	if driver == "" {
		driver = "postgres"
	}

	var connStr string
	if cfg.DbDSN != "" {
		connStr = cfg.DbDSN
	} else {
		sslmode := cfg.DbSSLMode
		if sslmode == "" {
			sslmode = "disable"
		}
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.DbUser, cfg.DbPassword, cfg.DbHost, cfg.DbPort, cfg.DbName, sslmode)
	}

	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

func intervalLabel(interval string) string {
	switch interval {
	case "year":
		return "Annual"
	case "month":
		return "Mensal"
	default:
		return interval
	}
}

// nullableString returns a sql.NullString. prob controls whether the value is NULL.
func nullableString(fake *gofakeit.Faker, alwaysNull bool) interface{} {
	if alwaysNull || fake.IntRange(0, 3) == 0 {
		return nil
	}
	return fake.LoremIpsumWord()
}

// uuidOrNil returns the uuid.UUID value or nil for SQL.
func uuidOrNil(id *uuid.UUID) interface{} {
	if id == nil {
		return nil
	}
	return *id
}
