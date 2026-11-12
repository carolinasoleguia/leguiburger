// migrate applies reviewed SQL files with checksums, project pinning and an atomic rehearsal.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type snapshot struct {
	Schema  string   `json:"schema"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Count   int64    `json:"count"`
	Digest  string   `json:"digest"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Migration aborted:", err)
		os.Exit(1)
	}
}
func run() error {
	apply := flag.Bool("apply", false, "Commit changes; default is a full rollback rehearsal")
	expected := flag.String("expect-project", "", "Required Supabase project reference")
	report := flag.String("report", ".cache/migration-report.json", "Local report path (no credentials or row contents)")
	verify := flag.String("verify", "database/tests/analytics_invariants.sql", "SQL assertions, always rolled back to a savepoint")
	flag.Parse()
	if *expected == "" {
		return fmt.Errorf("--expect-project is required")
	}
	_ = godotenv.Load()
	cfg, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL")
	}
	pooler := strings.HasSuffix(strings.ToLower(cfg.Host), ".pooler.supabase.com") && cfg.User == "postgres."+*expected
	direct := strings.EqualFold(cfg.Host, "db."+*expected+".supabase.co") && cfg.User == "postgres"
	if !pooler && !direct {
		return fmt.Errorf("connection does not match the expected project")
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	db := stdlib.OpenDB(*cfg)
	defer db.Close()
	db.SetMaxOpenConns(1)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot start migration transaction")
	}
	defer tx.Rollback()
	for _, q := range []string{`SET LOCAL lock_timeout='5s'`, `SET LOCAL statement_timeout='45s'`, `SELECT pg_advisory_xact_lock(734218,1102026)`} {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `SELECT set_config('leguiburger.expected_project',$1,true)`, *expected); err != nil {
		return err
	}
	var contractExists bool
	if err = tx.QueryRowContext(ctx, `SELECT to_regclass('legui_private.database_contract') IS NOT NULL`).Scan(&contractExists); err != nil {
		return err
	}
	if contractExists {
		var project, application string
		if err = tx.QueryRowContext(ctx, `SELECT project_ref,application FROM legui_private.database_contract WHERE singleton`).Scan(&project, &application); err != nil {
			return err
		}
		if project != *expected || application != "leguiburger" {
			return fmt.Errorf("database identity contract does not match the target application/project")
		}
	}
	before, err := capture(ctx, tx, nil)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS legui_private;
 REVOKE ALL ON SCHEMA legui_private FROM PUBLIC,anon,authenticated,service_role;
 CREATE TABLE IF NOT EXISTS legui_private.schema_migrations(version text PRIMARY KEY,checksum text NOT NULL CHECK(length(checksum)=64),applied_at timestamptz NOT NULL DEFAULT clock_timestamp());
 REVOKE ALL ON legui_private.schema_migrations FROM PUBLIC,anon,authenticated,service_role;`)
	if err != nil {
		return err
	}
	files, err := filepath.Glob("database/migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no migrations found")
	}
	applied, skipped := []string{}, []string{}
	for _, file := range files {
		content, e := os.ReadFile(file)
		if e != nil {
			return e
		}
		if strings.Contains(string(content), "\r\n") {
			return fmt.Errorf("migration must use LF before applying: %s", file)
		}
		sum := sha256.Sum256(content)
		checksum := hex.EncodeToString(sum[:])
		version := strings.TrimSuffix(filepath.Base(file), ".sql")
		var existing string
		e = tx.QueryRowContext(ctx, `SELECT checksum FROM legui_private.schema_migrations WHERE version=$1`, version).Scan(&existing)
		if e == nil {
			if existing != checksum {
				return fmt.Errorf("applied migration checksum differs: %s", version)
			}
			skipped = append(skipped, version)
			continue
		}
		if e != sql.ErrNoRows {
			return e
		}
		fmt.Println("Executing", version)
		if _, e = tx.ExecContext(ctx, string(content)); e != nil {
			return fmt.Errorf("%s: %w", version, e)
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO legui_private.schema_migrations(version,checksum) VALUES($1,$2)`, version, checksum); e != nil {
			return e
		}
		applied = append(applied, version)
	}
	checks := false
	if *verify != "" {
		assertions, e := os.ReadFile(*verify)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `SAVEPOINT migration_assertions`); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, string(assertions)); e != nil {
			return fmt.Errorf("invariant checks: %w", e)
		}
		if _, e = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT migration_assertions`); e != nil {
			return e
		}
		checks = true
		fmt.Println("Invariant checks passed; all fixtures rolled back")
	}
	after, err := capture(ctx, tx, before)
	if err != nil {
		return err
	}
	allowedSeeds := map[string]bool{"public.order_statuses_lookup": true, "public.payment_statuses_lookup": true, "public.price_history": true, "public.supply_cost_history": true}
	for table, old := range before {
		next := after[table]
		if old.Count == 0 && allowedSeeds[table] {
			continue
		}
		if old.Count != next.Count || old.Digest != next.Digest {
			return fmt.Errorf("existing rows changed unexpectedly in %s", table)
		}
	}
	fmt.Println("Existing row values and counts preserved")
	state := "rehearsed_and_rolled_back"
	if *apply {
		if err = tx.Commit(); err != nil {
			return err
		}
		state = "applied"
	} else {
		if err = tx.Rollback(); err != nil {
			return err
		}
	}
	result := map[string]any{"project": *expected, "status": state, "applied": applied, "skipped": skipped, "assertions_passed": checks, "original_rows_preserved": true, "before": before, "after": after, "finished_at": time.Now().UTC().Format(time.RFC3339)}
	encoded, _ := json.MarshalIndent(result, "", "  ")
	if err = os.MkdirAll(filepath.Dir(*report), 0700); err != nil {
		return err
	}
	if err = os.WriteFile(*report, encoded, 0600); err != nil {
		return err
	}
	fmt.Printf("%s: %d migrations, %d already applied. Report: %s\n", state, len(applied), len(skipped), *report)
	return nil
}
func capture(ctx context.Context, tx *sql.Tx, original map[string]snapshot) (map[string]snapshot, error) {
	result := map[string]snapshot{}
	if original == nil {
		rows, err := tx.QueryContext(ctx, `SELECT c.table_schema,c.table_name,c.column_name FROM information_schema.columns c JOIN information_schema.tables t USING(table_schema,table_name) WHERE c.table_schema IN ('public','legui_private') AND c.table_name<>'schema_migrations' AND t.table_type='BASE TABLE' ORDER BY c.table_schema,c.table_name,c.ordinal_position`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var schema, table, column string
			if err = rows.Scan(&schema, &table, &column); err != nil {
				rows.Close()
				return nil, err
			}
			key := schema + "." + table
			s := result[key]
			s.Schema = schema
			s.Table = table
			s.Columns = append(s.Columns, column)
			result[key] = s
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	} else {
		for table, s := range original {
			result[table] = snapshot{Schema: s.Schema, Table: s.Table, Columns: s.Columns}
		}
	}
	tables := make([]string, 0, len(result))
	for table := range result {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		s := result[table]
		cols := make([]string, len(s.Columns))
		for i, c := range s.Columns {
			cols[i] = pgx.Identifier{c}.Sanitize()
		}
		query := `SELECT count(*),md5(coalesce(string_agg(md5(row_to_json(__legui_snapshot)::text),'' ORDER BY md5(row_to_json(__legui_snapshot)::text)),'')) FROM (SELECT ` + strings.Join(cols, ",") + ` FROM ` + pgx.Identifier{s.Schema, s.Table}.Sanitize() + `) __legui_snapshot`
		if err := tx.QueryRowContext(ctx, query).Scan(&s.Count, &s.Digest); err != nil {
			return nil, err
		}
		result[table] = s
	}
	return result, nil
}
