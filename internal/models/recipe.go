package models

type Recipe struct {
	ProductID    string  `gorm:"primaryKey;type:uuid"`
	SupplyID     string  `gorm:"primaryKey;type:uuid"`
	QuantityUsed float64 `gorm:"type:decimal(10,3);not null"`
}
