package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"leguiburger/internal/auth"
	"leguiburger/internal/models"
	"leguiburger/internal/brands"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("usuario no encontrado")
	ErrDuplicateUserEmail = errors.New("ya existe un usuario con ese email")
	ErrInvalidUserData    = errors.New("email, password y brand_id son obligatorios")
	ErrInvalidUserRole    = errors.New("el rol del usuario no es válido")
	ErrBrandNotFound      = errors.New("la marca especificada no existe")
	ErrUnauthorizedAction  = errors.New("no tienes permisos para realizar esta acción")
)

type Service interface {
	CreateUser(ctx context.Context, firstName, lastName, brandID, email, password, role string) (*models.User, error)
	GetUser(ctx context.Context, id string) (*models.User, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	UpdateUser(ctx context.Context, id, firstName, lastName, email, password, role string, brandID *string, isActive *bool) (*models.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type service struct {
	repo       Repository
	brandRepo brands.Repository
}

func NewService(repo Repository, brandRepo brands.Repository) Service {
	return &service{repo: repo, brandRepo: brandRepo}
}

func (s *service) CreateUser(ctx context.Context, firstName, lastName, brandID, email, password, role string) (*models.User, error) {
	cleanFirstName := strings.TrimSpace(firstName)
	cleanLastName := strings.TrimSpace(lastName)
	cleanBrandID := strings.TrimSpace(brandID)
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanPassword := strings.TrimSpace(password)
	cleanRole := normalizeRole(role)

	if cleanFirstName == "" || cleanLastName == "" || cleanEmail == "" || cleanPassword == "" || cleanBrandID == "" {
		return nil, ErrInvalidUserData
	}
	if !isValidRole(cleanRole) {
		return nil, ErrInvalidUserRole
	}

	brand, err := s.brandRepo.GetByID(ctx, cleanBrandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFound
	}

	existing, err := s.repo.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateUserEmail
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(cleanPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error al hashear la contraseña: %w", err)
	}

	user := &models.User{
		FirstName:    cleanFirstName,
		LastName:     cleanLastName,
		Email:        cleanEmail,
		PasswordHash: string(hashedBytes),
		Role:         cleanRole,
		BrandID:      &cleanBrandID,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) GetUser(ctx context.Context, id string) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *service) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) UpdateUser(ctx context.Context, id, firstName, lastName, email, password, role string, brandID *string, isActive *bool) (*models.User, error) {
	claims, ok := auth.GetClaimsFromContext(ctx)
	if ok && strings.ToLower(strings.TrimSpace(claims.Role)) != auth.RoleOwner {
		return nil, ErrUnauthorizedAction
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if firstName != "" {
		user.FirstName = strings.TrimSpace(firstName)
	}
	if lastName != "" {
		user.LastName = strings.TrimSpace(lastName)
	}

	if email != "" {
		cleanEmail := strings.ToLower(strings.TrimSpace(email))
		if cleanEmail == "" {
			return nil, ErrInvalidUserData
		}
		existing, err := s.repo.GetByEmail(ctx, cleanEmail)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != user.ID {
			return nil, ErrDuplicateUserEmail
		}
		user.Email = cleanEmail
	}

	if password != "" {
		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("error al hashear la contraseña: %w", err)
		}
		user.PasswordHash = string(hashedBytes)
	}

	if role != "" {
		cleanRole := normalizeRole(role)
		if !isValidRole(cleanRole) {
			return nil, ErrInvalidUserRole
		}
		user.Role = cleanRole
	}

	if brandID != nil {
		if *brandID == "" {
			return nil, ErrInvalidUserData
		}
		brand, err := s.brandRepo.GetByID(ctx, *brandID)
		if err != nil {
			return nil, err
		}
		if brand == nil {
			return nil, ErrBrandNotFound
		}
		user.BrandID = brandID
	}

	if isActive != nil {
		user.IsActive = *isActive
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) DeleteUser(ctx context.Context, id string) error {
	claims, ok := auth.GetClaimsFromContext(ctx)
	if ok && strings.ToLower(strings.TrimSpace(claims.Role)) != auth.RoleOwner {
		return ErrUnauthorizedAction
	}
	return s.repo.Delete(ctx, id)
}

func normalizeRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	if r == "" {
		return "admin"
	}
	return r
}

func isValidRole(role string) bool {
	switch role {
	case "admin", "owner":
		return true
	default:
		return false
	}
}
