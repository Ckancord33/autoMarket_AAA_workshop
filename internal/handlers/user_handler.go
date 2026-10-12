package handlers

import (
	"automarket/internal/middleware"
	"automarket/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler actúa como un contenedor para las dependencias (el Service).
type UserHandler struct {
	userService services.UserService
}

// NewUserHandler es la función constructora que inyecta el servicio.
func NewUserHandler(us services.UserService) *UserHandler {
	return &UserHandler{
		userService: us,
	}
}

// RegisterRoutes registra las rutas de autenticación y CRUD de usuarios
func (h *UserHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	// Endpoint público para iniciar sesión
	router.POST("/login", h.Login)

	userGroup := router.Group("/users")
	userGroup.POST("/", h.CreateUser)
	userGroup.POST("/login", h.Login)

	protected := userGroup.Group("/")
	if authMiddleware != nil {
		protected.Use(authMiddleware)
	} else {
		protected.Use(middleware.RequireAuth())
	}
	{
		protected.GET("/:id", h.GetUser)
		managementGroup := protected.Group("/management")
		managementGroup.Use(middleware.RequireRoles("admin"))
		{
			managementGroup.GET("/", h.GetAllUsers)
			managementGroup.PATCH("/:id", h.UpdateUser)
		}
	}
}

// CreateUser maneja la petición POST para registrar un usuario
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResp, err := h.userService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, authResp)
}

// Login maneja la petición POST para autenticar un usuario y devolver el token JWT
func (h *UserHandler) Login(c *gin.Context) {
	var req services.LoginRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido: " + err.Error()})
		return
	}

	authResp, err := h.userService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResp)
}

// GetUser maneja la petición GET para obtener un usuario por ID.
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		id = c.GetString("userID")
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetAllUsers maneja la petición GET para obtener todos los usuarios (Get global).
func (h *UserHandler) GetAllUsers(c *gin.Context) {
	var filters services.UserFilterRequest

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
	id := c.Param("id")

	var req services.UpdateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de actualización inválidos: " + err.Error()})
		return
	}

	updatedUser, err := h.userService.Update(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el usuario: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}
