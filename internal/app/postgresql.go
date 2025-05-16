package app

import (
	"fmt"

	"github.com/ductm54/transfer-track/internal/dbutil"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:sql driver name: "postgres"
	"github.com/urfave/cli/v2"
)

var (
	// PostgresHost is the CLI flag for PostgreSQL host.
	PostgresHost = cli.StringFlag{ //nolint:gochecknoglobals
		Name:    "postgres-host",
		Usage:   "PostgresSQL host to connect",
		EnvVars: []string{"POSTGRES_HOST"},
		Value:   "127.0.0.1",
	}
	// PostgresPort is the CLI flag for PostgreSQL port.
	PostgresPort = cli.IntFlag{ //nolint:gochecknoglobals
		Name:    "postgres-port",
		Usage:   "PostgresSQL port to connect",
		EnvVars: []string{"POSTGRES_PORT"},
		Value:   5432, //nolint:gomnd
	}
	// PostgresUser is the CLI flag for PostgreSQL user.
	PostgresUser = cli.StringFlag{ //nolint:gochecknoglobals
		Name:    "postgres-user",
		Usage:   "PostgresSQL user to connect",
		EnvVars: []string{"POSTGRES_USER"},
		Value:   "go_project_template",
	}
	// PostgresPassword is the CLI flag for PostgreSQL password.
	PostgresPassword = cli.StringFlag{ //nolint:gochecknoglobals
		Name:    "postgres-password",
		Usage:   "PostgresSQL password to connect",
		EnvVars: []string{"POSTGRES_PASSWORD"},
		Value:   "go_project_template",
	}
	// PostgresDatabase is the CLI flag for PostgreSQL database name.
	PostgresDatabase = cli.StringFlag{ //nolint:gochecknoglobals
		Name:    "postgres-database",
		Usage:   "Postgres database to connect",
		EnvVars: []string{"POSTGRES_DATABASE"},
		Value:   "go_project_template",
	}
	// PostgresMigrationPath is the CLI flag for database migration path.
	PostgresMigrationPath = cli.StringFlag{ //nolint:gochecknoglobals
		Name:    "migration-path",
		Value:   "migrations",
		EnvVars: []string{"MIGRATION_PATH"},
	}
)

// PostgresSQLFlags creates new cli flags for PostgreSQL client.
func PostgresSQLFlags(defaultDB string) []cli.Flag {
	db := PostgresDatabase
	db.Value = defaultDB

	return []cli.Flag{
		&PostgresHost,
		&PostgresPort,
		&PostgresUser,
		&PostgresPassword,
		&db,
		&PostgresMigrationPath,
	}
}

// NewDB creates a DB instance from cli flags configuration.
func NewDB(specs map[string]any) (*sqlx.DB, error) {
	const driverName = "postgres"

	connStr := dbutil.FormatDSN(specs)

	db, err := sqlx.Connect(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, nil
}
