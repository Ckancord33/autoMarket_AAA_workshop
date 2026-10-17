package handlers

import (
	"automarket/internal/middleware"
	"automarket/internal/services"
	"automarket/pkg/response"
	"net/http"
	"strconv"

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
	// Nota: GET /publications/:id es atendido por WebHandler.PublicationDetailView para dar soporte tanto a la vista HTML como a clientes API (JSON) sin colisión de rutas.

	// --- Rutas Protegidas Generales (Requieren Autenticación) ---
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

	// --- Rutas Protegidas de Administración (HU-ADM-01: Moderar y autorizar) ---
	adminGroup := protected.Group("/")
	adminGroup.Use(middleware.RequireRoles("admin"))
	{
		adminGroup.POST("/:id/authorize", h.AuthorizePublication)
	}
}

func (h *PublicationHandler) CreatePublication(c *gin.Context) {
	var req services.CreatePublicationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Datos de publicación inválidos", err)
		return
	}

	if req.IDUser <= 0 {
		if uidStr := c.GetString("userID"); uidStr != "" {
			if uid, err := strconv.Atoi(uidStr); err == nil && uid > 0 {
				req.IDUser = uid
			}
		}
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

// AuthorizePublication maneja la petición POST para que un administrador autorice un vehículo
func (h *PublicationHandler) AuthorizePublication(c *gin.Context) {
	id := c.Param("id")

	authorizedPublication, err := h.publicationService.Authorize(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "No se pudo autorizar la publicación", err)
		return
	}

	response.Success(c, http.StatusOK, "Publicación autorizada exitosamente", authorizedPublication)
}
