package models

import "time"

type ProductionReport struct {
    ID                 string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    TenantID           string     `gorm:"type:uuid;not null;index:idx_tenant_production_date,unique" json:"tenant_id"`
    ProductionDate     time.Time  `gorm:"type:date;not null;index:idx_tenant_production_date,unique" json:"production_date"`
    MedallionsProduced int        `gorm:"not null;default:0" json:"medallions_produced"`
    BreadsPurchased    int        `gorm:"not null;default:0" json:"breads_purchased"`
    Notes              string     `gorm:"type:text" json:"notes,omitempty"`
    CreatedBy          *string    `gorm:"type:uuid" json:"created_by,omitempty"`
    CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
    UpdatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}
