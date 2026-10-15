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
	ErrPublicationNotFound = errors.New("publicación no encontrada")
)

// PublicationRepository gestiona la persistencia de las publicaciones de vehículos
// empleando almacenamiento en disco (JSON) con índices en memoria para lecturas O(1)
// y exclusión mutua mediante sync.RWMutex (exclusivamente Go Standard Library).
type PublicationRepository struct {
	filePath string
	mu       sync.RWMutex
	pubs     map[int]models.Publication // Índice primario en memoria por ID: O(1)
	lastID   int                        // Rastreador del ID más alto para autoincremento O(1)
}

// NewPublicationRepository inicializa el repositorio, cargando los datos a memoria
func NewPublicationRepository(filePath string) *PublicationRepository {
	repo := &PublicationRepository{
		filePath: filePath,
		pubs:     make(map[int]models.Publication),
		lastID:   0,
	}

	_ = repo.ensureFileAndLoad()
	return repo
}

// ensureFileAndLoad garantiza la existencia de data/publication.json y puebla el mapa en memoria
func (r *PublicationRepository) ensureFileAndLoad() error {
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

	var pubList []models.Publication
	if err := json.Unmarshal(data, &pubList); err != nil {
		return err
	}

	// Cargar publicaciones en memoria y sincronizar el último ID
	for _, p := range pubList {
		r.pubs[p.IDPublication] = p
		if p.IDPublication > r.lastID {
			r.lastID = p.IDPublication
		}
	}

	return nil
}

// persistLocked guarda el estado de la memoria en data/publication.json (requiere r.mu.Lock activo)
func (r *PublicationRepository) persistLocked() error {
	list := make([]models.Publication, 0, len(r.pubs))
	for _, p := range r.pubs {
		list = append(list, p)
	}

	// Ordenar por ID para mantener el JSON consistente
	sort.Slice(list, func(i, j int) bool {
		return list[i].IDPublication < list[j].IDPublication
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

// GetAll retorna todas las publicaciones almacenadas ordenadas de más reciente a más antigua
func (r *PublicationRepository) GetAll() ([]models.Publication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]models.Publication, 0, len(r.pubs))
	for _, p := range r.pubs {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].IDPublication > list[j].IDPublication
	})

	return list, nil
}

// GetByID busca una publicación por su ID en tiempo constante O(1)
func (r *PublicationRepository) GetByID(id int) (*models.Publication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pub, exists := r.pubs[id]
	if !exists {
		return nil, ErrPublicationNotFound
	}

	return &pub, nil
}

// GetByUserID retorna las publicaciones pertenecientes a un vendedor específico
func (r *PublicationRepository) GetByUserID(userID int) ([]models.Publication, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var userPubs []models.Publication
	for _, p := range r.pubs {
		if p.IDUser == userID {
			userPubs = append(userPubs, p)
		}
	}

	sort.Slice(userPubs, func(i, j int) bool {
		return userPubs[i].IDPublication > userPubs[j].IDPublication
	})

	return userPubs, nil
}

// Create asigna un ID autoincremental, almacena en memoria y persiste en JSON
func (r *PublicationRepository) Create(pub *models.Publication) (*models.Publication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	pub.IDPublication = r.lastID

	if pub.PublishedAt.IsZero() {
		pub.PublishedAt = time.Now()
	}

	// Guardar en memoria
	r.pubs[pub.IDPublication] = *pub

	// Persistir en disco
	if err := r.persistLocked(); err != nil {
		delete(r.pubs, pub.IDPublication)
		r.lastID--
		return nil, err
	}

	saved := *pub
	return &saved, nil
}

// Update actualiza una publicación existente en memoria y disco
func (r *PublicationRepository) Update(pub *models.Publication) (*models.Publication, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pubs[pub.IDPublication]; !exists {
		return nil, ErrPublicationNotFound
	}

	r.pubs[pub.IDPublication] = *pub

	if err := r.persistLocked(); err != nil {
		return nil, err
	}

	saved := *pub
	return &saved, nil
}

// Delete elimina una publicación por su ID
func (r *PublicationRepository) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pubs[id]; !exists {
		return ErrPublicationNotFound
	}

	delete(r.pubs, id)
	return r.persistLocked()
}
