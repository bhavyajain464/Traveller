UPDATE fare_products
SET
    metadata = jsonb_set(
        jsonb_set(
            jsonb_set(
                COALESCE(metadata, '{}'::jsonb),
                '{ac_multiplier}',
                '1.0'::jsonb,
                true
            ),
            '{express_multiplier}',
            '1.0'::jsonb,
            true
        ),
        '{transfer_policy}',
        '"metro_internal_free"'::jsonb,
        true
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'fare_product_dmrc_default';

UPDATE fare_products
SET
    metadata = jsonb_set(
        jsonb_set(
            jsonb_set(
                COALESCE(metadata, '{}'::jsonb),
                '{ac_multiplier}',
                '1.5'::jsonb,
                true
            ),
            '{express_multiplier}',
            '1.2'::jsonb,
            true
        ),
        '{transfer_policy}',
        '"paid_transfer"'::jsonb,
        true
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE id IN ('fare_product_dimts_default', 'fare_product_dtc_default');
