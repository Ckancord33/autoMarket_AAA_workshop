package middleware

import (
	"automarket/pkg/response"
	"automarket/pkg/security"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "No autorizado", "Encabezado de autorización ausente")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, http.StatusUnauthorized, "Formato de token inválido", "El encabezado debe tener el formato: Bearer <token>")
			c.Abort()
			return
		}

		payload, err := security.VerifyJWT(parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "Token inválido o expirado", err.Error())
			c.Abort()
			return
		}

		// Guardamos la identidad en el contexto de Gin
		c.Set("userID", payload.UserID)
		c.Set("userRole", payload.Role)

		c.Next()
	}
}

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleContext, exists := c.Get("userRole")
		if !exists {
			response.Error(c, http.StatusForbidden, "Sesión inválida", "No se encontró información de rol en la sesión")
			c.Abort()
			return
		}

		userRole := roleContext.(string)
		hasPermission := false

		for _, role := range allowedRoles {
			if strings.EqualFold(userRole, role) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			response.Error(c, http.StatusForbidden, "Acceso denegado: rol insuficiente", fmt.Sprintf("Se requiere uno de los siguientes roles: %s", strings.Join(allowedRoles, ", ")))
			c.Abort()
			return
		}

		c.Next()
	}
}
