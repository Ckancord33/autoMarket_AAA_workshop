package handlers

import (
	"automarket/internal/middleware"
	"automarket/internal/services"
	"automarket/pkg/response"
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

// RegisterRoutes registra las rutas de autenticación y moderación de usuarios
func (h *UserHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	// --- Rutas Públicas de la API de Usuarios (HU-VEN-01 y Autenticación) ---
	userGroup := router.Group("/users")
	userGroup.POST("/", h.CreateUser)
	userGroup.POST("/login", h.Login)

	// --- Rutas Protegidas de Administración (HU-ADM-01: Eliminar usuarios) ---
	protected := userGroup.Group("/")
	if authMiddleware != nil {
		protected.Use(authMiddleware)
	} else {
		protected.Use(middleware.RequireAuth())
	}
	protected.Use(middleware.RequireRoles("admin"))
	{
		protected.DELETE("/:id", h.DeleteUser)
	}
}

// CreateUser maneja la petición POST para registrar un usuario
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest

	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Datos de registro inválidos", err)
		return
	}

	authResp, err := h.userService.Register(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Error al registrar usuario", err)
		return
	}

	response.Success(c, http.StatusCreated, "Usuario registrado exitosamente", authResp)
}

// Login maneja la petición POST para autenticar un usuario y devolver el token JWT
func (h *UserHandler) Login(c *gin.Context) {
	var req services.LoginRequest

	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Formato de credenciales inválido", err)
		return
	}

	authResp, err := h.userService.Login(req)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Credenciales inválidas", err)
		return
	}

	response.Success(c, http.StatusOK, "Inicio de sesión exitoso", authResp)
}

// DeleteUser maneja la petición DELETE para que un administrador elimine un usuario.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	if err := h.userService.Delete(id); err != nil {
		response.Error(c, http.StatusNotFound, "Error al eliminar usuario", err)
		return
	}

	response.Success(c, http.StatusOK, "Usuario eliminado exitosamente", nil)
}
