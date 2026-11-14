package auth

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"time"
)

// Reauthenticate uses the live identity inside the caller's locked transaction.
func Reauthenticate(ctx context.Context, tx *gorm.DB, p SessionPrincipal, password string) error {
	if len(password) > 72 {
		return ErrInvalidCredentials
	}
	i, err := loadIdentity(ctx, tx, p.ID)
	if err != nil {
		return err
	}
	if err = lockIdentity(tx, i); err != nil {
		return err
	}
	i, err = loadIdentity(ctx, tx, p.ID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(i.PasswordHash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	return nil
}
func QueueMail(tx *gorm.DB, recipient, subject, body string) error {
	return enqueueMail(tx, recipient, subject, body, time.Now().Add(7*24*time.Hour))
}

// The existing persistent limiter commits independently of a failed mutation.
func LimitSensitiveAction(database *gorm.DB, r *http.Request, scope, key string) error {
	return (&SecurityHandler{database: database}).rate(r, scope, key, 10)
}
