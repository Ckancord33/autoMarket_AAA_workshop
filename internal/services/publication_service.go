package services

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"automarket/internal/models"
	"automarket/internal/repositories"
)

// DTOs para la transferencia de datos en el módulo de Publicaciones
type CreatePublicationRequest struct {
	IDUser      int     `json:"idUser,omitempty"`
	Brand       string  `json:"brand"`
	Model       string  `json:"model"`
	Year        int     `json:"year"`
	Color       string  `json:"color"`
	Condition   string  `json:"condition"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Mileage     int      `json:"mileage"`
	Images      []string `json:"images"`
}

type UpdatePublicationRequest struct {
	Brand       *string   `json:"brand"`
	Model       *string   `json:"model"`
	Year        *int      `json:"year"`
	Color       *string   `json:"color"`
	Condition   *string   `json:"condition"`
	Price       *float64  `json:"price"`
	Description *string   `json:"description"`
	Mileage     *int      `json:"mileage"`
	Status      *string   `json:"status"`
	Images      *[]string `json:"images"`
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
	Authorize(id string) (*models.Publication, error)
	Delete(id string) error
}

// publicationService es la implementación concreta de PublicationService
type publicationService struct {
	repo *repositories.PublicationRepository
}

// NewPublicationService inyecta el repositorio en el servicio
func NewPublicationService(repo *repositories.PublicationRepository) PublicationService {
	return &publicationService{
		repo: repo,
	}
}

// Create valida las reglas del vehículo y lo persiste con estado "pendiente" para moderación
func (s *publicationService) Create(req CreatePublicationRequest) (*models.Publication, error) {
	brand := strings.TrimSpace(req.Brand)
	model := strings.TrimSpace(req.Model)

	if brand == "" {
		return nil, errors.New("la marca del vehículo es obligatoria")
	}
	if model == "" {
		return nil, errors.New("el modelo del vehículo es obligatorio")
	}
	if req.Year < 1900 || req.Year > time.Now().Year()+1 {
		return nil, errors.New("el año del vehículo no es válido")
	}
	if req.Price <= 0 {
		return nil, errors.New("el precio debe ser mayor a cero")
	}
	if req.Mileage < 0 {
		return nil, errors.New("el kilometraje no puede ser negativo")
	}

	idUser := req.IDUser
	if idUser <= 0 {
		idUser = 2 // Por defecto se asocia al vendedor demo del sistema
	}

	newPub := &models.Publication{
		IDUser:       idUser,
		Brand:        brand,
		Model:        model,
		Year:         req.Year,
		Color:        strings.TrimSpace(req.Color),
		Condition:    strings.TrimSpace(req.Condition),
		Price:        req.Price,
		Description:  strings.TrimSpace(req.Description),
		Status:       "pendiente", // Todo auto creado entra a moderación (HU-ADM-01)
		IsAuthorized: false,
		PublishedAt:  time.Now(),
		Mileage:      req.Mileage,
		Images:       req.Images,
	}

	return s.repo.Create(newPub)
}

// GetByID convierte el parámetro de URL a entero y consulta al repositorio O(1)
func (s *publicationService) GetByID(idStr string) (*models.Publication, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return nil, errors.New("el ID de la publicación debe ser un número entero positivo")
	}

	return s.repo.GetByID(id)
}

// GetAll obtiene las publicaciones y aplica filtros opcionales (por Marca o Modelo)
func (s *publicationService) GetAll(filters PublicationFilterRequest) ([]models.Publication, error) {
	pubs, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	// Si no hay filtros, retorna todo el catálogo
	if strings.TrimSpace(filters.Brand) == "" && strings.TrimSpace(filters.Model) == "" {
		return pubs, nil
	}

	// Filtrado insensible a mayúsculas/minúsculas
	filtered := make([]models.Publication, 0)
	brandFilter := strings.ToLower(strings.TrimSpace(filters.Brand))
	modelFilter := strings.ToLower(strings.TrimSpace(filters.Model))

	for _, p := range pubs {
		matchBrand := brandFilter == "" || strings.Contains(strings.ToLower(p.Brand), brandFilter)
		matchModel := modelFilter == "" || strings.Contains(strings.ToLower(p.Model), modelFilter)

		if matchBrand && matchModel {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

// Update busca la publicación y actualiza únicamente los campos que no sean nulos
func (s *publicationService) Update(idStr string, req UpdatePublicationRequest) (*models.Publication, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return nil, errors.New("el ID de la publicación debe ser un número entero positivo")
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Actualizar solo los campos provistos
	if req.Brand != nil {
		brand := strings.TrimSpace(*req.Brand)
		if brand == "" {
			return nil, errors.New("la marca no puede estar vacía")
		}
		existing.Brand = brand
	}

	if req.Model != nil {
		model := strings.TrimSpace(*req.Model)
		if model == "" {
			return nil, errors.New("el modelo no puede estar vacío")
		}
		existing.Model = model
	}

	if req.Year != nil {
		if *req.Year < 1900 || *req.Year > time.Now().Year()+1 {
			return nil, errors.New("el año del vehículo no es válido")
		}
		existing.Year = *req.Year
	}

	if req.Color != nil {
		existing.Color = strings.TrimSpace(*req.Color)
	}

	if req.Condition != nil {
		existing.Condition = strings.TrimSpace(*req.Condition)
	}

	if req.Mileage != nil {
		if *req.Mileage < 0 {
			return nil, errors.New("el kilometraje no puede ser negativo")
		}
		existing.Mileage = *req.Mileage
	}

	if req.Price != nil {
		if *req.Price <= 0 {
			return nil, errors.New("el precio debe ser mayor a cero")
		}
		existing.Price = *req.Price
	}

	if req.Description != nil {
		existing.Description = strings.TrimSpace(*req.Description)
	}

	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		existing.Status = status
		if status == "vendida" {
			now := time.Now()
			existing.SoldAt = &now
		}
	}

	if req.Images != nil {
		existing.Images = *req.Images
	}

	return s.repo.Update(existing)
}

// Authorize autoriza una publicación haciéndola visible en el catálogo público (HU-ADM-01)
func (s *publicationService) Authorize(idStr string) (*models.Publication, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return nil, errors.New("el ID de la publicación debe ser un número entero positivo")
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	existing.IsAuthorized = true
	existing.Status = "publicada"

	return s.repo.Update(existing)
}

// Delete elimina una publicación por su ID (HU-ADM-01)
func (s *publicationService) Delete(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return errors.New("el ID de la publicación debe ser un número entero positivo")
	}

	return s.repo.Delete(id)
}
