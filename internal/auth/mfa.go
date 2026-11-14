package auth

import (
	"errors"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (h *SecurityHandler) challenge(r *http.Request, tx *gorm.DB) (authSession, identity, error) {
	value := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	c, err := ValidateToken(value)
	if err != nil || c.Purpose != "mfa" || c.SessionID == "" {
		return authSession{}, identity{}, unauthorizedAction()
	}
	s, err := loadSession(r.Context(), tx, c.SessionID, true)
	if err != nil {
		return s, identity{}, err
	}
	i, err := loadIdentity(r.Context(), tx, s.PrincipalID)
	if err != nil {
		return s, i, err
	}
	if s.MFAVerified || !i.principal().Platform() || i.ID != c.UserID || c.Subject != i.ID || i.Kind != s.PrincipalKind || i.Role != s.Role || i.BrandID != s.BrandID || digest(i.PasswordHash) != s.CredentialDigest {
		return s, i, unauthorizedAction()
	}
	return s, i, nil
}
func (h *SecurityHandler) enrollMFA(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "mfa-enroll-ip", 20); err != nil {
		return err
	}
	var output map[string]any
	err := h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		s, i, err := h.challenge(r, tx)
		if err != nil {
			return err
		}
		secret := newTOTPSecret()
		encrypted, err := sealSecret(secret, "totp/"+i.ID)
		if err != nil {
			return err
		}
		if err = tx.Exec(`INSERT INTO legui_private.auth_factors(principal_kind,principal_id,secret_cipher) VALUES('user',?,?) ON CONFLICT DO NOTHING`, i.ID, encrypted).Error; err != nil {
			return err
		}
		var factor struct {
			SecretCipher string
			VerifiedAt   *time.Time
		}
		if err = tx.Raw(`SELECT secret_cipher,verified_at FROM legui_private.auth_factors WHERE principal_kind='user' AND principal_id=? FOR UPDATE`, i.ID).Scan(&factor).Error; err != nil {
			return err
		}
		if factor.VerifiedAt != nil {
			return forbiddenAction()
		}
		secret, err = openSecret(factor.SecretCipher, "totp/"+i.ID)
		if err != nil {
			return err
		}
		values := url.Values{"secret": {secret}, "issuer": {"LeguiBurger"}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
		output = map[string]any{"secret": secret, "uri": "otpauth://totp/" + url.PathEscape("LeguiBurger:"+i.Email) + "?" + values.Encode()}
		return Audit(r.Context(), tx, i.principal(), "auth.mfa_enrollment_started", s.ID, nil)
	})
	if err != nil {
		return err
	}
	respondWithJSON(w, 200, output)
	return nil
}
func (h *SecurityHandler) verifyMFA(w http.ResponseWriter, r *http.Request) error {
	if err := h.ipRate(r, "mfa-ip", 30); err != nil {
		return err
	}
	var input struct {
		Code         string `json:"code"`
		RecoveryCode string `json:"recovery_code"`
	}
	if err := readSecurityJSON(r, &input); err != nil {
		return err
	}
	// Account-wide limits also cover multiple fresh password challenges.
	c, err := ValidateToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if err != nil || c.Purpose != "mfa" {
		return unauthorizedAction()
	}
	if err = h.rate(r, "mfa-account", c.UserID, 10); err != nil {
		return err
	}
	var output map[string]any
	var refresh string
	err = h.database.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		s, i, e := h.challenge(r, tx)
		if e != nil {
			return e
		}
		var factor struct {
			SecretCipher string
			VerifiedAt   *time.Time
			LastStep     int64
		}
		result := tx.Raw(`SELECT secret_cipher,verified_at,last_step FROM legui_private.auth_factors WHERE principal_kind='user' AND principal_id=? FOR UPDATE`, i.ID).Scan(&factor)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return unauthorizedAction()
		}
		enrollment := factor.VerifiedAt == nil
		if input.RecoveryCode != "" {
			if enrollment {
				return unauthorizedAction()
			}
			result = tx.Exec(`UPDATE legui_private.auth_recovery_codes SET used_at=clock_timestamp() WHERE principal_id=? AND digest=? AND used_at IS NULL`, i.ID, digest(i.ID+"/"+strings.TrimSpace(input.RecoveryCode)))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return &securityError{401, "INVALID_MFA", "Código inválido o ya utilizado"}
			}
		} else {
			secret, e := openSecret(factor.SecretCipher, "totp/"+i.ID)
			if e != nil {
				return e
			}
			step, ok := verifyTOTP(secret, input.Code, factor.LastStep, time.Now())
			if !ok {
				return &securityError{401, "INVALID_MFA", "Código inválido o ya utilizado"}
			}
			if e = tx.Exec(`UPDATE legui_private.auth_factors SET verified_at=coalesce(verified_at,clock_timestamp()),last_step=? WHERE principal_kind='user' AND principal_id=?`, step, i.ID).Error; e != nil {
				return e
			}
		}
		if e = tx.Exec(`UPDATE legui_private.auth_sessions SET mfa_verified=true,expires_at=clock_timestamp()+interval '7 days',last_seen_at=clock_timestamp() WHERE id=?`, s.ID).Error; e != nil {
			return e
		}
		output, refresh, e = issueSession(tx, i, s.ID, true)
		if e != nil {
			return e
		}
		if enrollment {
			codes := make([]string, 10)
			for index := range codes {
				codes[index] = randomToken()[:24]
				if e = tx.Exec(`INSERT INTO legui_private.auth_recovery_codes(principal_id,digest) VALUES(?,?)`, i.ID, digest(i.ID+"/"+codes[index])).Error; e != nil {
					return e
				}
			}
			output["recovery_codes"] = codes
		}
		event := "auth.mfa_verified"
		if input.RecoveryCode != "" {
			event = "auth.mfa_recovery_used"
		}
		return Audit(r.Context(), tx, i.principal(), event, s.ID, nil)
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return unauthorizedAction()
		}
		return err
	}
	setRefreshCookie(w, refresh)
	respondWithJSON(w, 200, output)
	return nil
}
