-- +goose Up
ALTER TABLE invoices
    ADD COLUMN stripe_payment_intent_id TEXT,
    ADD COLUMN stripe_charge_id         TEXT,
    ADD COLUMN amount_paid_cents        BIGINT;

CREATE UNIQUE INDEX idx_invoices_payment_intent
    ON invoices(stripe_payment_intent_id)
    WHERE stripe_payment_intent_id IS NOT NULL;

-- +goose Down
ALTER TABLE invoices
    DROP COLUMN stripe_payment_intent_id,
    DROP COLUMN stripe_charge_id,
    DROP COLUMN amount_paid_cents;
