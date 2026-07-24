package models

type Supply struct {
	ID                   string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TenantID             string  `gorm:"type:uuid;not null;uniqueIndex:idx_tenant_supply_name" json:"tenant_id"`
	Name                 string  `gorm:"type:varchar(100);not null;uniqueIndex:idx_tenant_supply_name" json:"name"`
	CurrentWholesaleCost float64 `gorm:"type:decimal(10,2);not null" json:"current_wholesale_cost"`
	CurrentStock         float64 `gorm:"type:numeric(12,3);not null;default:0.000" json:"current_stock"`
	MeasurementUnit      string  `gorm:"type:varchar(20);not null" json:"measurement_unit"`
	IsActive             bool    `gorm:"type:boolean;not null;default:true" json:"is_active"`
}
