package main

import (
	"log"
	"net/http"

	"automarket/internal/handlers"
	"automarket/internal/repositories"
	"automarket/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Inicializar capas del módulo Usuario (Persistencia JSON en data/user.json)
	userRepo := repositories.NewUserRepository("data/user.json")
	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)

	// 2. Inicializar router de Gin (incluye logger y recovery middleware nativo)
	router := gin.Default()

	// 3. Cargar plantillas HTML y servir archivos estáticos (CSS, JS, uploads)
	router.LoadHTMLGlob("web/templates/*.html")
	router.Static("/static", "./web/static")

	// 4. Registrar endpoints de Usuarios (/users)
	userHandler.RegisterRoutes(router, nil)

	// 5. Rutas para vistas del Frontend (SSR)
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

	// 6. Iniciar el servidor en el puerto 8080
	port := ":8080"
	log.Printf("💻 Servidor AutoMarket iniciando en http://localhost%s\n", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}
