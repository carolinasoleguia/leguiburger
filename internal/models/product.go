package models

import "time"

type Product struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string    `gorm:"type:uuid;not null;uniqueIndex:idx_tenant_product_name"`
	Name         string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_tenant_product_name"`
	Description  string    `gorm:"type:text"`
	CurrentPrice float64   `gorm:"type:decimal(10,2);not null"`
	CurrentStock int       `gorm:"type:int;not null;default:0"`
	TrackStock   bool      `gorm:"type:boolean;not null;default:true"`
	ImageURL     string    `gorm:"type:varchar(255)"`
	IsActive     bool      `gorm:"type:boolean;not null;default:true"`
	CreatedAt    time.Time `gorm:"type:timestamp with time zone;default:now()"`
}
