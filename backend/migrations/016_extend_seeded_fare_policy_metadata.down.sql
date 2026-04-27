UPDATE fare_products
SET
    metadata = metadata - 'ac_multiplier' - 'express_multiplier' - 'transfer_policy',
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (
    'fare_product_dmrc_default',
    'fare_product_dimts_default',
    'fare_product_dtc_default'
);
