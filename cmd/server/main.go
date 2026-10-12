package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Inicializar router de Gin (incluye logger y recovery middleware nativo)
	router := gin.Default()

	// Cargar plantillas HTML y servir archivos estáticos (CSS, JS, uploads)
	router.LoadHTMLGlob("web/templates/*.html")
	router.Static("/static", "./web/static")

	// Ruta para el Login (/login)
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	router.POST("/login", func(c *gin.Context) {
		// Simulación de login exitoso redirigiendo al catálogo principal
		c.Redirect(http.StatusSeeOther, "/")
	})

	// Ruta principal para el Marketplace / Catálogo (/)
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Iniciar el servidor en el puerto 8080
	port := ":8080"
	log.Printf("💻 Servidor AutoMarket iniciando en http://localhost%s\n", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}
