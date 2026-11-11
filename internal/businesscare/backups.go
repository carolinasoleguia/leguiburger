package businesscare

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"leguiburger/internal/auth"
	"net/http"
	"strings"
	"time"
)

// This is deliberately an allowlist. New tables do not silently export secrets.
// Account credentials, sessions, mail bodies, OAuth and provider payloads stay out.
var snapshotTables = []string{"brands", "tenants", "products", "tenant_products", "supplies", "extras", "recipes", "shipping_methods", "customers", "order_statuses_lookup", "payment_statuses_lookup", "orders", "order_details", "order_detail_extras", "sales", "production_reports", "order_status_events", "order_payments", "operating_expenses", "inventory_movements", "activity_events"}

type snapshot struct {
	Version   int                        `json:"version"`
	BrandID   string                     `json:"brand_id"`
	CreatedAt time.Time                  `json:"created_at"`
	Tables    map[string]json.RawMessage `json:"tables"`
}

func columns(tx *gorm.DB, table string) ([]string, error) {
	var cols []string
	e := tx.Raw(`SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=? AND is_generated='NEVER' ORDER BY ordinal_position`, table).Scan(&cols).Error
	return cols, e
}
func scopeFor(table string, cols []string) (string, bool) {
	if table == "order_statuses_lookup" || table == "payment_statuses_lookup" {
		return "?::uuid IS NOT NULL", true
	}
	has := func(s string) bool {
		for _, c := range cols {
			if c == s {
				return true
			}
		}
		return false
	}
	if table == "brands" {
		return "id=?::uuid", true
	}
	if has("brand_id") {
		return "brand_id=?::uuid", true
	}
	if has("tenant_id") {
		return "tenant_id IN(SELECT id FROM public.tenants WHERE brand_id=?::uuid)", true
	}
	if has("order_id") {
		return "order_id IN(SELECT id FROM public.orders WHERE tenant_id IN(SELECT id FROM public.tenants WHERE brand_id=?::uuid))", true
	}
	if has("order_detail_id") {
		return "order_detail_id IN(SELECT id FROM public.order_details WHERE order_id IN(SELECT id FROM public.orders WHERE tenant_id IN(SELECT id FROM public.tenants WHERE brand_id=?::uuid)))", true
	}
	if table == "recipes" && has("product_id") {
		return "product_id IN(SELECT id FROM public.products WHERE brand_id=?::uuid)", true
	}
	return "", false
}
func capture(tx *gorm.DB, id string) (snapshot, error) {
	s := snapshot{Version: 1, BrandID: id, CreatedAt: time.Now().UTC(), Tables: map[string]json.RawMessage{}}
	size := 0
	for _, table := range snapshotTables {
		cols, e := columns(tx, table)
		if e != nil {
			return s, e
		}
		if len(cols) == 0 {
			continue
		}
		where, ok := scopeFor(table, cols)
		if !ok {
			return s, fmt.Errorf("snapshot scope missing for %s", table)
		}
		var raw string
		// The size guard runs on PostgreSQL before transferring a potentially large JSON.
		e = tx.Raw(`WITH rows AS(SELECT to_jsonb(t) row FROM public.`+table+` t WHERE `+where+`), payload AS(SELECT coalesce(jsonb_agg(row),'[]') data FROM rows) SELECT CASE WHEN octet_length(data::text)<=? THEN data::text ELSE NULL END FROM payload`, id, maxBackup-size).Scan(&raw).Error
		if e != nil {
			return s, e
		}
		if raw == "" {
			return s, invalid("El respaldo supera los 20 MB; solicitá una exportación asistida")
		}
		size += len(raw)
		s.Tables[table] = json.RawMessage(raw)
	}
	return s, nil
}

// Recreate rows in isolated temporary tables with the original column types,
// generated expressions and CHECK constraints. No production row is overwritten.
func verifySnapshot(tx *gorm.DB, s snapshot) (map[string]any, error) {
	if s.Version != 1 {
		return nil, errors.New("unsupported backup format")
	}
	counts := map[string]int{}
	for _, table := range snapshotTables {
		raw, ok := s.Tables[table]
		if !ok {
			continue
		}
		cols, e := columns(tx, table)
		if e != nil {
			return nil, e
		}
		if len(cols) == 0 {
			return nil, errors.New("backup table missing")
		}
		names := make([]string, len(cols))
		for i, c := range cols {
			names[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
		}
		temp := "care_restore_" + table
		if e = tx.Exec(`CREATE TEMP TABLE ` + temp + ` (LIKE public.` + table + ` INCLUDING CONSTRAINTS INCLUDING GENERATED) ON COMMIT DROP`).Error; e != nil {
			return nil, e
		}
		fields := strings.Join(names, ",")
		if e = tx.Exec(`INSERT INTO `+temp+` (`+fields+`) SELECT `+fields+` FROM jsonb_populate_recordset(NULL::public.`+table+`,?::jsonb)`, string(raw)).Error; e != nil {
			return nil, e
		}
		var n int
		if e = tx.Raw(`SELECT count(*) FROM ` + temp).Scan(&n).Error; e != nil {
			return nil, e
		}
		counts[table] = n
		// Recreate PK/UNIQUE indexes to reject duplicate identities in a restored file.
		var defs []string
		if e = tx.Raw(`SELECT pg_get_indexdef(i.indexrelid) FROM pg_index i WHERE i.indrelid=('public.'||?)::regclass AND i.indisunique`, table).Scan(&defs).Error; e != nil {
			return nil, e
		}
		for index, def := range defs {
			pos := strings.Index(def, " USING ")
			if pos < 0 {
				continue
			}
			if e = tx.Exec(fmt.Sprintf("CREATE UNIQUE INDEX %s_%d ON %s%s", temp, index, temp, def[pos:])).Error; e != nil {
				return nil, e
			}
		}
	}
	var keys []struct{ Child, Parent, NonNull, Equality string }
	if e := tx.Raw(`SELECT ch.relname child,pa.relname parent,string_agg(format('c.%I IS NOT NULL',ca.attname),' AND ' ORDER BY k.ord) non_null,string_agg(format('p.%I=c.%I',pca.attname,ca.attname),' AND ' ORDER BY k.ord) equality
 FROM pg_constraint f JOIN pg_class ch ON ch.oid=f.conrelid JOIN pg_namespace ns ON ns.oid=ch.relnamespace AND ns.nspname='public'
 JOIN pg_class pa ON pa.oid=f.confrelid JOIN pg_namespace pn ON pn.oid=pa.relnamespace AND pn.nspname='public'
 CROSS JOIN LATERAL unnest(f.conkey,f.confkey) WITH ORDINALITY k(ca,pa,ord)
 JOIN pg_attribute ca ON ca.attrelid=ch.oid AND ca.attnum=k.ca JOIN pg_attribute pca ON pca.attrelid=pa.oid AND pca.attnum=k.pa
 WHERE f.contype='f' GROUP BY f.oid,ch.relname,pa.relname`).Scan(&keys).Error; e != nil {
		return nil, e
	}
	checked, excluded := 0, 0
	for _, fk := range keys {
		if _, ok := s.Tables[fk.Child]; !ok {
			continue
		}
		if _, ok := s.Tables[fk.Parent]; !ok {
			excluded++
			continue
		}
		var missing bool
		if e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM care_restore_` + fk.Child + ` c WHERE ` + fk.NonNull + ` AND NOT EXISTS(SELECT 1 FROM care_restore_` + fk.Parent + ` p WHERE ` + fk.Equality + `))`).Scan(&missing).Error; e != nil {
			return nil, e
		}
		if missing {
			return nil, fmt.Errorf("backup relationship missing: %s to %s", fk.Child, fk.Parent)
		}
		checked++
	}
	return map[string]any{"verified": true, "checks": "column_types, not_null, check_constraints, generated_columns, unique_keys, captured_foreign_keys", "foreign_keys_verified": checked, "external_identity_references": excluded, "rows": counts, "scope": "restaurant operational snapshot; authentication and provider credentials excluded"}, nil
}
func (h *Handler) backups(r *http.Request, p auth.SessionPrincipal) (any, error) {
	var raw string
	e := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		id := selected(r, p)
		if e := brand(tx, p, id, false); e != nil {
			return e
		}
		return tx.Raw(`SELECT coalesce(jsonb_agg(to_jsonb(b) ORDER BY created_at DESC),'[]')::text FROM (SELECT id,state,sha256,report,created_at,verified_at FROM legui_private.business_backups WHERE brand_id=? ORDER BY created_at DESC LIMIT 30)b`, id).Scan(&raw).Error
	})
	return json.RawMessage(raw), e
}
func (h *Handler) createBackup(r *http.Request, p auth.SessionPrincipal) (any, error) {
	var in struct{}
	if e := read(r, &in); e != nil {
		return nil, e
	}
	brandID := selected(r, p)
	return h.mutate(r, p, map[string]string{"brand_id": brandID}, func(tx *gorm.DB) (any, error) {
		if e := brand(tx, p, brandID, false); e != nil {
			return nil, e
		}
		var count int64
		if e := tx.Raw(`SELECT count(*) FROM legui_private.business_backups WHERE brand_id=? AND created_at>now()-interval '1 hour'`, brandID).Scan(&count).Error; e != nil {
			return nil, e
		}
		if count >= 3 {
			return nil, &problem{429, "Podés crear hasta tres respaldos por hora"}
		}
		s, e := capture(tx, brandID)
		if e != nil {
			return nil, e
		}
		report, e := verifySnapshot(tx, s)
		if e != nil {
			return nil, e
		}
		raw, e := json.Marshal(s)
		if e != nil {
			return nil, e
		}
		if len(raw) > maxBackup {
			return nil, invalid("El respaldo supera los 20 MB")
		}
		id := uuid.NewString()
		path := brandID + "/" + id + ".bin"
		encrypted, e := seal(raw, path)
		if e != nil {
			return nil, e
		}
		sum := sha256.Sum256(encrypted)
		encodedReport, _ := json.Marshal(report)
		if e = tx.Exec(`INSERT INTO legui_private.business_backups(id,brand_id,actor_id,payload_cipher,object_path,sha256,report,verified_at) VALUES(?,?,?,?,?,?,?::jsonb,now())`, id, brandID, p.ID, encrypted, path, hex.EncodeToString(sum[:]), string(encodedReport)).Error; e != nil {
			return nil, e
		}
		return map[string]any{"id": id, "state": "pending", "report": report}, auth.Audit(r.Context(), tx, p, "backup.created", id, nil)
	})
}
func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFromContext(r.Context())
	id := r.PathValue("id")
	if _, e := uuid.Parse(id); e != nil {
		auth.RespondWithError(w, 404, "NOT_FOUND", "Respaldo no encontrado")
		return
	}
	var item struct{ BrandID, ObjectPath, Sha256, State string }
	e := h.DB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		q := tx.Raw(`SELECT brand_id::text,object_path,sha256,state FROM legui_private.business_backups WHERE id=?`, id).Scan(&item)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return denied()
		}
		if e := brand(tx, p, item.BrandID, false); e != nil {
			return e
		}
		return auth.Audit(r.Context(), tx, p, "backup.downloaded", id, nil)
	})
	if e != nil || item.State != "ready" {
		auth.RespondWithError(w, 403, "BACKUP_UNAVAILABLE", "El respaldo no está disponible")
		return
	}
	encrypted, _, e := storage(r.Context(), "GET", "business-backups", item.ObjectPath, nil)
	sum := sha256.Sum256(encrypted)
	if e != nil || hex.EncodeToString(sum[:]) != item.Sha256 {
		auth.RespondWithError(w, 503, "BACKUP_INTEGRITY", "No se pudo verificar la integridad del respaldo")
		return
	}
	body, e := open(encrypted, item.ObjectPath)
	if e != nil {
		auth.RespondWithError(w, 503, "BACKUP_KEY", "No se pudo abrir el respaldo")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="leguiburger-`+id+`.json"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(body)
}
func uploadOne(ctx context.Context, db *gorm.DB) error {
	var job struct {
		ID, ObjectPath, Sha256, Claim string
		PayloadCipher                 []byte
		Attempts                      int
	}
	q := db.WithContext(ctx).Raw(`WITH picked AS(SELECT id FROM legui_private.business_backups WHERE state='pending' AND attempts<8 AND next_attempt_at<=now() AND (lease_until IS NULL OR lease_until<now()) ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED) UPDATE legui_private.business_backups b SET claim=gen_random_uuid(),lease_until=now()+interval '2 minutes',attempts=attempts+1 FROM picked WHERE b.id=picked.id RETURNING b.id::text,b.object_path,b.sha256,b.claim::text,b.payload_cipher,b.attempts`).Scan(&job)
	if q.Error != nil || q.RowsAffected == 0 {
		return q.Error
	}
	// A previous upload may have succeeded despite a lost response. Read first;
	// immutable paths and exact hashes make retries safe without overwriting data.
	existing, status, e := storage(ctx, "GET", "business-backups", job.ObjectPath, nil)
	if status == 404 {
		_, _, e = storage(ctx, "POST", "business-backups", job.ObjectPath, job.PayloadCipher)
		if e == nil {
			existing, _, e = storage(ctx, "GET", "business-backups", job.ObjectPath, nil)
		}
	}
	sum := sha256.Sum256(existing)
	if e == nil && hex.EncodeToString(sum[:]) != job.Sha256 {
		e = errors.New("backup integrity mismatch")
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if e != nil {
		state := "pending"
		if job.Attempts >= 8 {
			state = "review"
		}
		return db.WithContext(cleanup).Exec(`UPDATE legui_private.business_backups SET state=?,claim=NULL,lease_until=NULL,next_attempt_at=now()+interval '5 minutes' WHERE id=? AND claim=?`, state, job.ID, job.Claim).Error
	}
	return db.WithContext(cleanup).Exec(`UPDATE legui_private.business_backups SET state='ready',payload_cipher=NULL,claim=NULL,lease_until=NULL WHERE id=? AND claim=?`, job.ID, job.Claim).Error
}
