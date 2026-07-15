package models

import (
	"time"
)

type Tenant struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	Name      string    `gorm:"type:varchar(100)"`
	Subdomain string    `gorm:"type:varchar(100);uniqueIndex:idx_domain_subdomain"`
	TaxID     string    `gorm:"type:varchar(20)"`
	Active    bool      `gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
