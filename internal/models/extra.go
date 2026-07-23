package models

type Extra struct {
	ID           string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string  `gorm:"type:uuid;not null;uniqueIndex:idx_tenant_extra_name"`
	Name         string  `gorm:"type:varchar(100);not null;uniqueIndex:idx_tenant_extra_name"`
	CurrentPrice float64 `gorm:"type:decimal(10,2);not null"`
	CurrentStock int     `gorm:"type:int;not null;default:0"`
	TrackStock   bool    `gorm:"type:boolean;not null;default:true"`
	IsActive     bool    `gorm:"type:boolean;not null;default:true"`
}
