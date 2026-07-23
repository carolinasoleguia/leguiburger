package auth

import (
	"context"
	"errors"

	"leguiburger/internal/models"
	"leguiburger/internal/tenants"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials    = errors.New("credenciales inválidas")
	ErrTenantNotFoundForAuth = errors.New("el comercio especificado no existe")
)

// Clave secreta para firmar tokens (debería venir de variable de entorno os.Getenv("JWT_SECRET"))
var jwtSecret = []byte("tu_super_clave_secreta_jwt_leguiburger")

type LoginResponse struct {
	Token    string           `json:"token"`
	Employee *models.Employee `json:"employee"`
}

type Service interface {
	Login(ctx context.Context, tenantID, email, password string) (*LoginResponse, error)
}

type service struct {
	repo       Repository
	tenantRepo tenants.Repository
}

func NewService(repo Repository, tenantRepo tenants.Repository) Service {
	return &service{repo: repo, tenantRepo: tenantRepo}
}

func (s *service) Login(ctx context.Context, tenantID, email, password string) (*LoginResponse, error) {
	// 1. Búsqueda global del empleado
	employee, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if employee == nil {
		return nil, ErrInvalidCredentials
	}

	// 2. Validar restricciones de tenant si no es el Owner global
	if employee.Role != "owner" && tenantID != "" {
		if employee.TenantID == nil || *employee.TenantID != tenantID {
			return nil, errors.New("no autorizado para este comercio")
		}
	}

	// 3. Validar contraseña
	err = bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 4. Generar token
	token, err := GenerateToken(
		employee.ID,
		employee.Email,
		employee.Role,
		employee.TenantID,
	)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:    token,
		Employee: employee,
	}, nil
}
