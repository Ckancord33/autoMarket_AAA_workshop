package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"automarket/internal/handlers"
	"automarket/internal/middleware"
	"automarket/internal/repositories"
	"automarket/internal/services"

	"github.com/gin-gonic/gin"
)

func setupTestServer() *gin.Engine {
	gin.SetMode(gin.TestMode)

	auditRepo := repositories.NewAuditRepository("../../data/audit_log.json")
	userRepo := repositories.NewUserRepository("../../data/user.json")
	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)

	pubRepo := repositories.NewPublicationRepository("../../data/publication.json")
	pubService := services.NewPublicationService(pubRepo)
	pubHandler := handlers.NewPublicationHandler(pubService)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.AuditMiddleware(auditRepo))

	router.LoadHTMLGlob("../../web/templates/*.html")
	router.Static("/static", "../../web/static")

	userHandler.RegisterRoutes(router, nil)
	pubHandler.RegisterRoutes(router, nil)

	webHandler := handlers.NewWebHandler(userService, pubService)
	webHandler.RegisterWebRoutes(router)

	return router
}

func TestCatalogAndDetailWebRoutes(t *testing.T) {
	router := setupTestServer()

	// 1. Test GET / (Catálogo público)
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Esperado 200 en /, obtenido: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Catálogo de Vehículos Usados") {
		t.Errorf("No contiene el título del catálogo")
	}

	// 2. Test GET /publications/1 (Ficha técnica HTML)
	reqDetail, _ := http.NewRequest(http.MethodGet, "/publications/1", nil)
	reqDetail.Header.Set("Accept", "text/html")
	wDetail := httptest.NewRecorder()
	router.ServeHTTP(wDetail, reqDetail)

	if wDetail.Code != http.StatusOK {
		t.Fatalf("Esperado 200 en /publications/1, obtenido: %d", wDetail.Code)
	}
	detailBody := wDetail.Body.String()
	if !strings.Contains(detailBody, "BMW") && !strings.Contains(detailBody, "M4") {
		t.Errorf("No contiene información del vehículo esperado")
	}

	// 3. Test GET /publications/1 (Formato JSON para clientes API)
	reqJSON, _ := http.NewRequest(http.MethodGet, "/publications/1", nil)
	reqJSON.Header.Set("Accept", "application/json")
	wJSON := httptest.NewRecorder()
	router.ServeHTTP(wJSON, reqJSON)

	if wJSON.Code != http.StatusOK {
		t.Fatalf("Esperado 200 en /publications/1 JSON, obtenido: %d", wJSON.Code)
	}
	if !strings.Contains(wJSON.Body.String(), `"success":true`) {
		t.Errorf("Respuesta JSON no contiene success:true")
	}
}

func TestAuthFlowWebRoutes(t *testing.T) {
	router := setupTestServer()

	// 1. Login exitoso de vendedor
	form := url.Values{}
	form.Add("email", "carlos.vendedor@automarket.com")
	form.Add("password", "seller123")

	reqLogin, _ := http.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	reqLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusFound {
		t.Fatalf("Esperado 302 redirect al iniciar sesión, obtenido: %d", wLogin.Code)
	}
	cookie := wLogin.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "token=") {
		t.Fatalf("No se estableció la cookie token")
	}

	// 2. Acceder al panel de vendedor con la cookie
	reqSeller, _ := http.NewRequest(http.MethodGet, "/seller", nil)
	reqSeller.Header.Set("Cookie", cookie)
	wSeller := httptest.NewRecorder()
	router.ServeHTTP(wSeller, reqSeller)

	if wSeller.Code != http.StatusOK {
		t.Fatalf("Esperado 200 en /seller con cookie, obtenido: %d", wSeller.Code)
	}
	if !strings.Contains(wSeller.Body.String(), "Panel de Vendedor") {
		t.Errorf("El panel de vendedor no se renderizó correctamente")
	}
}

func TestAdminPanelAndPublicationFlow(t *testing.T) {
	router := setupTestServer()

	// 1. Iniciar sesión como administrador
	formAdmin := url.Values{}
	formAdmin.Add("email", "admin@automarket.com")
	formAdmin.Add("password", "admin123")

	reqAdminLogin, _ := http.NewRequest(http.MethodPost, "/login", strings.NewReader(formAdmin.Encode()))
	reqAdminLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wAdminLogin := httptest.NewRecorder()
	router.ServeHTTP(wAdminLogin, reqAdminLogin)

	if wAdminLogin.Code != http.StatusFound {
		t.Fatalf("Esperado 302 redirect al iniciar sesión admin, obtenido: %d", wAdminLogin.Code)
	}
	adminCookie := wAdminLogin.Header().Get("Set-Cookie")

	// 2. Acceder al panel de administración
	reqAdmin, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	reqAdmin.Header.Set("Cookie", adminCookie)
	wAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("Esperado 200 en /admin con cookie, obtenido: %d", wAdmin.Code)
	}
	if !strings.Contains(wAdmin.Body.String(), "Panel de Moderación y Control") {
		t.Errorf("El panel de administración no contiene el título esperado")
	}

	// 3. Crear una nueva publicación como vendedor
	formSeller := url.Values{}
	formSeller.Add("email", "carlos.vendedor@automarket.com")
	formSeller.Add("password", "seller123")

	reqSellerLogin, _ := http.NewRequest(http.MethodPost, "/login", strings.NewReader(formSeller.Encode()))
	reqSellerLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wSellerLogin := httptest.NewRecorder()
	router.ServeHTTP(wSellerLogin, reqSellerLogin)

	sellerCookie := wSellerLogin.Header().Get("Set-Cookie")

	formPub := url.Values{}
	formPub.Add("brand", "Audi")
	formPub.Add("model", "RS6 Avant")
	formPub.Add("year", "2024")
	formPub.Add("mileage", "5000")
	formPub.Add("price", "135000")
	formPub.Add("color", "Gris Nardo")
	formPub.Add("condition", "Nuevo")
	formPub.Add("imageUrl", "https://example.com/audi.jpg")
	formPub.Add("description", "Audi RS6 Avant V8 Biturbo en impecable estado.")

	reqCreatePub, _ := http.NewRequest(http.MethodPost, "/seller/publications", strings.NewReader(formPub.Encode()))
	reqCreatePub.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCreatePub.Header.Set("Cookie", sellerCookie)
	wCreatePub := httptest.NewRecorder()
	router.ServeHTTP(wCreatePub, reqCreatePub)

	if wCreatePub.Code != http.StatusOK {
		t.Fatalf("Esperado 200 al registrar publicación, obtenido: %d", wCreatePub.Code)
	}
	if !strings.Contains(wCreatePub.Body.String(), "RS6 Avant") {
		t.Errorf("La respuesta no contiene el vehículo recién creado")
	}
}
