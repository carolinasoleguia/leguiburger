package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

type Claims struct {
	SessionID string  `json:"session_id,omitempty"`
	Purpose   string  `json:"purpose,omitempty"`
	MFA       bool    `json:"mfa,omitempty"`
	UserID    string  `json:"user_id"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	TenantID  string  `json:"tenant_id,omitempty"`
	BrandID   *string `json:"brand_id,omitempty"`
	jwt.RegisteredClaims
}

func ConfigureJWTSecret(secret string) error {
	cleanSecret := strings.TrimSpace(secret)
	if cleanSecret == "" {
		return ErrJWTSecretRequired
	}

	jwtSecret = []byte(cleanSecret)
	return nil
}

// GenerateToken is a short-lived credential result. It cannot authorize API calls
// until the login handler attaches a persisted session through SessionToken.
func GenerateToken(userID, email, role string, tenantID *string, brandID *string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", ErrJWTSecretRequired
	}

	tID := ""
	if tenantID != nil {
		tID = *tenantID
	}

	claims := Claims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		TenantID: tID,
		BrandID:  brandID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "leguiburger",
			Audience:  jwt.ClaimStrings{"leguiburger-api"},
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken comprueba la validez de una cadena JWT recibida en la peticion.
func ValidateToken(tokenString string) (*Claims, error) {
	if len(jwtSecret) == 0 {
		return nil, ErrJWTSecretRequired
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metodo de firma no valido")
		}
		return jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuer("leguiburger"), jwt.WithAudience("leguiburger-api"), jwt.WithIssuedAt())

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("token invalido o expirado")
	}

	return claims, nil
}

func SessionToken(p SessionPrincipal, sessionID string, mfa bool, purpose string) (string, error) {
	lifetime := 15 * time.Minute
	if purpose == "mfa" {
		lifetime = 5 * time.Minute
	}
	var brand *string
	if p.BrandID != "" {
		brand = &p.BrandID
	}
	c := Claims{UserID: p.ID, Role: p.Role, BrandID: brand, TenantID: p.TenantID, SessionID: sessionID, MFA: mfa, Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{Subject: p.ID, Issuer: "leguiburger", Audience: jwt.ClaimStrings{"leguiburger-api"}, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(lifetime))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(jwtSecret)
}
