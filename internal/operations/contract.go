package operations

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
	"strings"
	"time"
)

const requiredMigration = "202611120002_order_stream"

// VerifyDatabase refuses to start the new API against an old or different project.
func VerifyDatabase(database *gorm.DB, dsn string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("configuración de base inválida")
	}
	expected := ""
	if strings.HasSuffix(cfg.Host, ".pooler.supabase.com") && strings.HasPrefix(cfg.User, "postgres.") {
		expected = strings.TrimPrefix(cfg.User, "postgres.")
	}
	if strings.HasPrefix(cfg.Host, "db.") && strings.HasSuffix(cfg.Host, ".supabase.co") {
		expected = strings.TrimSuffix(strings.TrimPrefix(cfg.Host, "db."), ".supabase.co")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var contract struct {
		ProjectRef  string
		Application string
		Ready       bool
	}
	result := NewRepository(database).DB.WithContext(ctx).Raw(`SELECT c.project_ref,c.application,EXISTS(SELECT 1 FROM legui_private.schema_migrations WHERE version=?) ready FROM legui_private.database_contract c WHERE c.singleton`, requiredMigration).Scan(&contract)
	if result.Error != nil || result.RowsAffected != 1 || !contract.Ready || contract.Application != "leguiburger" {
		return fmt.Errorf("la base no tiene el contrato y las migraciones requeridos de LeguiBurger; revisá database/README.md")
	}
	if expected != "" && contract.ProjectRef != expected {
		return fmt.Errorf("la identidad de la base no coincide con el proyecto configurado")
	}
	return nil
}
