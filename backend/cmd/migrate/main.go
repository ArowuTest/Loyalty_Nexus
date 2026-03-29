// Build: 2026-03-29T01:58:11Z
// cmd/migrate/main.go — Production database migration runner for Loyalty Nexus.
//
// Usage:
//
//	./migrate [up|down|version|force VERSION]
//
// Environment variables:
//
//	MIGRATE_DATABASE_URL — PostgreSQL connection string for migrations (preferred; supports external hostname + SSL)
//	DATABASE_URL         — PostgreSQL connection string fallback
//	MIGRATIONS_DIR       — path to the migrations directory (default: /app/migrations)
//
// This binary is built as a separate Docker stage and called by Render's
// preDeployCommand before the API server starts. It uses golang-migrate/v4
// which maintains a schema_migrations table to track applied versions,
// making every run idempotent — safe to call on every deploy.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Prefer MIGRATE_DATABASE_URL (external hostname + SSL) for preDeployCommand context.
	// Fall back to DATABASE_URL for runtime entrypoint usage.
	dbURL := os.Getenv("MIGRATE_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		log.Fatal("[migrate] Neither MIGRATE_DATABASE_URL nor DATABASE_URL is set")
	}
	log.Printf("[migrate] Using database host from URL (first 40 chars): %.40s...", dbURL)

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "/app/migrations"
	}

	sourceURL := "file://" + migrationsDir

	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		log.Fatalf("[migrate] failed to initialise migrate: %v", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("[migrate] source close error: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("[migrate] db close error: %v", dbErr)
		}
	}()

	// Enable verbose logging so Render build logs show each migration applied.
	m.Log = &migrateLogger{}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		if err := m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				log.Println("[migrate] No new migrations to apply — schema is up to date.")
				os.Exit(0)
			}
			log.Fatalf("[migrate] up failed: %v", err)
		}
		version, dirty, _ := m.Version()
		log.Printf("[migrate] Successfully migrated to version %d (dirty=%v)", version, dirty)

	case "fix-and-up":
		// Auto-detect and fix dirty migration state, then run all pending migrations.
		// This is safe for both fresh DBs and DBs with a dirty migration flag.
		version, dirty, vErr := m.Version()
		if vErr != nil && !errors.Is(vErr, migrate.ErrNilVersion) {
			log.Fatalf("[migrate] fix-and-up: version check failed: %v", vErr)
		}
		if dirty {
			// Rewind to the version BEFORE the dirty one so it gets re-run cleanly.
			prevVersion := int(version) - 1
			if prevVersion < 0 {
				prevVersion = 0
			}
			log.Printf("[migrate] fix-and-up: dirty version %d detected — rewinding to v%d", version, prevVersion)
			if err := m.Force(prevVersion); err != nil {
				log.Fatalf("[migrate] fix-and-up: force v%d failed: %v", prevVersion, err)
			}
			log.Printf("[migrate] fix-and-up: dirty flag cleared, will re-run v%d", version)
		} else if errors.Is(vErr, migrate.ErrNilVersion) {
			log.Println("[migrate] fix-and-up: fresh database (no migrations applied yet)")
		} else {
			log.Printf("[migrate] fix-and-up: clean state at v%d — no fix needed", version)
		}
		// Now run all pending migrations
		if err := m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				log.Println("[migrate] fix-and-up: no new migrations to apply — schema up to date.")
				os.Exit(0)
			}
			log.Fatalf("[migrate] fix-and-up: up failed: %v", err)
		}
		finalVersion, finalDirty, _ := m.Version()
		log.Printf("[migrate] fix-and-up: completed — now at version %d (dirty=%v)", finalVersion, finalDirty)

	case "down":
		if err := m.Steps(-1); err != nil {
			log.Fatalf("[migrate] down failed: %v", err)
		}
		version, dirty, _ := m.Version()
		log.Printf("[migrate] Rolled back one step — now at version %d (dirty=%v)", version, dirty)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				log.Println("[migrate] No migrations have been applied yet (version: nil)")
				os.Exit(0)
			}
			log.Fatalf("[migrate] version check failed: %v", err)
		}
		log.Printf("[migrate] Current version: %d (dirty=%v)", version, dirty)

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("[migrate] force requires a version number: ./migrate force VERSION")
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("[migrate] invalid version number %q: %v", os.Args[2], err)
		}
		if err := m.Force(v); err != nil {
			log.Fatalf("[migrate] force failed: %v", err)
		}
		log.Printf("[migrate] Forced version to %d", v)

	default:
		fmt.Fprintf(os.Stderr, "Usage: migrate [up|down|version|force VERSION]\n")
		os.Exit(1)
	}
}

// migrateLogger implements migrate.Logger to route output through the standard
// log package so all migration output appears in Render's structured log stream.
type migrateLogger struct{}

func (l *migrateLogger) Printf(format string, v ...interface{}) {
	log.Printf("[migrate] "+format, v...)
}

func (l *migrateLogger) Verbose() bool {
	return true
}
