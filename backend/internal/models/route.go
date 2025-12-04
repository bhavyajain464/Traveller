package models

type Route struct {
	ID          string `json:"id" db:"route_id"`
	AgencyID    string `json:"agency_id" db:"agency_id"`
	ShortName   string `json:"short_name" db:"route_short_name"`
	LongName    string `json:"long_name" db:"route_long_name"`
	Type        int    `json:"type" db:"route_type"`
	Description string `json:"description" db:"route_desc"`
	URL         string `json:"url" db:"route_url"`
	Color       string `json:"color" db:"route_color"`
	TextColor   string `json:"text_color" db:"route_text_color"`
}
