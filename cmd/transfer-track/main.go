// Package main is the entry point for the transfer-track service.
// It initializes the database, services, and HTTP server.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ductm54/transfer-track/internal/api"
	libapp "github.com/ductm54/transfer-track/internal/app"
	"github.com/ductm54/transfer-track/internal/dbutil"
	"github.com/ductm54/transfer-track/internal/scheduler"
	"github.com/ductm54/transfer-track/internal/server"
	"github.com/ductm54/transfer-track/internal/service"
	"github.com/ductm54/transfer-track/internal/storage"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load(".env")
	app := libapp.NewApp()
	app.Name = "transfer-track"
	app.Usage = "Track ETH and ERC20 token transfers"

	// Add PostgreSQL flags
	app.Flags = append(app.Flags,
		&libapp.PostgresHost,
		&libapp.PostgresPort,
		&libapp.PostgresUser,
		&libapp.PostgresPassword,
		&libapp.PostgresDatabase,
		&libapp.PostgresMigrationPath,
		&cli.StringFlag{
			Name:    "bind-addr",
			Value:   ":8080",
			Usage:   "HTTP server bind address",
			EnvVars: []string{"BIND_ADDR"},
		},
		&cli.StringFlag{
			Name:    "etherscan-api-key",
			Usage:   "Etherscan API key",
			EnvVars: []string{"ETHERSCAN_API_KEY"},
		},
		&cli.IntFlag{
			Name:    "refresh-interval",
			Value:   1,
			Usage:   "Minimum refresh interval in hours",
			EnvVars: []string{"REFRESH_INTERVAL_HOURS"},
		},
		&cli.StringFlag{
			Name:    "daily-refresh-time",
			Value:   "00:00:00",
			Usage:   "Daily refresh time (HH:MM:SS)",
			EnvVars: []string{"DAILY_REFRESH_TIME"},
		},
		&cli.IntFlag{
			Name:    "chain-id",
			Value:   1,
			Usage:   "Blockchain chain ID (1 for Ethereum Mainnet)",
			EnvVars: []string{"CHAIN_ID"},
		},
	)
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		log.Panic(err)
	}
}

func run(c *cli.Context) error {
	// Initialize logger
	logger, _, flush, err := libapp.NewLogger(c)
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}
	defer flush()

	zap.ReplaceGlobals(logger)
	l := logger.Sugar()
	l.Infow("Transfer Track service starting...")

	// Initialize database
	db, err := initDB(c)
	if err != nil {
		l.Panicw("cannot init DB", "err", err)
	}

	// Initialize storage
	store := storage.New(db, l)

	// Initialize transfer service
	transferService, err := service.NewTransferService(
		store,
		l,
		c.String("etherscan-api-key"),
		c.Int("refresh-interval"),
		c.String("daily-refresh-time"),
		c.Int("chain-id"),
	)
	if err != nil {
		l.Panicw("cannot create transfer service", "err", err)
	}

	l.Infow("Using Etherscan API v2", "chainID", c.Int("chain-id"))

	// Initialize scheduler
	sched := scheduler.NewScheduler(transferService, l)
	sched.Start()
	defer sched.Stop()

	// Initialize API handlers
	handler := api.NewHandler(transferService, store, l)

	// Initialize HTTP server
	bindAddr := c.String("bind-addr")
	srv := server.New(bindAddr)
	handler.RegisterRoutes(srv.GetEngine())

	// Start HTTP server
	errCh := make(chan error, 1)
	go func() {
		l.Infow("Starting HTTP server", "addr", bindAddr)
		errCh <- srv.Run()
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either server error or interrupt signal
	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-sigCh:
		l.Infow("Received signal, shutting down", "signal", sig)
	}

	// Perform initial data fetch if needed
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	shouldRefresh, err := transferService.ShouldRefreshData(ctx)
	if err == nil && shouldRefresh {
		l.Infow("Performing initial data fetch")
		err := transferService.FetchAndStoreTransfers(ctx)
		if err != nil {
			l.Warnw("Error performing initial data fetch", "err", err)
		}
	}

	return nil
}

func initDB(c *cli.Context) (*sqlx.DB, error) {
	db, err := libapp.NewDB(map[string]any{
		"host":     c.String(libapp.PostgresHost.Name),
		"port":     c.Int(libapp.PostgresPort.Name),
		"user":     c.String(libapp.PostgresUser.Name),
		"password": c.String(libapp.PostgresPassword.Name),
		"dbname":   c.String(libapp.PostgresDatabase.Name),
		"sslmode":  "disable",
	})
	if err != nil {
		return nil, fmt.Errorf("creating database connection: %w", err)
	}

	_, err = dbutil.RunMigrationUp(db.DB, c.String(libapp.PostgresMigrationPath.Name),
		c.String(libapp.PostgresDatabase.Name))
	if err != nil {
		return nil, fmt.Errorf("running database migrations: %w", err)
	}

	return db, nil
}
