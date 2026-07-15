package models

import (
	"time"
)

type Tenant struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name" validate:"required"`
	Subdomain string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"subdomain" validate:"required"`
	TaxID     string    `gorm:"type:varchar(100);not null" json:"tax_id" validate:"required"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
