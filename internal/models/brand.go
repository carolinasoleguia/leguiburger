package models

import "time"

type Brand struct {
	ID        string    `gorm:"primaryKey;type:uuid" json:"id"`
	Name      string    `gorm:"type:varchar(100)" json:"name"`
	TaxID     string    `gorm:"type:varchar(20)" json:"tax_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Tenants   []Tenant  `gorm:"foreignKey:BrandID" json:"tenants,omitempty"`
}
