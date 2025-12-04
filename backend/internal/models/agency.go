package models

type Agency struct {
	ID       string `json:"id" db:"agency_id"`
	Name     string `json:"name" db:"agency_name"`
	URL      string `json:"url" db:"agency_url"`
	Timezone string `json:"timezone" db:"agency_timezone"`
	Lang     string `json:"lang" db:"agency_lang"`
	Phone    string `json:"phone" db:"agency_phone"`
	FareURL  string `json:"fare_url" db:"agency_fare_url"`
}


