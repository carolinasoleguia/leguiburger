package models

import "time"

type Product struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BrandID     string    `gorm:"type:uuid;not null;uniqueIndex:idx_brand_product_name" json:"brand_id"`
	Brand       Brand     `gorm:"foreignKey:BrandID" json:"brand,omitempty"`
	Name        string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_brand_product_name" json:"name"`
	Description string    `gorm:"type:text"`
	BasePrice   float64   `gorm:"type:decimal(10,2);not null" json:"base_price"`
	ImageURL    string    `gorm:"type:varchar(255)" json:"image_url"`
	IsActive    bool      `gorm:"type:boolean;not null;default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
}
