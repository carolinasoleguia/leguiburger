package models

type TenantProduct struct {
	TenantID      string   `gorm:"primaryKey;type:uuid" json:"tenant_id"`
	ProductID     string   `gorm:"primaryKey;type:uuid" json:"product_id"`
	Tenant        Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Product       Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	PriceOverride *float64 `gorm:"type:decimal(10,2)" json:"price_override,omitempty"`
	CurrentStock  int      `gorm:"type:int;not null;default:0" json:"current_stock"`
	TrackStock    bool     `gorm:"type:boolean;not null;default:true" json:"track_stock"`
	IsAvailable   bool     `gorm:"type:boolean;not null;default:true" json:"is_available"`
	IsActive      bool     `gorm:"type:boolean;not null;default:true" json:"is_active"`
}
