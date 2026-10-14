package services

import (
	"errors"
	"strconv"
	"strings"

	"automarket/internal/models"
	"automarket/internal/repositories"
	"automarket/pkg/security"
)

// DTOs que espera el Handler
type CreateUserRequest struct {
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required,min=6"`
	Name     string `json:"name" form:"name" binding:"required"`
	Phone    string `json:"phone" form:"phone"`
	City     string `json:"city" form:"city"`
}

type LoginRequest struct {
	Email    string `json:"email" form:"email"`
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password" binding:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

type UpdateUserRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	City     string `json:"city"`
	Password string `json:"password"`
}

type UserFilterRequest struct {
	Role string `form:"role"`
	City string `form:"city"`
}

// UserService define el contrato de métodos que los Handlers necesitan invocar
type UserService interface {
	Create(req CreateUserRequest) (*models.User, error)
	Register(req CreateUserRequest) (*AuthResponse, error)
	Login(req LoginRequest) (*AuthResponse, error)
	GetByID(id string) (*models.User, error)
	GetAll(filters UserFilterRequest) ([]models.User, error)
	Update(id string, req UpdateUserRequest) (*models.User, error)
	Delete(id string) error
}

// userService es la implementación concreta de UserService
type userService struct {
	repo *repositories.UserRepository
}

// NewUserService crea e inyecta la dependencia del repositorio en el servicio
func NewUserService(repo *repositories.UserRepository) UserService {
	return &userService{
		repo: repo,
	}
}

// Create valida los datos de entrada, aplica reglas de negocio y persiste mediante el UserRepository
func (s *userService) Create(req CreateUserRequest) (*models.User, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	name := strings.TrimSpace(req.Name)

	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("el correo electrónico no es válido")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("la contraseña debe tener al menos 6 caracteres")
	}
	if name == "" {
		return nil, errors.New("el nombre es obligatorio")
	}

	// Por seguridad (prevención de escalada de privilegios), todo usuario creado
	// mediante petición HTTP se registra obligatoriamente con el rol "seller".
	// El rol "admin" se provisiona directamente en el archivo de datos/proceso interno.
	role := models.RoleSeller

	// Hashear la contraseña con salt criptográfico antes de persistir
	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("error al procesar la seguridad de la contraseña")
	}

	newUser := &models.User{
		Email:    email,
		Password: hashedPassword,
		Name:     name,
		Role:     role,
		Phone:    strings.TrimSpace(req.Phone),
		City:     strings.TrimSpace(req.City),
		IsActive: true,
	}

	created, err := s.repo.Create(newUser)
	if err != nil {
		return nil, err
	}

	// Ocultar la contraseña por seguridad antes de devolver la respuesta
	sanitized := *created
	sanitized.Password = ""
	return &sanitized, nil
}

// Register crea un usuario y genera su token JWT de sesión inmediatamente (Autologin)
func (s *userService) Register(req CreateUserRequest) (*AuthResponse, error) {
	user, err := s.Create(req)
	if err != nil {
		return nil, err
	}

	token, err := security.GenerateJWT(strconv.Itoa(user.IDUser), user.Role)
	if err != nil {
		return nil, errors.New("usuario creado pero falló la generación del token de sesión")
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

// Login valida credenciales mediante hash+salt y emite un token JWT de sesión
func (s *userService) Login(req LoginRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" && req.Username != "" {
		email = strings.TrimSpace(strings.ToLower(req.Username))
	}

	if email == "" {
		return nil, errors.New("el correo electrónico o usuario es requerido")
	}

	if strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("la contraseña es requerida")
	}

	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, errors.New("credenciales incorrectas")
	}

	if !user.IsActive {
		return nil, errors.New("la cuenta de usuario se encuentra inactiva")
	}

	if !security.CheckPassword(req.Password, user.Password) {
		return nil, errors.New("credenciales incorrectas")
	}

	token, err := security.GenerateJWT(strconv.Itoa(user.IDUser), user.Role)
	if err != nil {
		return nil, errors.New("error al generar el token de sesión")
	}

	sanitized := *user
	sanitized.Password = ""

	return &AuthResponse{
		Token: token,
		User:  &sanitized,
	}, nil
}

// GetByID convierte el id de string a entero y busca en el repositorio O(1)
func (s *userService) GetByID(idStr string) (*models.User, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return nil, errors.New("el ID del usuario debe ser un número entero positivo")
	}

	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	sanitized := *user
	sanitized.Password = ""
	return &sanitized, nil
}

// GetAll obtiene todos los usuarios del repositorio y aplica filtros opcionales (Role, City)
func (s *userService) GetAll(filters UserFilterRequest) ([]models.User, error) {
	users, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	result := make([]models.User, 0, len(users))
	for _, u := range users {
		// Filtrado opcional por Rol
		if filters.Role != "" && !strings.EqualFold(u.Role, filters.Role) {
			continue
		}
		// Filtrado opcional por Ciudad
		if filters.City != "" && !strings.EqualFold(u.City, filters.City) {
			continue
		}

		u.Password = ""
		result = append(result, u)
	}

	return result, nil
}

// Update valida el ID, actualiza los campos permitidos y los persiste a través del repositorio
func (s *userService) Update(idStr string, req UpdateUserRequest) (*models.User, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return nil, errors.New("el ID del usuario debe ser un número entero positivo")
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Actualizar solo los campos que vienen en la petición (soporte de PATCH)
	if strings.TrimSpace(req.Name) != "" {
		existing.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Phone) != "" {
		existing.Phone = strings.TrimSpace(req.Phone)
	}
	if strings.TrimSpace(req.City) != "" {
		existing.City = strings.TrimSpace(req.City)
	}

	if strings.TrimSpace(req.Password) != "" {
		if len(req.Password) < 6 {
			return nil, errors.New("la nueva contraseña debe tener al menos 6 caracteres")
		}
		hashedPassword, err := security.HashPassword(req.Password)
		if err != nil {
			return nil, errors.New("error al procesar la seguridad de la contraseña")
		}
		existing.Password = hashedPassword
	}

	updated, err := s.repo.Update(id, existing)
	if err != nil {
		return nil, err
	}

	sanitized := *updated
	sanitized.Password = ""
	return &sanitized, nil
}

// Delete valida el identificador y elimina un usuario a través del repositorio
func (s *userService) Delete(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return errors.New("el ID del usuario debe ser un número entero positivo")
	}
	return s.repo.Delete(id)
}
