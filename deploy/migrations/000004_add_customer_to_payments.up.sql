-- Add customer_id to payments table for direct customer association.
ALTER TABLE payments
    ADD COLUMN customer_id UUID REFERENCES customers(id) ON DELETE SET NULL;

-- Index for listing payments by customer within an organization.
CREATE INDEX IF NOT EXISTS idx_payments_customer_id
    ON payments (organization_id, customer_id);
