package models

import "time"

type Employee struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string    `gorm:"type:uuid;not null"`
	FirstName    string    `gorm:"type:varchar(50);not null"`
	LastName     string    `gorm:"type:varchar(50);not null"`
	Email        string    `gorm:"type:varchar(150);unique;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	Phone        string    `gorm:"type:varchar(20)"`
	Role         string    `gorm:"type:varchar(20);not null;default:'employee'"`
	IsActive     bool      `gorm:"type:boolean;not null;default:true"`
	CreatedAt    time.Time `gorm:"type:timestamp with time zone;default:now()"`
}
