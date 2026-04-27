package repository

import (
	"database/sql"
	"fmt"

	"indian-transit-backend/internal/database"
	"indian-transit-backend/internal/models"
)

type FareRepository struct {
	db *database.DB
}

func NewFareRepository(db *database.DB) *FareRepository {
	return &FareRepository{db: db}
}

func (r *FareRepository) GetAgencyIDByRouteID(routeID string) (string, error) {
	query := `SELECT agency_id FROM routes WHERE route_id = ?`
	var agencyID string
	if err := r.db.QueryRow(query, routeID).Scan(&agencyID); err != nil {
		return "", err
	}
	return agencyID, nil
}

func (r *FareRepository) GetStopCoordinates(stopID string) (float64, float64, error) {
	query := `SELECT stop_lat, stop_lon FROM stops WHERE stop_id = ?`
	var lat, lon float64
	if err := r.db.QueryRow(query, stopID).Scan(&lat, &lon); err != nil {
		return 0, 0, err
	}
	return lat, lon, nil
}

func (r *FareRepository) GetRouteNames(routeID string) (string, string, error) {
	query := `SELECT route_short_name, route_long_name FROM routes WHERE route_id = ?`
	var shortName, longName string
	if err := r.db.QueryRow(query, routeID).Scan(&shortName, &longName); err != nil {
		return "", "", fmt.Errorf("get route names: %w", err)
	}
	return shortName, longName, nil
}

func (r *FareRepository) GetActiveFareProductByAgencyID(agencyID string) (*models.FareProduct, error) {
	query := `SELECT id, agency_id, product_type, name, description, currency_code, base_fare, fare_per_km, fare_per_stop, transfer_fee, rule_version,
		valid_from, valid_until, metadata, created_at, updated_at
		FROM fare_products
		WHERE (agency_id = ? OR metadata->>'agency_hint' = ?)
		AND (valid_from IS NULL OR valid_from <= CURRENT_TIMESTAMP)
		AND (valid_until IS NULL OR valid_until >= CURRENT_TIMESTAMP)
		ORDER BY CASE WHEN agency_id = ? THEN 0 ELSE 1 END, valid_from DESC NULLS LAST
		LIMIT 1`

	product := &models.FareProduct{}
	var agencyRef, description sql.NullString
	var validFrom, validUntil sql.NullTime
	err := r.db.QueryRow(query, agencyID, agencyID, agencyID).Scan(
		&product.ID, &agencyRef, &product.ProductType, &product.Name, &description, &product.CurrencyCode,
		&product.BaseFare, &product.FarePerKM, &product.FarePerStop, &product.TransferFee, &product.RuleVersion,
		&validFrom, &validUntil, &product.Metadata, &product.CreatedAt, &product.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if agencyRef.Valid {
		product.AgencyID = &agencyRef.String
	}
	if description.Valid {
		product.Description = &description.String
	}
	if validFrom.Valid {
		t := validFrom.Time
		product.ValidFrom = &t
	}
	if validUntil.Valid {
		t := validUntil.Time
		product.ValidUntil = &t
	}
	return product, nil
}
