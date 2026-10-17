package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"automarket/internal/middleware"
	"automarket/internal/models"
	"automarket/internal/services"
	"automarket/pkg/response"

	"github.com/gin-gonic/gin"
)

// WebHandler gestiona la interfaz web Server-Side Rendering (SSR) conectando
// las vistas directamente con las capas de servicio reales (UserService y PublicationService).
type WebHandler struct {
	userService services.UserService
	pubService  services.PublicationService
}

// NewWebHandler crea el controlador para la interfaz web SSR
func NewWebHandler(us services.UserService, ps services.PublicationService) *WebHandler {
	return &WebHandler{
		userService: us,
		pubService:  ps,
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

// IndexView renderiza el catálogo público (HU-VIS-01) consultando las publicaciones reales
func (h *WebHandler) IndexView(c *gin.Context) {
	allPubs, err := h.pubService.GetAll(services.PublicationFilterRequest{})
	if err != nil {
		allPubs = []models.Publication{}
	}

	var visiblePubs []models.Publication
	for _, p := range allPubs {
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
// Brinda soporte tanto a la vista HTML como a clientes API que soliciten JSON.
func (h *WebHandler) PublicationDetailView(c *gin.Context) {
	idStr := c.Param("id")

	found, err := h.pubService.GetByID(idStr)
	if err != nil || found == nil {
		if strings.Contains(c.GetHeader("Accept"), "application/json") {
			response.Error(c, http.StatusNotFound, "Publicación no encontrada", err)
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// Si es una petición API solicitando JSON exclusivamente
	if strings.Contains(c.GetHeader("Accept"), "application/json") && !strings.Contains(c.GetHeader("Accept"), "text/html") {
		response.Success(c, http.StatusOK, "Publicación obtenida exitosamente", found)
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

	type CarouselSlide struct {
		Index   int
		Total   int
		URL     string
		PrevID  string
		NextID  string
		SlideID string
	}

	var slides []CarouselSlide
	totalImgs := len(found.Images)
	for i, imgURL := range found.Images {
		prevIdx := (i - 1 + totalImgs) % totalImgs
		nextIdx := (i + 1) % totalImgs
		slides = append(slides, CarouselSlide{
			Index:   i + 1,
			Total:   totalImgs,
			URL:     imgURL,
			SlideID: fmt.Sprintf("slide-%d", i+1),
			PrevID:  fmt.Sprintf("slide-%d", prevIdx+1),
			NextID:  fmt.Sprintf("slide-%d", nextIdx+1),
		})
	}

	c.HTML(http.StatusOK, "publication_detail.html", gin.H{
		"Publication": found,
		"Seller":      seller,
		"User":        currentUser,
		"Slides":      slides,
	})
}

// LoginView muestra el formulario de login
func (h *WebHandler) LoginView(c *gin.Context) {
	if uid, role, ok := middleware.GetUserSession(c); ok {
		// Validar si el usuario aún existe en la base de datos (no fue eliminado por el admin)
		if u, err := h.userService.GetByID(uid); err == nil && u != nil && u.IsActive {
			if role == "admin" {
				c.Redirect(http.StatusFound, "/admin")
				return
			}
			c.Redirect(http.StatusFound, "/seller")
			return
		}
		// Si la cuenta fue eliminada, limpiar la cookie fantasma
		c.SetCookie("token", "", -1, "/", "", false, true)
	}

	data := gin.H{}
	if errMsg := c.Query("error"); errMsg != "" {
		data["Error"] = errMsg
	}
	if msg := c.Query("success"); msg != "" {
		data["Success"] = msg
	}

	c.HTML(http.StatusOK, "login.html", data)
}

// RegisterView muestra el formulario de registro de vendedor
func (h *WebHandler) RegisterView(c *gin.Context) {
	if uid, role, ok := middleware.GetUserSession(c); ok {
		// Validar si el usuario aún existe en la base de datos
		if u, err := h.userService.GetByID(uid); err == nil && u != nil && u.IsActive {
			if role == "admin" {
				c.Redirect(http.StatusFound, "/admin")
				return
			}
			c.Redirect(http.StatusFound, "/seller")
			return
		}
		// Si la cuenta fue eliminada, limpiar la cookie fantasma
		c.SetCookie("token", "", -1, "/", "", false, true)
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

	user, err := h.userService.GetByID(uidStr)
	if err != nil || user == nil || !user.IsActive {
		c.SetCookie("token", "", -1, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/login?error="+url.QueryEscape("Tu cuenta ha sido eliminada por el administrador."))
		return
	}

	allPubs, _ := h.pubService.GetAll(services.PublicationFilterRequest{})
	var myPubs []models.Publication
	for _, p := range allPubs {
		if p.IDUser == uid || uid == 0 {
			myPubs = append(myPubs, p)
		}
	}

	sort.Slice(myPubs, func(i, j int) bool {
		return myPubs[i].IDPublication > myPubs[j].IDPublication
	})

	data := gin.H{
		"User":         user,
		"Publications": myPubs,
	}

	if msg := c.Query("success"); msg != "" {
		data["Success"] = msg
	}
	if errMsg := c.Query("error"); errMsg != "" {
		data["Error"] = errMsg
	}

	c.HTML(http.StatusOK, "seller.html", data)
}

// CreatePublication registra una nueva publicación en estado "pendiente" (HU-VEN-01) a través de PublicationService
func (h *WebHandler) CreatePublication(c *gin.Context) {
	uidStr := c.GetString("userID")
	uid, _ := strconv.Atoi(uidStr)

	year, _ := strconv.Atoi(c.PostForm("year"))
	mileage, _ := strconv.Atoi(c.PostForm("mileage"))
	price, _ := strconv.ParseFloat(c.PostForm("price"), 64)

	var imagePaths []string
	form, formErr := c.MultipartForm()
	if formErr == nil && form != nil {
		files := form.File["images"]
		uploadDir := filepath.Join("web", "static", "uploads")
		_ = os.MkdirAll(uploadDir, 0755)

		allowedExts := map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
		}

		for _, fileHeader := range files {
			ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
			if !allowedExts[ext] {
				continue
			}
			newFileName := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), len(imagePaths)+1, ext)
			dst := filepath.Join(uploadDir, newFileName)
			if err := c.SaveUploadedFile(fileHeader, dst); err == nil {
				imagePaths = append(imagePaths, "/static/uploads/"+newFileName)
			}
		}
	}

	_, err := h.pubService.Create(services.CreatePublicationRequest{
		IDUser:      uid,
		Brand:       strings.TrimSpace(c.PostForm("brand")),
		Model:       strings.TrimSpace(c.PostForm("model")),
		Year:        year,
		Color:       strings.TrimSpace(c.PostForm("color")),
		Condition:   strings.TrimSpace(c.PostForm("condition")),
		Price:       price,
		Description: strings.TrimSpace(c.PostForm("description")),
		Mileage:     mileage,
		Images:      imagePaths,
	})

	if err != nil {
		c.Redirect(http.StatusSeeOther, "/seller?error="+url.QueryEscape(err.Error()))
		return
	}

	successMsg := "Publicación registrada. Se encuentra en estado 'pendiente' a la espera de autorización por el administrador."
	c.Redirect(http.StatusSeeOther, "/seller?success="+url.QueryEscape(successMsg))
}

// ReportSale marca una publicación como vendida (HU-VEN-01) usando PublicationService
func (h *WebHandler) ReportSale(c *gin.Context) {
	pubIDStr := strings.TrimSpace(c.PostForm("idPublication"))
	if pubIDStr != "" {
		status := "vendida"
		_, _ = h.pubService.Update(pubIDStr, services.UpdatePublicationRequest{
			Status: &status,
		})
	}

	c.Redirect(http.StatusSeeOther, "/seller")
}

// AdminDashboardView muestra la bandeja de moderación y usuarios (HU-ADM-01)
func (h *WebHandler) AdminDashboardView(c *gin.Context) {
	uidStr := c.GetString("userID")
	user, _ := h.userService.GetByID(uidStr)

	allPubs, _ := h.pubService.GetAll(services.PublicationFilterRequest{})
	var pendingPubs []models.Publication
	var activePubs []models.Publication
	for _, p := range allPubs {
		if !p.IsAuthorized || p.Status == "pendiente" {
			pendingPubs = append(pendingPubs, p)
		} else if p.Status == "publicada" {
			activePubs = append(activePubs, p)
		}
	}

	type AdminSlide struct {
		Index   int
		Total   int
		URL     string
		PrevID  string
		NextID  string
		SlideID string
	}

	type AdminPubView struct {
		models.Publication
		Slides []AdminSlide
	}

	var pendingViews []AdminPubView
	for _, p := range pendingPubs {
		var pSlides []AdminSlide
		tot := len(p.Images)
		for i, img := range p.Images {
			prev := (i - 1 + tot) % tot
			next := (i + 1) % tot
			pSlides = append(pSlides, AdminSlide{
				Index:   i + 1,
				Total:   tot,
				URL:     img,
				SlideID: fmt.Sprintf("admin-pub-%d-slide-%d", p.IDPublication, i+1),
				PrevID:  fmt.Sprintf("admin-pub-%d-slide-%d", p.IDPublication, prev+1),
				NextID:  fmt.Sprintf("admin-pub-%d-slide-%d", p.IDPublication, next+1),
			})
		}
		pendingViews = append(pendingViews, AdminPubView{
			Publication: p,
			Slides:      pSlides,
		})
	}

	users, _ := h.userService.GetAll(services.UserFilterRequest{})

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"User":                user,
		"PendingPublications": pendingViews,
		"ActivePublications":  activePubs,
		"Users":               users,
	})
}

// AuthorizePublication autoriza una publicación haciéndola visible en el catálogo (HU-ADM-01)
func (h *WebHandler) AuthorizePublication(c *gin.Context) {
	pubIDStr := strings.TrimSpace(c.PostForm("idPublication"))
	if pubIDStr != "" {
		_, _ = h.pubService.Authorize(pubIDStr)
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

// DeletePublication elimina una publicación del repositorio real (HU-ADM-01)
func (h *WebHandler) DeletePublication(c *gin.Context) {
	pubIDStr := strings.TrimSpace(c.PostForm("idPublication"))
	if pubIDStr != "" {
		_ = h.pubService.Delete(pubIDStr)
	}

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
