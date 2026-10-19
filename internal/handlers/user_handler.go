package handlers

import (
	"net/http"
	"strings"

	"automarket/internal/middleware"
	"automarket/internal/services"
	"automarket/pkg/response"

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

// RegisterRoutes registra las rutas web de autenticación y los endpoints de la API de usuarios.
func (h *UserHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	// --- Vistas Web y Acciones de Autenticación (Login, Registro y Logout) ---
	router.GET("/login", h.LoginView)
	router.POST("/login", h.ProcessLogin)
	router.GET("/register", h.RegisterView)
	router.POST("/register", h.ProcessRegister)
	router.GET("/logout", h.ProcessLogout)

	// --- Acción Web de Administrador para eliminar usuarios ---
	adminWeb := router.Group("/admin")
	adminWeb.Use(middleware.RequireAuth())
	adminWeb.Use(middleware.RequireRoles("admin"))
	{
		adminWeb.POST("/users/delete", h.DeleteUserWeb)
	}

	// --- Rutas Públicas de la API REST de Usuarios (/users) ---
	userGroup := router.Group("/users")
	userGroup.POST("/", h.CreateUser)
	userGroup.POST("/login", h.Login)

	// --- Rutas Protegidas de Administración de la API (/users/:id) ---
	protected := userGroup.Group("/")
	if authMiddleware != nil {
		protected.Use(authMiddleware)
	} else {
		protected.Use(middleware.RequireAuth())
	}
	protected.Use(middleware.RequireRoles("admin"))
	{
		protected.DELETE("/:id", h.DeleteUser)
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

// --- Métodos de API REST JSON (/users) ---

// CreateUser maneja la petición POST para registrar un usuario
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest

	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Datos de registro inválidos", err)
		return
	}

	authResp, err := h.userService.Register(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Error al registrar usuario", err)
		return
	}

	response.Success(c, http.StatusCreated, "Usuario registrado exitosamente", authResp)
}

// Login maneja la petición POST para autenticar un usuario y devolver el token JWT
func (h *UserHandler) Login(c *gin.Context) {
	var req services.LoginRequest

	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Formato de credenciales inválido", err)
		return
	}

	authResp, err := h.userService.Login(req)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Credenciales inválidas", err)
		return
	}

	response.Success(c, http.StatusOK, "Inicio de sesión exitoso", authResp)
}

// DeleteUser maneja la petición DELETE para que un administrador elimine un usuario.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	if err := h.userService.Delete(id); err != nil {
		response.Error(c, http.StatusNotFound, "Error al eliminar usuario", err)
		return
	}

	response.Success(c, http.StatusOK, "Usuario eliminado exitosamente", nil)
}
