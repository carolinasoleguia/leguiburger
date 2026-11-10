package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"testing"
	"time"
)

func TestTokenRequiresExpectedAlgorithmAndExpiration(t *testing.T) {
	previous := jwtSecret
	defer func() { jwtSecret = previous }()
	if err := ConfigureJWTSecret("jwt-validation-unit-fixture"); err != nil {
		t.Fatal(err)
	}
	cases := []*jwt.Token{
		jwt.NewWithClaims(jwt.SigningMethodHS384, Claims{UserID: "fixture", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}),
		jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "fixture"}),
		jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: "fixture", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}),
	}
	for _, token := range cases {
		signed, err := token.SignedString(jwtSecret)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = ValidateToken(signed); err == nil {
			t.Fatal("accepted wrong algorithm, missing expiration or expired token")
		}
	}
}
