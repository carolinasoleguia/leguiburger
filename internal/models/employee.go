package models

import "time"

type Employee struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TenantID     *string   `gorm:"type:uuid" json:"tenant_id,omitempty"`
	FirstName    string    `gorm:"type:varchar(50);not null" json:"first_name"`
	LastName     string    `gorm:"type:varchar(50);not null" json:"last_name"`
	Email        string    `gorm:"type:varchar(150);unique;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Phone        string    `gorm:"type:varchar(20)" json:"phone"`
	Role         string    `gorm:"type:varchar(20);not null;default:'employee'" json:"role"`
	IsActive     bool      `gorm:"type:boolean;not null;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
}
