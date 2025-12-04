package models

type Calendar struct {
	ServiceID string `json:"service_id" db:"service_id"`
	Monday    int    `json:"monday" db:"monday"`
	Tuesday   int    `json:"tuesday" db:"tuesday"`
	Wednesday int    `json:"wednesday" db:"wednesday"`
	Thursday  int    `json:"thursday" db:"thursday"`
	Friday    int    `json:"friday" db:"friday"`
	Saturday  int    `json:"saturday" db:"saturday"`
	Sunday    int    `json:"sunday" db:"sunday"`
	StartDate string `json:"start_date" db:"start_date"`
	EndDate   string `json:"end_date" db:"end_date"`
}


