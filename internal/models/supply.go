package models

type Supply struct {
	ID                   string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID             string  `gorm:"type:uuid;not null;uniqueIndex:idx_tenant_supply_name"`
	Name                 string  `gorm:"type:varchar(100);not null;uniqueIndex:idx_tenant_supply_name"`
	CurrentWholesaleCost float64 `gorm:"type:decimal(10,2);not null"`
	CurrentStock         float64 `gorm:"type:numeric(12,3);not null;default:0.000"`
	MeasurementUnit      string  `gorm:"type:varchar(20);not null"`
	IsActive             bool    `gorm:"type:boolean;not null;default:true"`
}
