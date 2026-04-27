package models

import "time"

type FareProduct struct {
	ID           string     `json:"id" db:"id"`
	AgencyID     *string    `json:"agency_id,omitempty" db:"agency_id"`
	ProductType  string     `json:"product_type" db:"product_type"`
	Name         string     `json:"name" db:"name"`
	Description  *string    `json:"description,omitempty" db:"description"`
	CurrencyCode string     `json:"currency_code" db:"currency_code"`
	BaseFare     float64    `json:"base_fare" db:"base_fare"`
	FarePerKM    float64    `json:"fare_per_km" db:"fare_per_km"`
	FarePerStop  float64    `json:"fare_per_stop" db:"fare_per_stop"`
	TransferFee  float64    `json:"transfer_fee" db:"transfer_fee"`
	RuleVersion  string     `json:"rule_version" db:"rule_version"`
	ValidFrom    *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidUntil   *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	Metadata     string     `json:"metadata" db:"metadata"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type FareZone struct {
	ID           string    `json:"id" db:"id"`
	AgencyID     *string   `json:"agency_id,omitempty" db:"agency_id"`
	ZoneCode     string    `json:"zone_code" db:"zone_code"`
	Name         string    `json:"name" db:"name"`
	ParentZoneID *string   `json:"parent_zone_id,omitempty" db:"parent_zone_id"`
	Metadata     string    `json:"metadata" db:"metadata"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type StopZone struct {
	StopID         string     `json:"stop_id" db:"stop_id"`
	ZoneID         string     `json:"zone_id" db:"zone_id"`
	Priority       int        `json:"priority" db:"priority"`
	EffectiveFrom  *time.Time `json:"effective_from,omitempty" db:"effective_from"`
	EffectiveUntil *time.Time `json:"effective_until,omitempty" db:"effective_until"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

type UserEntitlement struct {
	ID              string     `json:"id" db:"id"`
	UserID          string     `json:"user_id" db:"user_id"`
	FareProductID   *string    `json:"fare_product_id,omitempty" db:"fare_product_id"`
	EntitlementType string     `json:"entitlement_type" db:"entitlement_type"`
	ReferenceID     *string    `json:"reference_id,omitempty" db:"reference_id"`
	Status          string     `json:"status" db:"status"`
	StartsAt        time.Time  `json:"starts_at" db:"starts_at"`
	EndsAt          *time.Time `json:"ends_at,omitempty" db:"ends_at"`
	Metadata        string     `json:"metadata" db:"metadata"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type FareCappingRule struct {
	ID            string     `json:"id" db:"id"`
	AgencyID      *string    `json:"agency_id,omitempty" db:"agency_id"`
	FareProductID *string    `json:"fare_product_id,omitempty" db:"fare_product_id"`
	CapType       string     `json:"cap_type" db:"cap_type"`
	Amount        float64    `json:"amount" db:"amount"`
	CurrencyCode  string     `json:"currency_code" db:"currency_code"`
	ValidFrom     *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidUntil    *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	Metadata      string     `json:"metadata" db:"metadata"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type FareTransaction struct {
	ID                        string     `json:"id" db:"id"`
	UserID                    string     `json:"user_id" db:"user_id"`
	SessionID                 *string    `json:"session_id,omitempty" db:"session_id"`
	SegmentID                 *string    `json:"segment_id,omitempty" db:"segment_id"`
	RouteBoardingID           *string    `json:"route_boarding_id,omitempty" db:"route_boarding_id"`
	FareProductID             *string    `json:"fare_product_id,omitempty" db:"fare_product_id"`
	CappingRuleID             *string    `json:"capping_rule_id,omitempty" db:"capping_rule_id"`
	Amount                    float64    `json:"amount" db:"amount"`
	CurrencyCode              string     `json:"currency_code" db:"currency_code"`
	RuleVersion               string     `json:"rule_version" db:"rule_version"`
	Status                    string     `json:"status" db:"status"`
	ReconciliationStatus      string     `json:"reconciliation_status" db:"reconciliation_status"`
	PaymentReferenceID        *string    `json:"payment_reference_id,omitempty" db:"payment_reference_id"`
	ChargedAt                 time.Time  `json:"charged_at" db:"charged_at"`
	ReconciledAt              *time.Time `json:"reconciled_at,omitempty" db:"reconciled_at"`
	AdjustmentReason          *string    `json:"adjustment_reason,omitempty" db:"adjustment_reason"`
	ReversedFromTransactionID *string    `json:"reversed_from_transaction_id,omitempty" db:"reversed_from_transaction_id"`
	Metadata                  string     `json:"metadata" db:"metadata"`
	CreatedAt                 time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at" db:"updated_at"`
}
