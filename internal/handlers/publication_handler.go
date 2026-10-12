package handlers

import (
	"automarket/internal/middleware"
	"net/http"

	"automarket/internal/services"

	"github.com/gin-gonic/gin"
)

type PublicationHandler struct {
	publicationService services.PublicationService
}

func NewPublicationHandler(ps services.PublicationService) *PublicationHandler {
	return &PublicationHandler{
		publicationService: ps,
	}
}

func (h *PublicationHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	pubGroup := router.Group("/publications")
	pubGroup.GET("/", h.GetAllPublication)
	pubGroup.GET("/:id", h.GetPublication)

	protected := pubGroup.Group("/")
	if authMiddleware != nil {
		protected.Use(authMiddleware)
	} else {
		protected.Use(middleware.RequireAuth())
	}
	{
		protected.POST("/", h.CreatePublication)
		protected.PATCH("/:id", h.UpdatePublication)
	}
}

func (h *PublicationHandler) CreatePublication(c *gin.Context) {
	var req services.CreatePublicationRequest

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
	var filters services.PublicationFilterRequest

	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filtros inválidos"})
		return
	}

	publications, err := h.publicationService.GetAll(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar las publicaciones"})
		return
	}

	c.JSON(http.StatusOK, publications)
}

func (h *PublicationHandler) UpdatePublication(c *gin.Context) {
	id := c.Param("id")

	var req services.UpdatePublicationRequest

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
