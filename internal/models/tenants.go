package models

import (
	"time"
)

type Tenant struct {
	ID string `gorm:"primaryKey;type:uuid" json:"id"`

	BrandID   string    `gorm:"type:uuid;not null" json:"brand_id"`
	Brand     Brand     `gorm:"foreignKey:BrandID" json:"brand"`
	Subdomain string    `gorm:"type:varchar(100);uniqueIndex:idx_domain_subdomain" json:"subdomain"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
