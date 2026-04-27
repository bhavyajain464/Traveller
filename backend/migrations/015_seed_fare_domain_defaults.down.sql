DELETE FROM fare_capping_rules
WHERE id IN ('fare_cap_dmrc_daily', 'fare_cap_dimts_daily', 'fare_cap_dtc_daily');

DELETE FROM fare_zones
WHERE id IN ('fare_zone_dmrc_core', 'fare_zone_dimts_city', 'fare_zone_dtc_city');

DELETE FROM fare_products
WHERE id IN ('fare_product_dmrc_default', 'fare_product_dimts_default', 'fare_product_dtc_default');
