// Package storage provides database operations for the transfer tracking service.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Storage handles database operations.
type Storage struct {
	db     *sqlx.DB
	logger *zap.SugaredLogger
}

// New creates a new Storage instance.
func New(db *sqlx.DB, logger *zap.SugaredLogger) *Storage {
	return &Storage{
		db:     db,
		logger: logger,
	}
}

// SourceAddress represents a source address to track.
type SourceAddress struct {
	ID        int64     `db:"id" json:"id"`
	Address   string    `db:"address" json:"address"`
	Label     string    `db:"label" json:"label"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// TargetAddress represents a target address to track.
type TargetAddress struct {
	ID        int64     `db:"id" json:"id"`
	Address   string    `db:"address" json:"address"`
	Label     string    `db:"label" json:"label"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Token represents an ERC20 token to track.
type Token struct {
	ID        int64     `db:"id" json:"id"`
	Address   string    `db:"address" json:"address"`
	Symbol    string    `db:"symbol" json:"symbol"`
	Name      string    `db:"name" json:"name"`
	Decimals  int       `db:"decimals" json:"decimals"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Transfer represents a token transfer.
type Transfer struct {
	ID           int64     `db:"id" json:"id"`
	Hash         string    `db:"hash" json:"hash"`
	BlockNumber  int64     `db:"block_number" json:"block_number"`
	Timestamp    time.Time `db:"timestamp" json:"timestamp"`
	FromAddress  string    `db:"from_address" json:"from_address"`
	ToAddress    string    `db:"to_address" json:"to_address"`
	TokenAddress string    `db:"token_address" json:"token_address"`
	Amount       string    `db:"amount" json:"amount"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// Config represents a system configuration entry.
type Config struct {
	ID        int64     `db:"id" json:"id"`
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// TokenAmount represents the total amount of a token transferred.
type TokenAmount struct {
	TokenAddress string `db:"token_address" json:"token_address"`
	Symbol       string `db:"symbol" json:"symbol"`
	Name         string `db:"name" json:"name"`
	Decimals     int    `db:"decimals" json:"decimals"`
	TotalAmount  string `db:"total_amount" json:"total_amount"`
	// NormalizedAmount is calculated as TotalAmount / 10^Decimals
	NormalizedAmount string `json:"normalized_amount"`
}

// AddSourceAddress adds a new source address.
func (s *Storage) AddSourceAddress(ctx context.Context, address, label string) (*SourceAddress, error) {
	// Normalize address to lowercase
	address = strings.ToLower(address)

	query := `
		INSERT INTO source_addresses (address, label, updated_at)
		VALUES ($1, $2, NOW())
		RETURNING id, address, label, created_at, updated_at
	`

	var result SourceAddress
	err := s.db.GetContext(ctx, &result, query, address, label)

	if err != nil {
		return nil, fmt.Errorf("adding source address: %w", err)
	}

	return &result, nil
}

// GetSourceAddresses retrieves all source addresses.
func (s *Storage) GetSourceAddresses(ctx context.Context) ([]SourceAddress, error) {
	query := `SELECT id, address, label, created_at, updated_at FROM source_addresses ORDER BY id`

	var addresses []SourceAddress
	err := s.db.SelectContext(ctx, &addresses, query)

	if err != nil {
		return nil, fmt.Errorf("getting source addresses: %w", err)
	}

	return addresses, nil
}

// DeleteSourceAddress deletes a source address.
func (s *Storage) DeleteSourceAddress(ctx context.Context, id int64) error {
	query := `DELETE FROM source_addresses WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting source address: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// AddTargetAddress adds a new target address.
func (s *Storage) AddTargetAddress(ctx context.Context, address, label string) (*TargetAddress, error) {
	// Normalize address to lowercase
	address = strings.ToLower(address)

	query := `
		INSERT INTO target_addresses (address, label, updated_at)
		VALUES ($1, $2, NOW())
		RETURNING id, address, label, created_at, updated_at
	`

	var result TargetAddress
	err := s.db.GetContext(ctx, &result, query, address, label)

	if err != nil {
		return nil, fmt.Errorf("adding target address: %w", err)
	}

	return &result, nil
}

// GetTargetAddresses retrieves all target addresses.
func (s *Storage) GetTargetAddresses(ctx context.Context) ([]TargetAddress, error) {
	query := `SELECT id, address, label, created_at, updated_at FROM target_addresses ORDER BY id`

	var addresses []TargetAddress
	err := s.db.SelectContext(ctx, &addresses, query)

	if err != nil {
		return nil, fmt.Errorf("getting target addresses: %w", err)
	}

	return addresses, nil
}

// DeleteTargetAddress deletes a target address.
func (s *Storage) DeleteTargetAddress(ctx context.Context, id int64) error {
	query := `DELETE FROM target_addresses WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting target address: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// AddToken adds a new token to track.
func (s *Storage) AddToken(ctx context.Context, address, symbol, name string, decimals int) (*Token, error) {
	// Normalize address to lowercase
	address = strings.ToLower(address)

	query := `
		INSERT INTO tokens (address, symbol, name, decimals, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, address, symbol, name, decimals, created_at, updated_at
	`

	var result Token
	err := s.db.GetContext(ctx, &result, query, address, symbol, name, decimals)

	if err != nil {
		return nil, fmt.Errorf("adding token: %w", err)
	}

	return &result, nil
}

// GetTokens retrieves all tokens.
func (s *Storage) GetTokens(ctx context.Context) ([]Token, error) {
	query := `SELECT id, address, symbol, name, decimals, created_at, updated_at FROM tokens ORDER BY id`

	var tokens []Token
	err := s.db.SelectContext(ctx, &tokens, query)

	if err != nil {
		return nil, fmt.Errorf("getting tokens: %w", err)
	}

	return tokens, nil
}

// DeleteToken deletes a token.
func (s *Storage) DeleteToken(ctx context.Context, id int64) error {
	query := `DELETE FROM tokens WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// AddTransfersBatch adds multiple transfers in a single transaction.
func (s *Storage) AddTransfersBatch(ctx context.Context, transfers []*Transfer) error {
	if len(transfers) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	// Ensure transaction is rolled back if an error occurs
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				// We can't return the error here, so we just log it
				s.logger.Errorw("Failed to rollback transaction", "err", rollbackErr)
			}
		}
	}()

	// Prepare the statement
	stmt, err := tx.PreparexContext(ctx, `
		INSERT INTO transfers (hash, block_number, timestamp, from_address, to_address, token_address, amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (hash, token_address, from_address, to_address) DO NOTHING
	`)

	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}

	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			s.logger.Warnw("Failed to close statement", "err", closeErr)
		}
	}()

	// Execute the statement for each transfer
	for _, transfer := range transfers {
		// Normalize addresses to lowercase
		transfer.FromAddress = strings.ToLower(transfer.FromAddress)
		transfer.ToAddress = strings.ToLower(transfer.ToAddress)
		transfer.TokenAddress = strings.ToLower(transfer.TokenAddress)

		_, err = stmt.ExecContext(
			ctx,
			transfer.Hash,
			transfer.BlockNumber,
			transfer.Timestamp,
			transfer.FromAddress,
			transfer.ToAddress,
			transfer.TokenAddress,
			transfer.Amount,
		)

		if err != nil {
			return fmt.Errorf("executing statement: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetTotalAmounts retrieves the total amounts of each token transferred from source addresses to target addresses.
func (s *Storage) GetTotalAmounts(ctx context.Context, startTime, endTime time.Time) ([]TokenAmount, error) {
	query := `
		SELECT
			t.token_address,
			tk.symbol,
			tk.name,
			tk.decimals,
			SUM(t.amount) as total_amount
		FROM
			transfers t
		JOIN
			tokens tk ON t.token_address = tk.address
		WHERE
			t.from_address IN (SELECT address FROM source_addresses)
			AND t.to_address IN (SELECT address FROM target_addresses)
			AND t.timestamp BETWEEN $1 AND $2
		GROUP BY
			t.token_address, tk.symbol, tk.name, tk.decimals
		ORDER BY
			tk.symbol
	`

	var amounts []TokenAmount
	err := s.db.SelectContext(ctx, &amounts, query, startTime, endTime)

	if err != nil {
		return nil, fmt.Errorf("getting total amounts: %w", err)
	}

	return amounts, nil
}

// GetLastProcessedBlock retrieves the last processed block number for a specific address and token.
// If tokenAddress is empty or "0x0000000000000000000000000000000000000000",
// it returns the last block for ETH transfers.
// Otherwise, it returns the last block for the specified ERC20 token transfers.
func (s *Storage) GetLastProcessedBlock(ctx context.Context, address, tokenAddress string) (int64, error) {
	// Normalize addresses to lowercase
	address = strings.ToLower(address)
	tokenAddress = strings.ToLower(tokenAddress)

	// Check if we're looking for ETH or ERC20 transfers
	var query string

	var args []any

	if tokenAddress == "" || tokenAddress == "0x0000000000000000000000000000000000000000" {
		// For ETH transfers
		query = `
			SELECT COALESCE(MAX(block_number), 0) as last_block
			FROM transfers
			WHERE (from_address = $1 OR to_address = $1)
			AND token_address = '0x0000000000000000000000000000000000000000'
		`
		args = []any{address}
	} else {
		// For specific ERC20 token transfers
		query = `
			SELECT COALESCE(MAX(block_number), 0) as last_block
			FROM transfers
			WHERE (from_address = $1 OR to_address = $1)
			AND token_address = $2
		`
		args = []any{address, tokenAddress}
	}

	var lastBlock int64
	err := s.db.GetContext(ctx, &lastBlock, query, args...)

	if err != nil {
		return 0, fmt.Errorf("getting last processed block for address %s and token %s: %w", address, tokenAddress, err)
	}

	return lastBlock, nil
}

// GetLastProcessedBlockForERC20 retrieves the minimum last processed block number for a specific address across all ERC20 tokens.
// This is useful for fetching all ERC20 transfers in a single query.
func (s *Storage) GetLastProcessedBlockForERC20(ctx context.Context, address string) (int64, error) {
	// Normalize address to lowercase
	address = strings.ToLower(address)

	// Query for the minimum block number across all ERC20 tokens
	// We use MIN to ensure we don't miss any transfers
	query := `
		SELECT COALESCE(MIN(last_block), 0) as min_last_block
		FROM (
			SELECT token_address, COALESCE(MAX(block_number), 0) as last_block
			FROM transfers
			WHERE (from_address = $1 OR to_address = $1)
			AND token_address != '0x0000000000000000000000000000000000000000'
			GROUP BY token_address
		) as token_blocks
	`

	var lastBlock int64
	err := s.db.GetContext(ctx, &lastBlock, query, address)

	if err != nil {
		return 0, fmt.Errorf("getting last processed block for ERC20 tokens for address %s: %w", address, err)
	}

	return lastBlock, nil
}

// GetConfig retrieves a configuration value.
func (s *Storage) GetConfig(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM config WHERE key = $1`

	var value string
	err := s.db.GetContext(ctx, &value, query, key)

	if err != nil {
		return "", fmt.Errorf("getting config %s: %w", key, err)
	}

	return value, nil
}

// UpdateConfig updates a configuration value.
func (s *Storage) UpdateConfig(ctx context.Context, key, value string) error {
	query := `
		UPDATE config
		SET value = $2, updated_at = NOW()
		WHERE key = $1
	`

	result, err := s.db.ExecContext(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("updating config %s: %w", key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Insert if not exists
		insertQuery := `
			INSERT INTO config (key, value)
			VALUES ($1, $2)
		`

		_, err := s.db.ExecContext(ctx, insertQuery, key, value)
		if err != nil {
			return fmt.Errorf("inserting config %s: %w", key, err)
		}
	}

	return nil
}
