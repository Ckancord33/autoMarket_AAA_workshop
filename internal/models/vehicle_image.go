package models

import "time"

// VehicleImage representa la entidad VEHICLE_IMAGE asociada a una publicación.
type VehicleImage struct {
	IDImage   int       `json:"idImage"`
	IDVehicle int       `json:"idVehicle"`
	ImageURL  string    `json:"imageUrl"`
	IsPrimary bool      `json:"isPrimary"`
	CreatedAt time.Time `json:"createdAt"`
}
