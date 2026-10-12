package handlers

import (
	"automarket/internal/middleware"
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

	router.GET("/", h.GetAllPublication)

	protected := router.Group("/")

	protected.Use(middleware.RequireAuth())
	{
		protected.POST("/", h.CreatePublication)
		protected.GET("/:id", h.GetMyPublication)
		protected.PATCH("/:id", h.UpdatePublication)
	}

}

func (h *PublicationHandler) CreatePublication(c *gin.Context) {
	var req CreatePublicationRequest
	userID := c.GetString("userID")
	req.OwnerID = userID

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

func (h *PublicationHandler) GetMyPublication(c *gin.Context) {
	userID := c.GetString("userID")

	publication, err := h.publicationService.GetByOwner(userID)
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
	userID := c.GetString("userID")
	role := c.GetString("userRole")
	req.OwnerID = userID
	req.Role = role

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
