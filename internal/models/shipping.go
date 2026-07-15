package models

import (
	"time"
)

type ShippingMethod struct {
	ID            string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string    `gorm:"type:uuid;not null;uniqueIndex:idx_tenant_shipping_name_typ"`
	Name          string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_tenant_shipping_name_typ"`
	Typification  string    `gorm:"type:varchar(50);not null;default:'DELIVERY';uniqueIndex:idx_tenant_shipping_name_typ"`
	Description   string    `gorm:"type:varchar(100);not null"`
	Cost          float64   `gorm:"type:decimal(10,2);not null;default:0.00"`
	EstimatedTime string    `gorm:"type:varchar(50)"`
	IsActive      bool      `gorm:"type:boolean;not null;default:true"`
	CreatedAt     time.Time `gorm:"type:timestamp with time zone;default:now()"`
}
