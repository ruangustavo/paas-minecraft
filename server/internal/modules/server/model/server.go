package model

import (
	"time"

	"github.com/google/uuid"
)

type Server struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Port      int       `gorm:"not null;default:25565" json:"port"`
	Subdomain string    `gorm:"size:255" json:"subdomain"`
	Status    string    `gorm:"size:20;not null;default:'running'" json:"status"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
