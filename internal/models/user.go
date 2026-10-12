package models

import "time"

// User representa la entidad USER en la base de datos.
type User struct {
	IDUser    int       `json:"idUser"`
	Email     string    `json:"email"`
	Password  string    `json:"password,omitempty"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"createdAt"`
	IsActive  bool      `json:"isActive"`
}
