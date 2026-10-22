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

	"github.com/gin-gonic/gin"
)

type PublicationHandler struct {
	publicationService services.PublicationService
	userService        services.UserService
}

func NewPublicationHandler(ps services.PublicationService, us services.UserService) *PublicationHandler {
	return &PublicationHandler{
		publicationService: ps,
		userService:        us,
	}
}

func (h *PublicationHandler) RegisterRoutes(router *gin.Engine) {
	// Catálogo público y ficha técnica (HU-VIS-01)
	router.GET("/", h.IndexView)
	router.GET("/publications/:id", h.PublicationDetailView)

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
	}
}

// IndexView renderiza el catálogo público (HU-VIS-01) consultando las publicaciones reales
func (h *PublicationHandler) IndexView(c *gin.Context) {
	allPubs, err := h.publicationService.GetAll(services.PublicationFilterRequest{})
	if err != nil {
		allPubs = []models.Publication{}
	}

	var visiblePubs []models.Publication
	for _, p := range allPubs {
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
func (h *PublicationHandler) PublicationDetailView(c *gin.Context) {
	idStr := c.Param("id")

	found, err := h.publicationService.GetByID(idStr)
	if err != nil || found == nil {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// Obtener datos del vendedor asociado
	seller, _ := h.userService.GetByID(strconv.Itoa(found.IDUser))
	if seller == nil {
		seller = &models.User{
			Name: "Vendedor no disponible",
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

// SellerDashboardView muestra el panel del vendedor con sus publicaciones (HU-VEN-01)
func (h *PublicationHandler) SellerDashboardView(c *gin.Context) {
	uidStr := c.GetString("userID")
	uid, _ := strconv.Atoi(uidStr)

	user, err := h.userService.GetByID(uidStr)
	if err != nil || user == nil {
		c.SetCookie("token", "", -1, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/login?error="+url.QueryEscape("Tu cuenta ha sido eliminada por el administrador."))
		return
	}

	allPubs, _ := h.publicationService.GetAll(services.PublicationFilterRequest{})
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

// CreatePublication registra una nueva publicación en estado "pendiente" (HU-VEN-01) desde formulario web
func (h *PublicationHandler) CreatePublication(c *gin.Context) {
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

	_, err := h.publicationService.Create(services.CreatePublicationRequest{
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
func (h *PublicationHandler) ReportSale(c *gin.Context) {
	pubIDStr := strings.TrimSpace(c.PostForm("idPublication"))
	if pubIDStr != "" {
		status := "vendida"
		_, _ = h.publicationService.Update(pubIDStr, services.UpdatePublicationRequest{
			Status: &status,
		})
	}

	c.Redirect(http.StatusSeeOther, "/seller")
}

// AdminDashboardView muestra la bandeja de moderación y usuarios (HU-ADM-01)
func (h *PublicationHandler) AdminDashboardView(c *gin.Context) {
	uidStr := c.GetString("userID")
	user, _ := h.userService.GetByID(uidStr)

	allPubs, _ := h.publicationService.GetAll(services.PublicationFilterRequest{})
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

// AuthorizePublication autoriza una publicación desde el panel web (HU-ADM-01)
func (h *PublicationHandler) AuthorizePublication(c *gin.Context) {
	pubIDStr := strings.TrimSpace(c.PostForm("idPublication"))
	if pubIDStr != "" {
		_, _ = h.publicationService.Authorize(pubIDStr)
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}

// DeletePublication elimina una publicación desde el panel web (HU-ADM-01)
func (h *PublicationHandler) DeletePublication(c *gin.Context) {
	pubIDStr := strings.TrimSpace(c.PostForm("idPublication"))
	if pubIDStr != "" {
		_ = h.publicationService.Delete(pubIDStr)
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}
