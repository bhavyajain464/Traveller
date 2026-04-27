DELETE FROM fare_capping_rules
WHERE id LIKE 'fare_cap_seeded_%_daily';

DELETE FROM fare_products
WHERE id LIKE 'fare_product_seeded_%';

DROP INDEX IF EXISTS idx_fare_transactions_reconciliation_status;

ALTER TABLE fare_transactions
    DROP CONSTRAINT IF EXISTS fare_transactions_reversed_from_transaction_id_fkey;

ALTER TABLE fare_transactions
    DROP COLUMN IF EXISTS reversed_from_transaction_id,
    DROP COLUMN IF EXISTS adjustment_reason,
    DROP COLUMN IF EXISTS reconciled_at,
    DROP COLUMN IF EXISTS payment_reference_id,
    DROP COLUMN IF EXISTS reconciliation_status;
