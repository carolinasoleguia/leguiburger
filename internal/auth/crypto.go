package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func jsonText(value any) string { encoded, _ := json.Marshal(value); return string(encoded) }
func secretAEAD() (cipher.AEAD, error) {
	// Domain-separated encryption key; rotating JWT_SECRET also invalidates sessions.
	key := os.Getenv("AUTH_ENCRYPTION_KEY")
	if key == "" {
		key = string(jwtSecret)
	}
	if key == "" {
		return nil, ErrJWTSecretRequired
	}
	sum := sha256.Sum256([]byte("leguiburger/auth-secrets/v1/" + key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func sealSecret(value, domain string) (string, error) {
	a, err := secretAEAD()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(a.Seal(nonce, nonce, []byte(value), []byte(domain))), nil
}
func openSecret(value, domain string) (string, error) {
	a, err := secretAEAD()
	if err != nil {
		return "", err
	}
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(b) < a.NonceSize() {
		return "", errors.New("invalid encrypted secret")
	}
	plain, err := a.Open(nil, b[:a.NonceSize()], b[a.NonceSize():], []byte(domain))
	return string(plain), err
}
func newTOTPSecret() string {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}
func totpAt(secret string, step int64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, uint64(step))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg)
	b := mac.Sum(nil)
	offset := b[len(b)-1] & 15
	n := (binary.BigEndian.Uint32(b[offset:offset+4]) & 0x7fffffff) % 1000000
	return fmt.Sprintf("%06d", n)
}
func verifyTOTP(secret, code string, last int64, now time.Time) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	step := now.Unix() / 30
	for _, offset := range []int64{0, -1, 1} {
		candidate := step + offset
		if candidate > last && subtle.ConstantTimeCompare([]byte(totpAt(secret, candidate)), []byte(code)) == 1 {
			return candidate, true
		}
	}
	return 0, false
}
func validPassword(value string) bool {
	return len(value) >= 12 && len([]byte(value)) <= 72 && strings.TrimSpace(value) == value
}
