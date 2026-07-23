package models

import (
	"time"
)

type Tenant struct {
	ID        string    `gorm:"primaryKey;type:uuid" json:"id"`
	Name      string    `gorm:"type:varchar(100)" json:"name"`
	Subdomain string    `gorm:"type:varchar(100);uniqueIndex:idx_domain_subdomain" json:"subdomain"`
	TaxID     string    `gorm:"type:varchar(20)" json:"tax_id"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
