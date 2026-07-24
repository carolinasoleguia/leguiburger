package models

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	FirstName    string    `gorm:"type:varchar(50);not null" json:"first_name"`
	LastName     string    `gorm:"type:varchar(50);not null" json:"last_name"`
	Email        string    `gorm:"type:varchar(150);unique;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Role         string    `gorm:"type:varchar(50);not null" json:"role"`
	BrandID      *string   `gorm:"type:uuid" json:"brand_id,omitempty"`
	IsActive     bool      `gorm:"type:boolean;not null;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:timestamp with time zone;default:now()" json:"updated_at"`
}
