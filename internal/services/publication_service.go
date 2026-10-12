package services

import (
	"automarket/internal/models"
)

// DTOs para la transferencia de datos en el módulo de Publicaciones
type CreatePublicationRequest struct {
	Brand       string  `json:"brand"`
	Model       string  `json:"model"`
	Year        int     `json:"year"`
	Color       string  `json:"color"`
	Condition   string  `json:"condition"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Mileage     int     `json:"mileage"`
	ImageURL    string  `json:"imageUrl"`
}

type UpdatePublicationRequest struct {
	Price       *float64 `json:"price"`
	Description *string  `json:"description"`
	Status      *string  `json:"status"`
	ImageURL    *string  `json:"imageUrl"`
}

type PublicationFilterRequest struct {
	Brand string `form:"brand"`
	Model string `form:"model"`
}

// PublicationService define el contrato de métodos del servicio que invoca el Handler
type PublicationService interface {
	Create(req CreatePublicationRequest) (*models.Publication, error)
	GetByID(id string) (*models.Publication, error)
	GetAll(filters PublicationFilterRequest) ([]models.Publication, error)
	Update(id string, req UpdatePublicationRequest) (*models.Publication, error)
}
