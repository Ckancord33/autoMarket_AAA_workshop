package models

import "time"

// Publication representa la entidad PUBLICATION en el sistema.
type Publication struct {
	IDVehicle    int        `json:"idVehicle"`
	IDUser       int        `json:"idUser"`
	Brand        string     `json:"brand"`
	Model        string     `json:"model"`
	Year         int        `json:"year"`
	Color        string     `json:"color"`
	Condition    string     `json:"condition"`
	Price        float64    `json:"price"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	IsAuthorized bool       `json:"isAuthorized"`
	PublishedAt  time.Time  `json:"publishedAt"`
	SoldAt       *time.Time `json:"soldAt,omitempty"`
	Mileage      int        `json:"mileage"`
}
