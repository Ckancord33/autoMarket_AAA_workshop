package repositories

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"

	"automarket/internal/models"
)

var (
	ErrUserNotFound      = errors.New("usuario no encontrado")
	ErrUserAlreadyExists = errors.New("el correo electrónico ya se encuentra registrado")
)

type UserRepository struct {
	filePath   string
	mu         sync.RWMutex
	users      map[int]models.User // Índice primario en memoria por ID: O(1)
	emailIndex map[string]int      // Índice secundario en memoria por Email: O(1)
	lastID     int                 // Rastreador del ID más alto para autoincremento O(1)
}

// NewUserRepository crea una nueva instancia del repositorio con índices en memoria y persistencia JSON
func NewUserRepository(filePath string) *UserRepository {
	repo := &UserRepository{
		filePath:   filePath,
		users:      make(map[int]models.User),
		emailIndex: make(map[string]int),
		lastID:     0,
	}

	// Cargar datos a memoria una sola vez al arrancar
	_ = repo.ensureFileAndLoad()
	return repo
}

// ensureFileAndLoad garantiza que el archivo exista e inicializa los índices en memoria
func (r *UserRepository) ensureFileAndLoad() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, err := os.Stat(r.filePath)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		if err := os.WriteFile(r.filePath, []byte("[]"), 0644); err != nil {
			return err
		}
		return nil
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var userList []models.User
	if err := json.Unmarshal(data, &userList); err != nil {
		return err
	}

	// Poblar los índices en memoria
	for _, u := range userList {
		r.users[u.IDUser] = u
		r.emailIndex[u.Email] = u.IDUser
		if u.IDUser > r.lastID {
			r.lastID = u.IDUser
		}
	}

	return nil
}

// persistLocked guarda el estado de la memoria en data/user.json (debe llamarse con r.mu.Lock activo)
func (r *UserRepository) persistLocked() error {
	list := make([]models.User, 0, len(r.users))
	for _, u := range r.users {
		list = append(list, u)
	}

	// Ordenar por ID para mantener el archivo JSON consistente y predecible
	sort.Slice(list, func(i, j int) bool {
		return list[i].IDUser < list[j].IDUser
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

// GetAll retorna todos los usuarios almacenados en memoria ordenados por ID
func (r *UserRepository) GetAll() ([]models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]models.User, 0, len(r.users))
	for _, u := range r.users {
		list = append(list, u)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].IDUser < list[j].IDUser
	})

	return list, nil
}

// GetByID busca un usuario por ID directamente en el índice en memoria O(1)
func (r *UserRepository) GetByID(id int) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// GetByEmail busca un usuario por Email directamente en el índice en memoria O(1)
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.emailIndex[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	user := r.users[id]
	return &user, nil
}

// Create inserta un nuevo usuario en memoria O(1) y persiste en data/user.json
func (r *UserRepository) Create(user *models.User) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validación instantánea de email duplicado O(1)
	if _, exists := r.emailIndex[user.Email]; exists {
		return nil, ErrUserAlreadyExists
	}

	// Autoincremento O(1)
	r.lastID++
	user.IDUser = r.lastID

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	// Actualizar índices en memoria
	r.users[user.IDUser] = *user
	r.emailIndex[user.Email] = user.IDUser

	// Persistir cambios al disco
	if err := r.persistLocked(); err != nil {
		// Rollback en memoria si falla la escritura a disco
		delete(r.users, user.IDUser)
		delete(r.emailIndex, user.Email)
		r.lastID--
		return nil, err
	}

	return user, nil
}

// Update modifica un usuario existente en memoria O(1) y persiste en disco
func (r *UserRepository) Update(id int, updated *models.User) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Si cambió el correo, validar que no colisione con otro usuario existente
	if updated.Email != current.Email {
		if otherID, exists := r.emailIndex[updated.Email]; exists && otherID != id {
			return nil, ErrUserAlreadyExists
		}
	}

	// Preservar ID y fecha de creación original
	updated.IDUser = id
	if updated.CreatedAt.IsZero() {
		updated.CreatedAt = current.CreatedAt
	}

	// Si no enviaron nueva contraseña, conservar la anterior
	if updated.Password == "" {
		updated.Password = current.Password
	}

	// Actualizar índices en memoria
	oldEmail := current.Email
	if oldEmail != updated.Email {
		delete(r.emailIndex, oldEmail)
		r.emailIndex[updated.Email] = id
	}
	r.users[id] = *updated

	// Persistir a disco
	if err := r.persistLocked(); err != nil {
		// Revertir en memoria en caso de error de disco
		r.users[id] = current
		if oldEmail != updated.Email {
			delete(r.emailIndex, updated.Email)
			r.emailIndex[oldEmail] = id
		}
		return nil, err
	}

	return updated, nil
}

// Delete remueve un usuario de memoria O(1) y sincroniza con data/user.json
func (r *UserRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return ErrUserNotFound
	}

	// Eliminar de índices en memoria
	delete(r.users, id)
	delete(r.emailIndex, user.Email)

	// Persistir cambios en disco
	if err := r.persistLocked(); err != nil {
		// Revertir en memoria si falla el disco
		r.users[id] = user
		r.emailIndex[user.Email] = id
		return err
	}

	return nil
}
