package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PublicationHandler struct {
	publicationService PublicationService
}

func NewPublicationHandler(ps PublicationService) *PublicationHandler {
	return &PublicationHandler{
		publicationService: ps,
	}
}

func (h *PublicationHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	pubGroup := router.Group("/publications")
	pubGroup.Use(authMiddleware)

	pubGroup.POST("/", h.CreatePublication)
	pubGroup.GET("/:id", h.GetPublication)
	pubGroup.GET("/", h.GetAllPublication)
	pubGroup.PATCH("/:id", h.UpdatePublication)
}

func (h *PublicationHandler) CreatePublication(c *gin.Context) {
	var req CreatePublicationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	publication, err := h.publicationService.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno del servidor"})
		return
	}

	c.JSON(http.StatusCreated, publication)
}

func (h *PublicationHandler) GetPublication(c *gin.Context) {
	id := c.Param("id")

	publication, err := h.publicationService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Publicación no encontrada"})
		return
	}

	c.JSON(http.StatusOK, publication)
}

func (h *PublicationHandler) GetAllPublication(c *gin.Context) {

	var filters PublicationFilterRequest

	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filtros inválidos"})
		return
	}

	users, err := h.publicationService.GetAll(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar los usuarios"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *PublicationHandler) UpdatePublication(c *gin.Context) {
	id := c.Param("id")

	var req UpdatePublicationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de actualización inválidos: " + err.Error()})
		return
	}

	updatedPublication, err := h.publicationService.Update(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar la publicación"})
		return
	}

	c.JSON(http.StatusOK, updatedPublication)
}
