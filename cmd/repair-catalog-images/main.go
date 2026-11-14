// Command repair-catalog-images recovers existing catalog photos without
// exposing a legacy filesystem directory or copying products between brands.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"leguiburger/internal/db"
	"leguiburger/internal/operations"
	"leguiburger/internal/products"
)

type imageRow struct{ ID, BrandID, Name, ImageURL string }

func run() error {
	project := flag.String("expect-project", "", "Required Supabase project reference")
	tenant := flag.String("tenant", "", "Required local whose brand images will be repaired")
	root := flag.String("uploads-root", "uploads/products", "Directory containing existing legacy images")
	apply := flag.Bool("apply", false, "Upload, verify and update image references")
	publish := flag.Bool("publish", false, "Publish this local after all images are verified")
	ordering := flag.Bool("enable-orders", false, "Enable ordering on this published local")
	flag.Parse()
	if _, e := uuid.Parse(*tenant); e != nil || *project == "" {
		return errors.New("specify --tenant UUID and --expect-project")
	}
	if *ordering && !*publish {
		return errors.New("--enable-orders requires --publish")
	}
	_ = godotenv.Load()
	if strings.TrimRight(os.Getenv("SUPABASE_URL"), "/") != "https://"+*project+".supabase.co" {
		return errors.New("storage project does not match --expect-project")
	}
	database, e := gorm.Open(postgres.New(postgres.Config{DSN: os.Getenv("DATABASE_URL"), PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		return errors.New("database unavailable")
	}
	pool, e := database.DB()
	if e != nil {
		return e
	}
	defer pool.Close()
	if e = operations.VerifyDatabase(database, os.Getenv("DATABASE_URL")); e != nil {
		return e
	}
	var actual string
	if e = database.Raw(`SELECT project_ref FROM legui_private.database_contract WHERE singleton`).Scan(&actual).Error; e != nil || actual != *project {
		return errors.New("database project mismatch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	database = database.WithContext(ctx)
	var brand string
	q := database.Raw(`SELECT t.brand_id::text FROM public.tenants t JOIN public.brands b ON b.id=t.brand_id WHERE t.id=? AND t.active AND b.suspended_at IS NULL AND b.closed_at IS NULL`, *tenant).Scan(&brand)
	if q.Error != nil || q.RowsAffected != 1 {
		return errors.New("active restaurant not found")
	}
	var rows []imageRow
	if e = database.Raw(`SELECT id::text,brand_id::text,name,image_url FROM public.products WHERE brand_id=? AND image_url LIKE '/uploads/products/%' ORDER BY id`, brand).Scan(&rows).Error; e != nil {
		return e
	}
	fmt.Printf("Legacy images: %d; apply=%t; publish=%t; ordering=%t\n", len(rows), *apply, *publish, *ordering)
	if !*apply {
		return nil
	}
	db.DB = database
	store := products.NewProductImageStoreFromEnv()
	for _, row := range rows {
		link, e := products.RecoverLegacyImage(ctx, store, row.BrandID, *root, row.ImageURL)
		if e != nil {
			return fmt.Errorf("product %s: %w", row.ID, e)
		}
		if e = verifyImage(ctx, link, *project); e != nil {
			return fmt.Errorf("product %s: %w", row.ID, e)
		}
		e = database.Transaction(func(tx *gorm.DB) error {
			changed := tx.Exec(`UPDATE public.products SET image_url=? WHERE id=? AND brand_id=? AND image_url=?`, link, row.ID, brand, row.ImageURL)
			if changed.Error != nil {
				return changed.Error
			}
			if changed.RowsAffected != 1 {
				return errors.New("image changed concurrently; original retained")
			}
			return tx.Exec(`INSERT INTO public.activity_events(brand_id,event_name,source,environment,facts) VALUES(?,'catalog.image_recovered','server','production',?::jsonb)`, brand, jsonString(map[string]string{"product_id": row.ID, "previous_url": row.ImageURL, "image_url": link})).Error
		})
		if e != nil {
			return e
		}
		fmt.Printf("Recovered and verified: %s\n", row.Name)
	}
	if *publish {
		var visible []imageRow
		e = database.Raw(`SELECT p.id::text,p.image_url FROM public.products p JOIN public.tenant_products tp ON tp.product_id=p.id WHERE tp.tenant_id=? AND p.brand_id=? AND p.is_active AND tp.is_active`, *tenant, brand).Scan(&visible).Error
		if e != nil {
			return e
		}
		if len(visible) == 0 {
			return errors.New("local has no active assigned products; not published")
		}
		for _, row := range visible {
			if e = verifyImage(ctx, row.ImageURL, *project); e != nil {
				return fmt.Errorf("visible product %s: %w", row.ID, e)
			}
		}
		e = database.Transaction(func(tx *gorm.DB) error {
			r := tx.Exec(`UPDATE public.tenants t SET menu_published=true,ordering_enabled=CASE WHEN ? THEN true ELSE ordering_enabled END FROM public.brands b WHERE t.id=? AND t.brand_id=? AND b.id=t.brand_id AND t.active AND b.suspended_at IS NULL AND b.closed_at IS NULL`, *ordering, *tenant, brand)
			if r.Error != nil {
				return r.Error
			}
			if r.RowsAffected != 1 {
				return errors.New("local changed before publication")
			}
			return tx.Exec(`INSERT INTO public.activity_events(brand_id,tenant_id,event_name,source,environment,facts) VALUES(?,?,'menu.recovered_publication','server','production',?::jsonb)`, brand, *tenant, jsonString(map[string]bool{"published": true, "enable_orders": *ordering})).Error
		})
		if e != nil {
			return e
		}
		fmt.Println("Published: /catalogo/" + *tenant)
	}
	return nil
}
func jsonString(value any) string { raw, _ := json.Marshal(value); return string(raw) }
func verifyImage(ctx context.Context, raw, project string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host != project+".supabase.co" || !strings.HasPrefix(u.Path, "/storage/v1/object/public/") {
		return errors.New("image is not in the expected public storage")
	}
	request, e := http.NewRequestWithContext(ctx, http.MethodHead, raw, nil)
	if e != nil {
		return e
	}
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, e := client.Do(request)
	if e != nil {
		return errors.New("public image unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || !strings.HasPrefix(response.Header.Get("Content-Type"), "image/") {
		return errors.New("public URL does not serve an image")
	}
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
