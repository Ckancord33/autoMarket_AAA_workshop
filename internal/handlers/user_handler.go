package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler actúa como un contenedor para las dependencias (el Service).
type UserHandler struct {
	userService UserService // Se asume que UserService es una interfaz ya definida
}

// NewUserHandler es la función constructora que inyecta el servicio.
func NewUserHandler(us UserService) *UserHandler {
	return &UserHandler{
		userService: us,
	}
}

// RegisterRoutes ahora incluye las nuevas rutas del CRUD
func (h *UserHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	userGroup := router.Group("/users")
	userGroup.Use(authMiddleware)

	userGroup.POST("/", h.CreateUser)
	userGroup.GET("/:id", h.GetUser)
	userGroup.GET("/", h.GetAllUsers)
	userGroup.PATCH("/:id", h.UpdateUser)
}

// CreateUser maneja la petición POST para crear un usuario.
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno del servidor"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUser maneja la petición GET para obtener un usuario por ID.
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetAllUsers maneja la petición GET para obtener todos los usuarios (Get global).
func (h *UserHandler) GetAllUsers(c *gin.Context) {

	var filters UserFilterRequest

	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filtros inválidos"})
		return
	}

	users, err := h.userService.GetAll(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar los usuarios"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// UpdateUser maneja la petición PATCH para actualizar un usuario existente.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id") // Obtenemos el ID de la URL

	var req UpdateUserRequest

	// Vinculamos el JSON del body a la estructura req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de actualización inválidos: " + err.Error()})
		return
	}

	// Pasamos tanto el ID como los datos a actualizar al servicio
	updatedUser, err := h.userService.Update(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el usuario"})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
