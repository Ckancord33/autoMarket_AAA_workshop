package repositories

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"automarket/internal/models"
)

var (
	ErrAuditLogNotFound = errors.New("registro de auditoría no encontrado")
)

// AuditRepository gestiona la persistencia y lectura de los eventos de auditoría (AAA)
// utilizando sincronización en memoria (sync.RWMutex) y almacenamiento en disco en formato JSON.
// Emplea un slice para preservar el orden cronológico estricto y un map para búsquedas por ID en O(1).
type AuditRepository struct {
	filePath string
	mu       sync.RWMutex
	logs     []models.AuditLog         // Preserva el orden cronológico natural de los eventos
	logIndex map[int64]models.AuditLog // Índice primario en memoria por ID: O(1)
	lastID   int64
}

// NewAuditRepository inicializa el repositorio, asegurando el archivo JSON y precargando los logs en memoria.
func NewAuditRepository(filePath string) *AuditRepository {
	repo := &AuditRepository{
		filePath: filePath,
		logs:     make([]models.AuditLog, 0),
		logIndex: make(map[int64]models.AuditLog),
		lastID:   0,
	}

	_ = repo.ensureFileAndLoad()
	return repo
}

// ensureFileAndLoad garantiza que el archivo exista e inicializa la lista y el mapa de índices en memoria.
func (r *AuditRepository) ensureFileAndLoad() error {
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

	var loadedLogs []models.AuditLog
	if err := json.Unmarshal(data, &loadedLogs); err != nil {
		return err
	}

	r.logs = loadedLogs
	for _, l := range r.logs {
		r.logIndex[l.IDLog] = l
		if l.IDLog > r.lastID {
			r.lastID = l.IDLog
		}
	}

	return nil
}

// persistLocked guarda el estado de memoria en el archivo JSON (debe invocarse con r.mu.Lock activo).
func (r *AuditRepository) persistLocked() error {
	data, err := json.MarshalIndent(r.logs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

// Create registra de forma atómica e inmutable un nuevo evento de auditoría en memoria y disco.
func (r *AuditRepository) Create(entry *models.AuditLog) (*models.AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	entry.IDLog = r.lastID

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Guardar en la secuencia cronológica y en el índice O(1)
	r.logs = append(r.logs, *entry)
	r.logIndex[entry.IDLog] = *entry

	if err := r.persistLocked(); err != nil {
		return nil, err
	}

	saved := *entry
	return &saved, nil
}

// GetAll retorna todos los registros de auditoría en orden cronológico.
func (r *AuditRepository) GetAll() ([]models.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.AuditLog, len(r.logs))
	copy(result, r.logs)
	return result, nil
}

// GetByID busca un registro de auditoría por su ID único en tiempo constante O(1).
func (r *AuditRepository) GetByID(id int64) (*models.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	log, exists := r.logIndex[id]
	if !exists {
		return nil, ErrAuditLogNotFound
	}

	return &log, nil
}

// GetByUserID retorna todos los eventos generados por un usuario específico.
func (r *AuditRepository) GetByUserID(userID int) ([]models.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var userLogs []models.AuditLog
	for _, l := range r.logs {
		if l.IDUser != nil && *l.IDUser == userID {
			userLogs = append(userLogs, l)
		}
	}

	return userLogs, nil
}
