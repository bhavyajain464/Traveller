INSERT INTO fare_products (
    id, agency_id, product_type, name, description, currency_code,
    base_fare, fare_per_km, fare_per_stop, transfer_fee, rule_version,
    valid_from, metadata
)
VALUES
(
    'fare_product_dmrc_default',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DMRC' LIMIT 1),
    'single_journey',
    'DMRC Default Fare',
    'Seeded baseline fare product for Delhi Metro style pricing.',
    'INR',
    10.00, 2.50, 0, 0, 'seed-v1',
    CURRENT_TIMESTAMP,
    '{"agency_hint":"DMRC","mode":"metro"}'::jsonb
),
(
    'fare_product_dimts_default',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DIMTS' LIMIT 1),
    'single_journey',
    'DIMTS Default Fare',
    'Seeded baseline fare product for DIMTS bus pricing.',
    'INR',
    5.00, 1.50, 0, 2.00, 'seed-v1',
    CURRENT_TIMESTAMP,
    '{"agency_hint":"DIMTS","mode":"bus"}'::jsonb
),
(
    'fare_product_dtc_default',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DTC' LIMIT 1),
    'single_journey',
    'DTC Default Fare',
    'Seeded baseline fare product for DTC bus pricing.',
    'INR',
    5.00, 1.50, 0, 2.00, 'seed-v1',
    CURRENT_TIMESTAMP,
    '{"agency_hint":"DTC","mode":"bus"}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    agency_id = EXCLUDED.agency_id,
    product_type = EXCLUDED.product_type,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    currency_code = EXCLUDED.currency_code,
    base_fare = EXCLUDED.base_fare,
    fare_per_km = EXCLUDED.fare_per_km,
    fare_per_stop = EXCLUDED.fare_per_stop,
    transfer_fee = EXCLUDED.transfer_fee,
    rule_version = EXCLUDED.rule_version,
    valid_from = EXCLUDED.valid_from,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO fare_zones (id, agency_id, zone_code, name, metadata)
VALUES
(
    'fare_zone_dmrc_core',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DMRC' LIMIT 1),
    'DMRC_CORE',
    'DMRC Core Zone',
    '{"agency_hint":"DMRC"}'::jsonb
),
(
    'fare_zone_dimts_city',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DIMTS' LIMIT 1),
    'DIMTS_CITY',
    'DIMTS City Zone',
    '{"agency_hint":"DIMTS"}'::jsonb
),
(
    'fare_zone_dtc_city',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DTC' LIMIT 1),
    'DTC_CITY',
    'DTC City Zone',
    '{"agency_hint":"DTC"}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    agency_id = EXCLUDED.agency_id,
    zone_code = EXCLUDED.zone_code,
    name = EXCLUDED.name,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO fare_capping_rules (
    id, agency_id, fare_product_id, cap_type, amount, currency_code, valid_from, metadata
)
VALUES
(
    'fare_cap_dmrc_daily',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DMRC' LIMIT 1),
    'fare_product_dmrc_default',
    'daily',
    120.00,
    'INR',
    CURRENT_TIMESTAMP,
    '{"agency_hint":"DMRC"}'::jsonb
),
(
    'fare_cap_dimts_daily',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DIMTS' LIMIT 1),
    'fare_product_dimts_default',
    'daily',
    60.00,
    'INR',
    CURRENT_TIMESTAMP,
    '{"agency_hint":"DIMTS"}'::jsonb
),
(
    'fare_cap_dtc_daily',
    (SELECT agency_id FROM agencies WHERE agency_id = 'DTC' LIMIT 1),
    'fare_product_dtc_default',
    'daily',
    60.00,
    'INR',
    CURRENT_TIMESTAMP,
    '{"agency_hint":"DTC"}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    agency_id = EXCLUDED.agency_id,
    fare_product_id = EXCLUDED.fare_product_id,
    cap_type = EXCLUDED.cap_type,
    amount = EXCLUDED.amount,
    currency_code = EXCLUDED.currency_code,
    valid_from = EXCLUDED.valid_from,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP;
