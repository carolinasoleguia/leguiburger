package operations

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Repository struct{ DB *gorm.DB }

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{DB: db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})}
}
func jsonQuery(db *gorm.DB, query string, args ...any) (json.RawMessage, error) {
	var row struct{ Payload string }
	result := db.Raw(query, args...).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || row.Payload == "" {
		return nil, notFound()
	}
	return json.RawMessage(row.Payload), nil
}
func (r *Repository) JSON(ctx context.Context, query string, args ...any) (json.RawMessage, error) {
	return jsonQuery(r.DB.WithContext(ctx), query, args...)
}
