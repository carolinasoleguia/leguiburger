package production

import (
    "context"
    "errors"
    "strings"
    "time"

    "leguiburger/internal/models"
    "leguiburger/internal/tenants"
)

var (
    ErrProductionNotFound        = errors.New("registro de producción no encontrado")
    ErrInvalidProductionData     = errors.New("datos de producción invalidos")
    ErrDuplicateProductionDate   = errors.New("ya existe un registro para esa fecha")
    ErrTenantNotFoundForProduction = errors.New("el comercio especificado no existe")
)

type Service interface {
    CreateProductionReport(ctx context.Context, tenantID, productionDate string, medallionsProduced, breadsPurchased int, notes string, createdBy *string) (*models.ProductionReport, error)
    GetProductionReport(ctx context.Context, tenantID, id string) (*models.ProductionReport, error)
    ListProductionReports(ctx context.Context, tenantID string, startDate, endDate *string) ([]models.ProductionReport, error)
    UpdateProductionReport(ctx context.Context, tenantID, id string, productionDate *string, medallionsProduced, breadsPurchased *int, notes *string) (*models.ProductionReport, error)
    DeleteProductionReport(ctx context.Context, tenantID, id string) error
}

type service struct {
    repo       Repository
    tenantRepo tenants.Repository
}

func NewService(repo Repository, tenantRepo tenants.Repository) Service {
    return &service{repo: repo, tenantRepo: tenantRepo}
}

func (s *service) CreateProductionReport(ctx context.Context, tenantID, productionDate string, medallionsProduced, breadsPurchased int, notes string, createdBy *string) (*models.ProductionReport, error) {
    cleanDate := strings.TrimSpace(productionDate)
    if cleanDate == "" {
        return nil, ErrInvalidProductionData
    }
    parsedDate, err := time.Parse("2006-01-02", cleanDate)
    if err != nil {
        return nil, ErrInvalidProductionData
    }
    if medallionsProduced < 0 || breadsPurchased < 0 {
        return nil, ErrInvalidProductionData
    }

    if err := s.validateTenant(ctx, tenantID); err != nil {
        return nil, err
    }

    existing, err := s.repo.GetByDate(ctx, tenantID, parsedDate)
    if err != nil {
        return nil, err
    }
    if existing != nil {
        return nil, ErrDuplicateProductionDate
    }

    report := &models.ProductionReport{
        TenantID:           tenantID,
        ProductionDate:     parsedDate,
        MedallionsProduced: medallionsProduced,
        BreadsPurchased:    breadsPurchased,
        Notes:              strings.TrimSpace(notes),
        CreatedBy:          createdBy,
    }

    if err := s.repo.Create(ctx, report); err != nil {
        return nil, err
    }

    return report, nil
}

func (s *service) GetProductionReport(ctx context.Context, tenantID, id string) (*models.ProductionReport, error) {
    report, err := s.repo.GetByID(ctx, tenantID, id)
    if err != nil {
        return nil, err
    }
    if report == nil {
        return nil, ErrProductionNotFound
    }
    return report, nil
}

func (s *service) ListProductionReports(ctx context.Context, tenantID string, startDate, endDate *string) ([]models.ProductionReport, error) {
    if err := s.validateTenant(ctx, tenantID); err != nil {
        return nil, err
    }

    if startDate != nil {
        cleanStart := strings.TrimSpace(*startDate)
        if _, err := time.Parse("2006-01-02", cleanStart); err != nil {
            return nil, ErrInvalidProductionData
        }
        startDate = &cleanStart
    }
    if endDate != nil {
        cleanEnd := strings.TrimSpace(*endDate)
        if _, err := time.Parse("2006-01-02", cleanEnd); err != nil {
            return nil, ErrInvalidProductionData
        }
        endDate = &cleanEnd
    }

    return s.repo.FetchAll(ctx, tenantID, startDate, endDate)
}

func (s *service) UpdateProductionReport(ctx context.Context, tenantID, id string, productionDate *string, medallionsProduced, breadsPurchased *int, notes *string) (*models.ProductionReport, error) {
    report, err := s.repo.GetByID(ctx, tenantID, id)
    if err != nil {
        return nil, err
    }
    if report == nil {
        return nil, ErrProductionNotFound
    }

    if productionDate != nil {
        cleanDate := strings.TrimSpace(*productionDate)
        if cleanDate == "" {
            return nil, ErrInvalidProductionData
        }
        parsedDate, err := time.Parse("2006-01-02", cleanDate)
        if err != nil {
            return nil, ErrInvalidProductionData
        }
        if !parsedDate.Equal(report.ProductionDate) {
            existing, err := s.repo.GetByDate(ctx, tenantID, parsedDate)
            if err != nil {
                return nil, err
            }
            if existing != nil {
                return nil, ErrDuplicateProductionDate
            }
            report.ProductionDate = parsedDate
        }
    }

    if medallionsProduced != nil {
        if *medallionsProduced < 0 {
            return nil, ErrInvalidProductionData
        }
        report.MedallionsProduced = *medallionsProduced
    }
    if breadsPurchased != nil {
        if *breadsPurchased < 0 {
            return nil, ErrInvalidProductionData
        }
        report.BreadsPurchased = *breadsPurchased
    }
    if notes != nil {
        report.Notes = strings.TrimSpace(*notes)
    }

    if err := s.repo.Update(ctx, report); err != nil {
        return nil, err
    }

    return report, nil
}

func (s *service) DeleteProductionReport(ctx context.Context, tenantID, id string) error {
    report, err := s.repo.GetByID(ctx, tenantID, id)
    if err != nil {
        return err
    }
    if report == nil {
        return ErrProductionNotFound
    }
    return s.repo.Delete(ctx, tenantID, id)
}

func (s *service) validateTenant(ctx context.Context, tenantID string) error {
    tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
    if err != nil {
        return err
    }
    if tenant == nil || !tenant.Active {
        return ErrTenantNotFoundForProduction
    }
    return nil
}
