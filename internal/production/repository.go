package production

import (
    "context"
    "errors"
    "time"

    "leguiburger/internal/db"
    "leguiburger/internal/models"

    "gorm.io/gorm"
)

type Repository interface {
    Create(ctx context.Context, report *models.ProductionReport) error
    GetByID(ctx context.Context, tenantID, id string) (*models.ProductionReport, error)
    GetByDate(ctx context.Context, tenantID string, productionDate time.Time) (*models.ProductionReport, error)
    FetchAll(ctx context.Context, tenantID string, startDate, endDate *string) ([]models.ProductionReport, error)
    Update(ctx context.Context, report *models.ProductionReport) error
    Delete(ctx context.Context, tenantID, id string) error
}

type repository struct{}

func NewRepository() Repository {
    return &repository{}
}

func (r *repository) Create(ctx context.Context, report *models.ProductionReport) error {
    return db.DB.WithContext(ctx).Create(report).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (*models.ProductionReport, error) {
    var report models.ProductionReport
    err := db.DB.WithContext(ctx).
        Where("tenant_id = ? AND id = ?", tenantID, id).
        First(&report).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return &report, nil
}

func (r *repository) GetByDate(ctx context.Context, tenantID string, productionDate time.Time) (*models.ProductionReport, error) {
    var report models.ProductionReport
    err := db.DB.WithContext(ctx).
        Where("tenant_id = ? AND production_date = ?", tenantID, productionDate).
        First(&report).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return &report, nil
}

func (r *repository) FetchAll(ctx context.Context, tenantID string, startDate, endDate *string) ([]models.ProductionReport, error) {
    var reports []models.ProductionReport
    query := db.DB.WithContext(ctx).Where("tenant_id = ?", tenantID)
    if startDate != nil {
        query = query.Where("production_date >= ?", *startDate)
    }
    if endDate != nil {
        query = query.Where("production_date <= ?", *endDate)
    }
    err := query.Order("production_date DESC").Find(&reports).Error
    return reports, err
}

func (r *repository) Update(ctx context.Context, report *models.ProductionReport) error {
    return db.DB.WithContext(ctx).Save(report).Error
}

func (r *repository) Delete(ctx context.Context, tenantID, id string) error {
    return db.DB.WithContext(ctx).
        Where("tenant_id = ? AND id = ?", tenantID, id).
        Delete(&models.ProductionReport{}).Error
}
