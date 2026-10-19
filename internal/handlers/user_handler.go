package handlers

import (
	"net/http"
	"strings"

	"automarket/internal/middleware"
	"automarket/internal/services"

	"github.com/gin-gonic/gin"
)

// UserHandler actúa como el controlador para autenticación, gestión y vistas web de usuarios.
type UserHandler struct {
	userService services.UserService
}

// NewUserHandler es la función constructora que inyecta el servicio.
func NewUserHandler(us services.UserService) *UserHandler {
	return &UserHandler{
		userService: us,
	}
}

// RegisterRoutes registra las rutas web de autenticación y administración de usuarios.
func (h *UserHandler) RegisterRoutes(router *gin.Engine) {
	// --- Vistas Web y Acciones de Autenticación (Login, Registro y Logout) ---
	router.GET("/login", h.LoginView)
	router.POST("/login", h.ProcessLogin)
	router.GET("/register", h.RegisterView)
	router.POST("/register", h.ProcessRegister)
	router.GET("/logout", h.ProcessLogout)

	// --- Acción Web de Administrador para eliminar usuarios (HU-ADM-01) ---
	adminWeb := router.Group("/admin")
	adminWeb.Use(middleware.RequireAuth())
	adminWeb.Use(middleware.RequireRoles("admin"))
	{
		adminWeb.POST("/users/delete", h.DeleteUserWeb)
	}
}

// --- Métodos de Vistas Web (SSR) ---

// LoginView muestra el formulario de login
func (h *UserHandler) LoginView(c *gin.Context) {
	if uid, role, ok := middleware.GetUserSession(c); ok {
		if u, err := h.userService.GetByID(uid); err == nil && u != nil && u.IsActive {
			if role == "admin" {
				c.Redirect(http.StatusFound, "/admin")
				return
			}
			c.Redirect(http.StatusFound, "/seller")
			return
		}
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
func (h *UserHandler) RegisterView(c *gin.Context) {
	if uid, role, ok := middleware.GetUserSession(c); ok {
		if u, err := h.userService.GetByID(uid); err == nil && u != nil && u.IsActive {
			if role == "admin" {
				c.Redirect(http.StatusFound, "/admin")
				return
			}
			c.Redirect(http.StatusFound, "/seller")
			return
		}
		c.SetCookie("token", "", -1, "/", "", false, true)
	}

	c.HTML(http.StatusOK, "register.html", gin.H{})
}

// ProcessLogin maneja el inicio de sesión desde formulario HTML nativo
func (h *UserHandler) ProcessLogin(c *gin.Context) {
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

	c.SetCookie("token", authResp.Token, 86400, "/", "", false, true)

	if authResp.User.Role == "admin" {
		c.Redirect(http.StatusFound, "/admin")
		return
	}
	c.Redirect(http.StatusFound, "/seller")
}

// ProcessRegister crea un vendedor desde formulario HTML nativo (HU-VEN-01)
func (h *UserHandler) ProcessRegister(c *gin.Context) {
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
func (h *UserHandler) ProcessLogout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// DeleteUserWeb elimina un usuario realmente de data/user.json desde el panel admin web
func (h *UserHandler) DeleteUserWeb(c *gin.Context) {
	userID := strings.TrimSpace(c.PostForm("idUser"))
	if userID != "" {
		_ = h.userService.Delete(userID)
	}

	c.Redirect(http.StatusSeeOther, "/admin")
}
