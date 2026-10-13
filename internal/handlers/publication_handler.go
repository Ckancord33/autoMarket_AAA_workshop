package handlers

import (
	"automarket/internal/middleware"
	"automarket/internal/services"
	"automarket/pkg/response"
	"net/http"

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
		response.Error(c, http.StatusBadRequest, "Datos de publicación inválidos", err)
		return
	}

	publication, err := h.publicationService.Create(req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error interno del servidor", err)
		return
	}

	response.Success(c, http.StatusCreated, "Publicación creada exitosamente", publication)
}

func (h *PublicationHandler) GetPublication(c *gin.Context) {
	id := c.Param("id")

	publication, err := h.publicationService.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Publicación no encontrada", err)
		return
	}

	response.Success(c, http.StatusOK, "Publicación obtenida exitosamente", publication)
}

func (h *PublicationHandler) GetAllPublication(c *gin.Context) {
	var filters services.PublicationFilterRequest

	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "Filtros inválidos", err)
		return
	}

	publications, err := h.publicationService.GetAll(filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar las publicaciones", err)
		return
	}

	response.Success(c, http.StatusOK, "Publicaciones obtenidas exitosamente", publications)
}

func (h *PublicationHandler) UpdatePublication(c *gin.Context) {
	id := c.Param("id")

	var req services.UpdatePublicationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Datos de actualización inválidos", err)
		return
	}

	updatedPublication, err := h.publicationService.Update(id, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "No se pudo actualizar la publicación", err)
		return
	}

	response.Success(c, http.StatusOK, "Publicación actualizada exitosamente", updatedPublication)
}
