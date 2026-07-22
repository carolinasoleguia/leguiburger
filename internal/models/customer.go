package models

import "time"

type Customer struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string    `gorm:"type:uuid;not null;uniqueIndex:idx_tenant_customer_email"`
	FirstName string    `gorm:"type:varchar(50);not null"`
	LastName  string    `gorm:"type:varchar(50);not null"`
	Email     string    `gorm:"type:varchar(150);not null;uniqueIndex:idx_tenant_customer_email"`
	Phone     string    `gorm:"type:varchar(20)"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone;default:now()"`
}
