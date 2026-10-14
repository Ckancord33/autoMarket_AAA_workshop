package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"automarket/internal/middleware"
	"automarket/internal/models"
	"automarket/internal/services"

	"github.com/gin-gonic/gin"
)

// WebHandler gestiona la interfaz web Server-Side Rendering (SSR).
// La persistencia de publicaciones se almacena en el archivo data/publication.json.
type WebHandler struct {
	userService services.UserService
	filePath    string
	mu          sync.RWMutex
	pubs        map[int]models.Publication
	lastPubID   int
}

// NewWebHandler crea el controlador para la interfaz web SSR
func NewWebHandler(us services.UserService) *WebHandler {
	h := &WebHandler{
		userService: us,
		filePath:    "data/publication.json",
		pubs:        make(map[int]models.Publication),
		lastPubID:   0,
	}

	_ = h.ensureFileAndLoad()
	return h
}

// ensureFileAndLoad garantiza la existencia de data/publication.json y carga los datos a memoria.
func (h *WebHandler) ensureFileAndLoad() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	info, err := os.Stat(h.filePath)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		seed := h.generateSeedPublications()
		data, err := json.MarshalIndent(seed, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(h.filePath, data, 0644); err != nil {
			return err
		}
		for _, p := range seed {
			h.pubs[p.IDPublication] = p
			if p.IDPublication > h.lastPubID {
				h.lastPubID = p.IDPublication
			}
		}
		return nil
	}

	data, err := os.ReadFile(h.filePath)
	if err != nil {
		return err
	}

	var pubList []models.Publication
	if err := json.Unmarshal(data, &pubList); err != nil {
		return err
	}

	for _, p := range pubList {
		h.pubs[p.IDPublication] = p
		if p.IDPublication > h.lastPubID {
			h.lastPubID = p.IDPublication
		}
	}

	return nil
}

// persistLocked guarda el estado de las publicaciones en data/publication.json
func (h *WebHandler) persistLocked() error {
	list := make([]models.Publication, 0, len(h.pubs))
	for _, p := range h.pubs {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].IDPublication < list[j].IDPublication
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(h.filePath, data, 0644)
}

// generateSeedPublications inicializa el catálogo si el archivo data/publication.json está vacío
func (h *WebHandler) generateSeedPublications() []models.Publication {
	now := time.Now()
	return []models.Publication{
		{
			IDPublication: 1,
			IDUser:        2,
			Brand:         "BMW",
			Model:         "M4 Competition Coupé",
			Year:          2023,
			Color:         "Azul Isle of Man",
			Condition:     "Usado Certificado",
			Price:         84900,
			Description:   "BMW M4 Competition Coupé en estado impecable. Motor 3.0L M TwinPower Turbo de 510 HP, caja M Steptronic de 8 velocidades con Drivelogic. Mantenimientos al día en concesionario oficial.",
			Status:        "publicada",
			IsAuthorized:  true,
			PublishedAt:   now.Add(-48 * time.Hour),
			Mileage:       12500,
			ImageURL:      "https://images.unsplash.com/photo-1555215695-3004980ad54e?auto=format&fit=crop&w=900&q=80",
		},
		{
			IDPublication: 2,
			IDUser:        2,
			Brand:         "Porsche",
			Model:         "911 Carrera S (992)",
			Year:          2022,
			Color:         "Gris Plata GT",
			Condition:     "Usado",
			Price:         138000,
			Description:   "Porsche 911 Carrera S con paquete Sport Chrono, escape deportivo y rines Carrera Classic de 20/21 pulgadas. Cuidado de colección, único dueño.",
			Status:        "publicada",
			IsAuthorized:  true,
			PublishedAt:   now.Add(-24 * time.Hour),
			Mileage:       8900,
			ImageURL:      "https://images.unsplash.com/photo-1614162692292-7ac56d7f7f1e?auto=format&fit=crop&w=900&q=80",
		},
		{
			IDPublication: 3,
			IDUser:        2,
			Brand:         "Toyota",
			Model:         "Land Cruiser 300 VX",
			Year:          2024,
			Color:         "Blanco Perlado",
			Condition:     "Nuevo",
			Price:         112000,
			Description:   "Toyota Land Cruiser 300 VX V6 Twin Turbo Diésel. Suspensión adaptativa, 7 puestos con tapicería en cuero y tracción integral permanente.",
			Status:        "publicada",
			IsAuthorized:  true,
			PublishedAt:   now.Add(-12 * time.Hour),
			Mileage:       2800,
			ImageURL:      "https://images.unsplash.com/photo-1594502184342-2e12f877aa73?auto=format&fit=crop&w=900&q=80",
		},
		{
			IDPublication: 4,
			IDUser:        2,
			Brand:         "Mercedes-Benz",
			Model:         "AMG GT Coupé",
			Year:          2023,
			Color:         "Negro Obsidiana",
			Condition:     "Usado",
			Price:         125000,
			Description:   "Mercedes-AMG GT Coupé Biturbo con paquete aerodinámico AMG. En espera de autorización por el equipo de moderación.",
			Status:        "pendiente",
			IsAuthorized:  false,
			PublishedAt:   now.Add(-2 * time.Hour),
			Mileage:       14200,
			ImageURL:      "https://images.unsplash.com/photo-1617814076367-b759c7d7e738?auto=format&fit=crop&w=900&q=80",
		},
	}
}

// RegisterWebRoutes registra todas las rutas SSR de la interfaz
func (h *WebHandler) RegisterWebRoutes(router *gin.Engine) {
	// Catálogo público y ficha técnica (HU-VIS-01)
	router.GET("/", h.IndexView)
	router.GET("/publications/:id", h.PublicationDetailView)

	// Autenticación (Login, Registro y Logout)
	router.GET("/login", h.LoginView)
	router.POST("/login", h.ProcessLogin)
	router.GET("/register", h.RegisterView)
	router.POST("/register", h.ProcessRegister)
	router.GET("/logout", h.ProcessLogout)

	// Rutas protegidas para Vendedor (HU-VEN-01)
	sellerGroup := router.Group("/seller")
	sellerGroup.Use(middleware.RequireAuth())
	sellerGroup.Use(middleware.RequireRoles("seller", "admin"))
	{
		sellerGroup.GET("", h.SellerDashboardView)
		sellerGroup.POST("/publications", h.CreatePublication)
		sellerGroup.POST("/publications/sell", h.ReportSale)
	}

	// Rutas protegidas para Administrador (HU-ADM-01)
	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.RequireAuth())
	adminGroup.Use(middleware.RequireRoles("admin"))
	{
		adminGroup.GET("", h.AdminDashboardView)
		adminGroup.POST("/publications/authorize", h.AuthorizePublication)
		adminGroup.POST("/publications/delete", h.DeletePublication)
		adminGroup.POST("/users/delete", h.DeleteUser)
	}
}

// IndexView renderiza el catálogo público (HU-VIS-01) sin filtros no solicitados
func (h *WebHandler) IndexView(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var visiblePubs []models.Publication
	for _, p := range h.pubs {
		// En catálogo público SOLO se muestran las autorizadas y en estado "publicada"
		if p.IsAuthorized && p.Status == "publicada" {
			visiblePubs = append(visiblePubs, p)
		}
	}

	sort.Slice(visiblePubs, func(i, j int) bool {
		return visiblePubs[i].IDPublication > visiblePubs[j].IDPublication
	})

	var currentUser *models.User
	if uid, _, ok := middleware.GetUserSession(c); ok {
		if u, err := h.userService.GetByID(uid); err == nil {
			currentUser = u
		}
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Publications": visiblePubs,
		"User":         currentUser,
	})
}

// PublicationDetailView muestra la ficha técnica y los datos de contacto del vendedor (HU-VIS-01)
func (h *WebHandler) PublicationDetailView(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	h.mu.RLock()
	found, exists := h.pubs[id]
	h.mu.RUnlock()

	if !exists {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// Obtener datos del vendedor asociado
	seller, err := h.userService.GetByID(strconv.Itoa(found.IDUser))
	if err != nil || seller == nil {
		seller = &models.User{
			Name:  "Carlos Vendedor",
			Email: "carlos.vendedor@automarket.com",
			Phone: "+57 310 987 6543",
			City:  "Bogotá D.C.",
		}
	}

	var currentUser *models.User
	if uid, _, ok := middleware.GetUserSession(c); ok {
		if u, err := h.userService.GetByID(uid); err == nil {
			currentUser = u
		}
	}

	c.HTML(http.StatusOK, "publication_detail.html", gin.H{
		"Publication": found,
		"Seller":      seller,
		"User":        currentUser,
	})
}

// LoginView muestra el formulario de login
func (h *WebHandler) LoginView(c *gin.Context) {
	if _, role, ok := middleware.GetUserSession(c); ok {
		if role == "admin" {
			c.Redirect(http.StatusFound, "/admin")
			return
		}
		c.Redirect(http.StatusFound, "/seller")
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{})
}

// RegisterView muestra el formulario de registro de vendedor
func (h *WebHandler) RegisterView(c *gin.Context) {
	if _, role, ok := middleware.GetUserSession(c); ok {
		if role == "admin" {
			c.Redirect(http.StatusFound, "/admin")
			return
		}
		c.Redirect(http.StatusFound, "/seller")
		return
	}

	c.HTML(http.StatusOK, "register.html", gin.H{})
}

// ProcessLogin maneja el inicio de sesión desde formulario HTML nativo
func (h *WebHandler) ProcessLogin(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := strings.TrimSpace(c.PostForm("password"))

	authResp, err := h.userService.Login(services.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	// Guardar sesión en cookie segura HTTP
	c.SetCookie("token", authResp.Token, 86400, "/", "", false, true)

	if authResp.User.Role == "admin" {
		c.Redirect(http.StatusFound, "/admin")
		return
	}
	c.Redirect(http.StatusFound, "/seller")
}

// ProcessRegister crea un vendedor desde formulario HTML nativo (HU-VEN-01)
func (h *WebHandler) ProcessRegister(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(c.PostForm("email"))
	phone := strings.TrimSpace(c.PostForm("phone"))
	city := strings.TrimSpace(c.PostForm("city"))
	password := strings.TrimSpace(c.PostForm("password"))

	authResp, err := h.userService.Register(services.CreateUserRequest{
		Name:     name,
		Email:    email,
		Phone:    phone,
		City:     city,
		Password: password,
	})
	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	c.SetCookie("token", authResp.Token, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/seller")
}

// ProcessLogout finaliza la sesión borrando la cookie
func (h *WebHandler) ProcessLogout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// SellerDashboardView muestra el panel del vendedor con sus publicaciones (HU-VEN-01)
func (h *WebHandler) SellerDashboardView(c *gin.Context) {
	uidStr := c.GetString("userID")
	uid, _ := strconv.Atoi(uidStr)

	user, _ := h.userService.GetByID(uidStr)

	h.mu.RLock()
	var myPubs []models.Publication
	for _, p := range h.pubs {
		if p.IDUser == uid || uid == 0 {
			myPubs = append(myPubs, p)
		}
	}
	h.mu.RUnlock()

	sort.Slice(myPubs, func(i, j int) bool {
		return myPubs[i].IDPublication > myPubs[j].IDPublication
	})

	c.HTML(http.StatusOK, "seller.html", gin.H{
		"User":         user,
		"Publications": myPubs,
	})
}

// CreatePublication registra una nueva publicación en estado "pendiente" (HU-VEN-01) y persiste a data/publication.json
func (h *WebHandler) CreatePublication(c *gin.Context) {
	uidStr := c.GetString("userID")
	uid, _ := strconv.Atoi(uidStr)

	year, _ := strconv.Atoi(c.PostForm("year"))
	mileage, _ := strconv.Atoi(c.PostForm("mileage"))
	price, _ := strconv.ParseFloat(c.PostForm("price"), 64)

	newPub := models.Publication{
		IDUser:       uid,
		Brand:        strings.TrimSpace(c.PostForm("brand")),
		Model:        strings.TrimSpace(c.PostForm("model")),
		Year:         year,
		Color:        strings.TrimSpace(c.PostForm("color")),
		Condition:    strings.TrimSpace(c.PostForm("condition")),
		Price:        price,
		Description:  strings.TrimSpace(c.PostForm("description")),
		Status:       "pendiente", // Forzado por HU-VEN-01
		IsAuthorized: false,       // Forzado por HU-VEN-01
		PublishedAt:  time.Now(),
		Mileage:      mileage,
		ImageURL:     strings.TrimSpace(c.PostForm("imageUrl")),
	}

	h.mu.Lock()
	h.lastPubID++
	newPub.IDPublication = h.lastPubID
	h.pubs[newPub.IDPublication] = newPub
	_ = h.persistLocked()
	h.mu.Unlock()

	user, _ := h.userService.GetByID(uidStr)

	h.mu.RLock()
	var myPubs []models.Publication
	for _, p := range h.pubs {
		if p.IDUser == uid || uid == 0 {
			myPubs = append(myPubs, p)
		}
	}
	h.mu.RUnlock()

	sort.Slice(myPubs, func(i, j int) bool {
		return myPubs[i].IDPublication > myPubs[j].IDPublication
	})

	c.HTML(http.StatusOK, "seller.html", gin.H{
		"User":         user,
		"Publications": myPubs,
		"Success":      "Publicación registrada. Se encuentra en estado 'pendiente' a la espera de autorización por el administrador.",
	})
}

// ReportSale marca una publicación como vendida (HU-VEN-01) y persiste a data/publication.json
func (h *WebHandler) ReportSale(c *gin.Context) {
	pubIDStr := c.PostForm("idPublication")
	pubID, _ := strconv.Atoi(pubIDStr)

	h.mu.Lock()
	if p, exists := h.pubs[pubID]; exists {
		now := time.Now()
		p.Status = "vendida"
		p.SoldAt = &now
		h.pubs[pubID] = p
		_ = h.persistLocked()
	}
	h.mu.Unlock()

	c.Redirect(http.StatusSeeOther, "/seller")
}

// AdminDashboardView muestra la bandeja de moderación y usuarios (HU-ADM-01)
func (h *WebHandler) AdminDashboardView(c *gin.Context) {
	uidStr := c.GetString("userID")
	user, _ := h.userService.GetByID(uidStr)

	h.mu.RLock()
	var pendingPubs []models.Publication
	var activePubs []models.Publication
	for _, p := range h.pubs {
		if !p.IsAuthorized || p.Status == "pendiente" {
			pendingPubs = append(pendingPubs, p)
		} else if p.Status == "publicada" {
			activePubs = append(activePubs, p)
		}
	}
	h.mu.RUnlock()

	sort.Slice(pendingPubs, func(i, j int) bool {
		return pendingPubs[i].IDPublication > pendingPubs[j].IDPublication
	})

	users, _ := h.userService.GetAll(services.UserFilterRequest{})

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"User":                user,
		"PendingPublications": pendingPubs,
		"ActivePublications":  activePubs,
		"Users":               users,
	})
}

// AuthorizePublication autoriza una publicación haciéndola visible en el catálogo (HU-ADM-01) y persiste
func (h *WebHandler) AuthorizePublication(c *gin.Context) {
	pubIDStr := c.PostForm("idPublication")
	pubID, _ := strconv.Atoi(pubIDStr)

	h.mu.Lock()
	if p, exists := h.pubs[pubID]; exists {
		p.IsAuthorized = true
		p.Status = "publicada"
		h.pubs[pubID] = p
		_ = h.persistLocked()
	}
	h.mu.Unlock()

	c.Redirect(http.StatusSeeOther, "/admin")
}

// DeletePublication elimina una publicación (HU-ADM-01) y persiste a data/publication.json
func (h *WebHandler) DeletePublication(c *gin.Context) {
	pubIDStr := c.PostForm("idPublication")
	pubID, _ := strconv.Atoi(pubIDStr)

	h.mu.Lock()
	if _, exists := h.pubs[pubID]; exists {
		delete(h.pubs, pubID)
		_ = h.persistLocked()
	}
	h.mu.Unlock()

	c.Redirect(http.StatusSeeOther, "/admin")
}

// DeleteUser elimina un usuario realmente de data/user.json (HU-ADM-01)
func (h *WebHandler) DeleteUser(c *gin.Context) {
	userID := strings.TrimSpace(c.PostForm("idUser"))
	if userID != "" {
		_ = h.userService.Delete(userID)
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}
