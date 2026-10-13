package response

import (
	"github.com/gin-gonic/gin"
)

// Response representa la estructura estándar de salida para la API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// Success envía una respuesta HTTP exitosa estandarizada (200, 201, etc.)
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error envía una respuesta HTTP de fallo estandarizada (400, 401, 404, 500, etc.)
func Error(c *gin.Context, statusCode int, message string, err interface{}) {
	var errDetails interface{} = err

	// Si err implementa la interfaz error nativa, extraemos su texto
	if e, ok := err.(error); ok && e != nil {
		errDetails = e.Error()
	}

	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
		Error:   errDetails,
	})
}
