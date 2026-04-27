ALTER TABLE fare_transactions
    ADD COLUMN IF NOT EXISTS reconciliation_status VARCHAR(50) NOT NULL DEFAULT 'unreconciled'
        CHECK (reconciliation_status IN ('unreconciled', 'reconciled', 'waived', 'reversed', 'failed')),
    ADD COLUMN IF NOT EXISTS payment_reference_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS reconciled_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS adjustment_reason TEXT,
    ADD COLUMN IF NOT EXISTS reversed_from_transaction_id VARCHAR(255);

ALTER TABLE fare_transactions
    DROP CONSTRAINT IF EXISTS fare_transactions_reversed_from_transaction_id_fkey;

ALTER TABLE fare_transactions
    ADD CONSTRAINT fare_transactions_reversed_from_transaction_id_fkey
    FOREIGN KEY (reversed_from_transaction_id) REFERENCES fare_transactions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fare_transactions_reconciliation_status
    ON fare_transactions(reconciliation_status, charged_at);

INSERT INTO fare_products (
    id, agency_id, product_type, name, description, currency_code,
    base_fare, fare_per_km, fare_per_stop, transfer_fee, rule_version,
    valid_from, metadata
)
SELECT
    'fare_product_seeded_' || regexp_replace(lower(a.agency_id), '[^a-z0-9]+', '_', 'g') AS id,
    a.agency_id,
    'single_journey',
    a.agency_name || ' Seeded Default Fare',
    'Auto-seeded fare product derived from agency data.',
    'INR',
    CASE
        WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 10.00
        ELSE 5.00
    END,
    CASE
        WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 2.50
        ELSE 1.50
    END,
    0,
    CASE
        WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 0
        ELSE 2.00
    END,
    'seed-v2',
    CURRENT_TIMESTAMP,
    jsonb_build_object(
        'agency_hint', a.agency_id,
        'mode', CASE
            WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 'metro'
            ELSE 'bus'
        END,
        'ac_multiplier', CASE
            WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 1.0
            ELSE 1.5
        END,
        'express_multiplier', CASE
            WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 1.0
            ELSE 1.2
        END,
        'transfer_policy', CASE
            WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 'metro_internal_free'
            ELSE 'paid_transfer'
        END
    )
FROM agencies a
WHERE NOT EXISTS (
    SELECT 1
    FROM fare_products fp
    WHERE fp.agency_id = a.agency_id
);

INSERT INTO fare_capping_rules (
    id, agency_id, fare_product_id, cap_type, amount, currency_code, valid_from, metadata
)
SELECT
    'fare_cap_seeded_' || regexp_replace(lower(a.agency_id), '[^a-z0-9]+', '_', 'g') || '_daily' AS id,
    a.agency_id,
    fp.id,
    'daily',
    CASE
        WHEN lower(a.agency_id) LIKE '%metro%' OR lower(a.agency_name) LIKE '%metro%' OR lower(a.agency_id) = 'dmrc' THEN 120.00
        ELSE 60.00
    END,
    'INR',
    CURRENT_TIMESTAMP,
    jsonb_build_object('agency_hint', a.agency_id, 'seeded', true)
FROM agencies a
JOIN fare_products fp ON fp.agency_id = a.agency_id
WHERE NOT EXISTS (
    SELECT 1
    FROM fare_capping_rules fcr
    WHERE fcr.agency_id = a.agency_id
      AND fcr.cap_type = 'daily'
);
