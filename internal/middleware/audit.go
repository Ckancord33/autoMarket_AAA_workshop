package middleware

import (
	"log"
	"strconv"
	"strings"
	"time"

	"automarket/internal/models"
	"automarket/internal/repositories"

	"github.com/gin-gonic/gin"
)

// AuditMiddleware intercepta cada petición HTTP para registrar trazas operativas
// de forma no bloqueante cumpliendo con el pilar de Auditing (AAA).
func AuditMiddleware(repo *repositories.AuditRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Omitir archivos estáticos del audit log para no generar ruido
		if strings.HasPrefix(c.Request.URL.Path, "/static") {
			c.Next()
			return
		}

		start := time.Now()

		// Procesar la petición aguas abajo (middlewares de auth y controladores)
		c.Next()

		// Recopilar telemetría de respuesta
		latency := time.Since(start).Milliseconds()
		statusCode := c.Writer.Status()
		endpoint := c.Request.URL.Path
		method := c.Request.Method
		ipAddress := c.ClientIP()

		// Extraer identidad inyectada en el contexto por RequireAuth() (si existe)
		var idUserPtr *int
		role := "Anonymous"

		if uidVal, exists := c.Get("userID"); exists {
			if uidStr, ok := uidVal.(string); ok {
				if idInt, err := strconv.Atoi(uidStr); err == nil {
					idUserPtr = &idInt
				}
			}
		}

		if roleVal, exists := c.Get("userRole"); exists {
			if rStr, ok := roleVal.(string); ok && rStr != "" {
				role = rStr
			}
		}

		// Determinar la acción semántica según el método y endpoint
		action := resolveAction(method, endpoint)

		// Extraer detalles de error si ocurrieron
		var details *string
		if len(c.Errors) > 0 {
			errStr := c.Errors.String()
			details = &errStr
		}

		logEntry := models.AuditLog{
			IDUser:     idUserPtr,
			Role:       role,
			Action:     action,
			Method:     method,
			Endpoint:   endpoint,
			IPAddress:  ipAddress,
			StatusCode: statusCode,
			LatencyMs:  int(latency),
			Details:    details,
			Timestamp:  time.Now(),
		}

		// Persistencia no bloqueante mediante goroutine para no penalizar la latencia del cliente
		go func(entry models.AuditLog) {
			if _, err := repo.Create(&entry); err != nil {
				log.Printf("⚠️ Error al persistir audit log: %v", err)
			}
		}(logEntry)

		// Imprimir traza estructurada en consola
		userDisplay := "anonymous"
		if idUserPtr != nil {
			userDisplay = strconv.Itoa(*idUserPtr)
		}
		log.Printf("[AUDIT] %s | %s | %s %s | Status: %d | User: %s (%s) | %dms",
			logEntry.Timestamp.Format("2006-01-02 15:04:05"),
			action, method, endpoint, statusCode, userDisplay, role, latency)
	}
}

// resolveAction infiere una acción de seguridad semántica a partir de la ruta y el verbo HTTP.
func resolveAction(method, path string) string {
	switch {
	case method == "POST" && path == "/login":
		return "AUTH_LOGIN"
	case method == "GET" && path == "/logout":
		return "AUTH_LOGOUT"
	case method == "POST" && path == "/register":
		return "USER_REGISTER"
	case method == "POST" && path == "/seller/publications":
		return "PUBLICATION_CREATE"
	case method == "POST" && path == "/seller/publications/sell":
		return "PUBLICATION_REPORT_SALE"
	case method == "POST" && path == "/admin/publications/authorize":
		return "ADMIN_AUTHORIZE_PUBLICATION"
	case method == "POST" && path == "/admin/publications/delete":
		return "ADMIN_DELETE_PUBLICATION"
	case method == "POST" && path == "/admin/users/delete":
		return "ADMIN_DELETE_USER"
	case method == "GET" && path == "/seller":
		return "VIEW_SELLER_DASHBOARD"
	case method == "GET" && path == "/admin":
		return "VIEW_ADMIN_DASHBOARD"
	case method == "GET" && strings.HasPrefix(path, "/publications/"):
		return "VIEW_PUBLICATION_DETAIL"
	case path == "/":
		return "VIEW_CATALOG"
	case path == "/login" || path == "/register":
		return "VIEW_SSR"
	default:
		return method + " " + path
	}
}
