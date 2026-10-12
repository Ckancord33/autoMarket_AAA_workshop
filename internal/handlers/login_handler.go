package handlers

import (
	"automarket/internal/middleware"
	"automarket/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DTO para la petición de Login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthService define el contrato necesario para autenticar un usuario
type AuthService interface {
	VerifyCredentials(email, password string) (*models.User, error)
}

type AuthHandler struct {
	authService AuthService
}

func NewAuthHandler(as AuthService) *AuthHandler {
	return &AuthHandler{
		authService: as,
	}
}

func (h *AuthHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/login", h.Login)
}

// POST /login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido: " + err.Error()})
		return
	}

	// 1. Verificar credenciales con el servicio
	user, err := h.authService.VerifyCredentials(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales incorrectas"})
		return
	}

	// 2. Generar JWT usando security.go (internal/middleware)
	token, err := middleware.GenerateJWT(strconv.Itoa(user.IDUser), user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el token"})
		return
	}

	// 3. Respuesta con token y datos básicos
	c.JSON(http.StatusOK, gin.H{
		"message": "Inicio de sesión exitoso",
		"token":   token,
		"user": gin.H{
			"id":    user.IDUser,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}
