package main

import (
	"log"

	"automarket/internal/handlers"
	"automarket/internal/middleware"
	"automarket/internal/repositories"
	"automarket/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Inicializar repositorio de Auditoría AAA (data/audit_log.json)
	auditRepo := repositories.NewAuditRepository("data/audit_log.json")

	// 2. Inicializar capas del módulo Usuario (Persistencia JSON en data/user.json)
	userRepo := repositories.NewUserRepository("data/user.json")
	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)

	// 3. Inicializar capas del módulo Publicación (Persistencia JSON en data/publication.json)
	pubRepo := repositories.NewPublicationRepository("data/publication.json")
	pubService := services.NewPublicationService(pubRepo)
	pubHandler := handlers.NewPublicationHandler(pubService, userService)

	// 4. Inicializar router de Gin sin el logger genérico (usando gin.New)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.AuditMiddleware(auditRepo))

	// 5. Cargar plantillas HTML y servir archivos estáticos (CSS, JS, uploads)
	router.LoadHTMLGlob("web/templates/*.html")
	router.Static("/static", "./web/static")

	// 6. Registrar rutas y vistas del módulo de Usuarios (Login, Registro, /users)
	userHandler.RegisterRoutes(router, nil)

	// 7. Registrar rutas y vistas del módulo de Publicaciones (Catálogo, Panel Seller, Panel Admin, /publications)
	pubHandler.RegisterRoutes(router, nil)

	// 8. Iniciar el servidor en el puerto 8080
	port := ":8080"
	log.Printf("💻 Servidor AutoMarket iniciando en http://localhost%s\n", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}
