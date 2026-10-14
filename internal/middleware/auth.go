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
		tokenString := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// Si no viene en el header, intentar extraer de la Cookie HTTP (modo SSR sin JS)
		if tokenString == "" {
			if cookieToken, err := c.Cookie("token"); err == nil && cookieToken != "" {
				tokenString = cookieToken
			}
		}

		if tokenString == "" {
			if strings.Contains(c.GetHeader("Accept"), "text/html") {
				c.Redirect(http.StatusSeeOther, "/login")
				c.Abort()
				return
			}
			response.Error(c, http.StatusUnauthorized, "No autorizado", "Encabezado de autorización o sesión ausente")
			c.Abort()
			return
		}

		payload, err := security.VerifyJWT(tokenString)
		if err != nil {
			if strings.Contains(c.GetHeader("Accept"), "text/html") {
				c.Redirect(http.StatusSeeOther, "/login")
				c.Abort()
				return
			}
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

// GetUserSession intenta extraer de forma opcional la identidad del usuario desde la cookie
func GetUserSession(c *gin.Context) (userID string, role string, ok bool) {
	tokenString, err := c.Cookie("token")
	if err != nil || tokenString == "" {
		return "", "", false
	}

	payload, err := security.VerifyJWT(tokenString)
	if err != nil {
		return "", "", false
	}

	return payload.UserID, payload.Role, true
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
