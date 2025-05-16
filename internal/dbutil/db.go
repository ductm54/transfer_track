// Package dbutil provides database utility functions.
package dbutil

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" //nolint:go migrate
	"github.com/jmoiron/sqlx"
)

// NewDB creates a DB instance from dsn.
func NewDB(dsn string) (*sqlx.DB, error) {
	const driverName = "postgres"

	db, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	return db, nil
}

// FormatDSN formats a PostgreSQL connection string.
func FormatDSN(props map[string]any) string {
	// Format: "host=localhost port=5432 user=postgres password=postgres dbname=mydb sslmode=disable"
	parts := make([]string, 0, len(props))
	for k, v := range props {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	return strings.Join(parts, " ")
}

// RunMigrationUp runs database migrations from the specified folder.
func RunMigrationUp(db *sql.DB, migrationFolderPath, databaseName string) (*migrate.Migrate, error) {
	driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationFolderPath),
		databaseName, driver,
	)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return m, nil
}
