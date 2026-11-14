package businesscare

import (
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
)

// Account deactivation is distinct from business archival. Immutable financial
// records and author identities remain available for authorized accounting.
func (h *Handler) closeAccount(r *http.Request, p auth.SessionPrincipal) (any, error) {
	if e := auth.LimitSensitiveAction(h.DB, r, "account-closure", p.ID); e != nil {
		return nil, &problem{429, "Esperá antes de volver a intentar la baja"}
	}
	var in struct {
		Password     string `json:"password"`
		Confirmation string `json:"confirmation"`
	}
	if e := read(r, &in); e != nil {
		return nil, e
	}
	if in.Confirmation != "CERRAR CUENTA" {
		return nil, invalid("Escribí CERRAR CUENTA para confirmar")
	}
	return h.mutate(r, p, in, func(tx *gorm.DB) (any, error) {
		if e := auth.Reauthenticate(r.Context(), tx, p, in.Password); e != nil {
			return nil, e
		}
		if p.Platform() {
			if e := tx.Exec(`SELECT id FROM public.users WHERE is_active AND role IN ('owner','super_admin') ORDER BY id FOR UPDATE`).Error; e != nil {
				return nil, e
			}
			var remaining int64
			if e := tx.Raw(`SELECT count(*) FROM public.users WHERE is_active AND role IN ('owner','super_admin') AND id<>?`, p.ID).Scan(&remaining).Error; e != nil {
				return nil, e
			}
			if remaining == 0 {
				return nil, invalid("La plataforma necesita al menos otro administrador activo antes de dar de baja tu cuenta")
			}
		}
		if p.Kind == "user" {
			var active int64
			if e := tx.Raw(`SELECT count(*) FROM public.brands WHERE primary_owner_id=? AND closed_at IS NULL`, p.ID).Scan(&active).Error; e != nil {
				return nil, e
			}
			if active > 0 {
				return nil, invalid("Transferí la responsabilidad o cerrá el restaurante antes de dar de baja tu cuenta")
			}
			if e := tx.Exec(`UPDATE public.brands SET primary_owner_id=NULL WHERE primary_owner_id=? AND closed_at IS NOT NULL`, p.ID).Error; e != nil {
				return nil, e
			}
		}
		table := "public.users"
		if p.Kind == "employee" {
			table = "public.employees"
		}
		if e := tx.Exec(`UPDATE `+table+` SET is_active=false WHERE id=?`, p.ID).Error; e != nil {
			return nil, e
		}
		return map[string]bool{"deactivated": true, "history_preserved": true}, auth.Audit(r.Context(), tx, p, "account.closed", p.ID, nil)
	})
}
