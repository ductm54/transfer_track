// Package service provides business logic for the transfer tracking service.
package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ductm54/transfer-track/internal/etherscan"
	"github.com/ductm54/transfer-track/internal/storage"
	"go.uber.org/zap"
)

// Configuration keys for database storage.
const (
	// Database config keys.
	configKeyLastETHUpdate       = "last_eth_update"
	configKeyLastTokenUpdate     = "last_token_update"
	configKeyDailyRefreshTime    = "daily_refresh_time"
	configKeyMinRefreshInterval  = "min_refresh_interval_hours"
	defaultMinRefreshIntervalHrs = 1
)

// TransferService handles the transfer tracking logic.
type TransferService struct {
	store        *storage.Storage
	etherscanAPI *etherscan.Client
	logger       *zap.SugaredLogger
}

// NewTransferService creates a new TransferService.
func NewTransferService(
	store *storage.Storage, logger *zap.SugaredLogger, apiKey string,
	refreshInterval int, dailyRefreshTime string, chainID ...int,
) (*TransferService, error) {
	ctx := context.Background()

	// Only use the API key from the environment
	if apiKey == "" {
		logger.Warnw("No Etherscan API key provided, API calls will likely fail")
	}

	// Store refresh interval if provided
	if refreshInterval > 0 {
		err := store.UpdateConfig(ctx, configKeyMinRefreshInterval, strconv.Itoa(refreshInterval))
		if err != nil {
			logger.Warnw("Failed to store refresh interval in config", "err", err)
		}
	}

	// Store daily refresh time if provided
	if dailyRefreshTime != "" {
		// Validate time format
		_, err := time.Parse("15:04:05", dailyRefreshTime)
		if err == nil {
			err = store.UpdateConfig(ctx, configKeyDailyRefreshTime, dailyRefreshTime)
			if err != nil {
				logger.Warnw("Failed to store daily refresh time in config", "err", err)
			}
		} else {
			logger.Warnw("Invalid daily refresh time format", "time", dailyRefreshTime, "err", err)
		}
	}

	// Create Etherscan client with chain ID if provided
	var etherscanClient *etherscan.Client

	if len(chainID) > 0 && chainID[0] > 0 {
		logger.Infow("Using custom chain ID for Etherscan API", "chainID", chainID[0])
		etherscanClient = etherscan.NewClientWithChainID(apiKey, logger, chainID[0])
	} else {
		logger.Infow("Using default chain ID (1) for Etherscan API")
		etherscanClient = etherscan.NewClient(apiKey, logger)
	}

	return &TransferService{
		store:        store,
		etherscanAPI: etherscanClient,
		logger:       logger,
	}, nil
}

// UpdateRefreshInterval updates the minimum refresh interval in hours.
func (s *TransferService) UpdateRefreshInterval(ctx context.Context, hours int) error {
	if hours < 1 {
		return fmt.Errorf("refresh interval must be at least 1 hour")
	}

	err := s.store.UpdateConfig(ctx, configKeyMinRefreshInterval, strconv.Itoa(hours))
	if err != nil {
		return fmt.Errorf("updating refresh interval: %w", err)
	}

	return nil
}

// GetRefreshInterval gets the minimum refresh interval in hours.
func (s *TransferService) GetRefreshInterval(ctx context.Context) (int, error) {
	value, err := s.store.GetConfig(ctx, configKeyMinRefreshInterval)
	if err != nil {
		return defaultMinRefreshIntervalHrs, fmt.Errorf("getting refresh interval: %w", err)
	}

	hours, err := strconv.Atoi(value)
	if err != nil {
		return defaultMinRefreshIntervalHrs, fmt.Errorf("parsing refresh interval: %w", err)
	}

	return hours, nil
}

// UpdateDailyRefreshTime updates the daily refresh time.
func (s *TransferService) UpdateDailyRefreshTime(ctx context.Context, timeStr string) error {
	// Validate time format (HH:MM:SS)
	_, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		return fmt.Errorf("invalid time format, expected HH:MM:SS: %w", err)
	}

	err = s.store.UpdateConfig(ctx, configKeyDailyRefreshTime, timeStr)
	if err != nil {
		return fmt.Errorf("updating daily refresh time: %w", err)
	}

	return nil
}

// GetDailyRefreshTime gets the daily refresh time.
func (s *TransferService) GetDailyRefreshTime(ctx context.Context) (string, error) {
	value, err := s.store.GetConfig(ctx, configKeyDailyRefreshTime)
	if err != nil {
		return "00:00:00", fmt.Errorf("getting daily refresh time: %w", err)
	}

	return value, nil
}

// ShouldRefreshData checks if data should be refreshed based on last update time.
func (s *TransferService) ShouldRefreshData(ctx context.Context) (bool, error) {
	// Get last update time
	lastETHUpdateStr, err := s.store.GetConfig(ctx, configKeyLastETHUpdate)
	if err != nil {
		s.logger.Warnw("Failed to get last ETH update time, assuming refresh is needed", "err", err)
		return true, nil // If error, assume refresh is needed
	}

	lastETHUpdate, err := time.Parse(time.RFC3339, lastETHUpdateStr)
	if err != nil {
		s.logger.Warnw("Failed to parse last ETH update time, assuming refresh is needed",
			"err", err, "timeStr", lastETHUpdateStr)
		return true, nil // If error, assume refresh is needed
	}

	// Get refresh interval
	refreshInterval, err := s.GetRefreshInterval(ctx)
	if err != nil {
		s.logger.Warnw("Failed to get refresh interval, using default", "err", err, "default", defaultMinRefreshIntervalHrs)
		refreshInterval = defaultMinRefreshIntervalHrs
	}

	// Check if enough time has passed since last update
	return time.Since(lastETHUpdate) > time.Duration(refreshInterval)*time.Hour, nil
}

// FetchAndStoreTransfers fetches and stores transfers for all source addresses and tokens.
func (s *TransferService) FetchAndStoreTransfers(ctx context.Context) error {
	// Always fetch the latest data for manual refresh
	s.logger.Infow("Fetching latest transfer data")

	// Get source addresses
	sourceAddresses, err := s.store.GetSourceAddresses(ctx)
	if err != nil {
		return fmt.Errorf("getting source addresses: %w", err)
	}

	if len(sourceAddresses) == 0 {
		s.logger.Infow("No source addresses configured, skipping transfer fetch")
		return nil
	}

	// We don't need to get tokens anymore since we fetch all ERC20 transfers in a single query

	// Set time range (last 30 days by default)
	endTime := time.Now()
	startTime := endTime.AddDate(0, -1, 0) // 1 month ago

	// Process each source address
	for _, sourceAddr := range sourceAddresses {
		// Fetch ETH transfers
		err = s.fetchAndStoreETHTransfers(ctx, sourceAddr.Address, startTime, endTime)
		if err != nil {
			s.logger.Errorw("Error fetching ETH transfers", "address", sourceAddr.Address, "err", err)
			continue
		}

		// Fetch all ERC20 transfers in a single query
		err = s.fetchAndStoreAllERC20Transfers(ctx, sourceAddr.Address, startTime, endTime)
		if err != nil {
			s.logger.Errorw("Error fetching ERC20 transfers", "address", sourceAddr.Address, "err", err)
			continue
		}
	}

	// Update last update time
	now := time.Now().Format(time.RFC3339)

	err = s.store.UpdateConfig(ctx, configKeyLastETHUpdate, now)
	if err != nil {
		s.logger.Errorw("Error updating last ETH update time", "err", err)
	}

	err = s.store.UpdateConfig(ctx, configKeyLastTokenUpdate, now)
	if err != nil {
		s.logger.Errorw("Error updating last token update time", "err", err)
	}

	return nil
}

// fetchAndStoreETHTransfers fetches and stores ETH transfers for a specific address.
func (s *TransferService) fetchAndStoreETHTransfers(ctx context.Context, address string, startTime, endTime time.Time) error {
	// ETH token address is 0x0000000000000000000000000000000000000000
	ethTokenAddress := "0x0000000000000000000000000000000000000000"

	// Get the last processed block for this address and ETH
	lastBlock, err := s.store.GetLastProcessedBlock(ctx, address, ethTokenAddress)
	if err != nil {
		s.logger.Warnw("Failed to get last processed block for ETH, starting from block 0",
			"address", address,
			"err", err)
		// Start from block 0 if we couldn't get the last processed block
		lastBlock = 0
	}

	s.logger.Infow("Fetching ETH transfers",
		"address", address,
		"startTime", startTime,
		"endTime", endTime,
		"lastProcessedBlock", lastBlock)

	// Fetch ETH transfers starting from the last processed block
	transactions, err := s.etherscanAPI.GetETHTransfers(ctx, address, startTime, endTime, lastBlock)
	if err != nil {
		return fmt.Errorf("fetching ETH transfers: %w", err)
	}

	s.logger.Infow("Fetched ETH transfers", "address", address, "count", len(transactions))

	// Prepare batch of transfers with preallocated capacity
	transfers := make([]*storage.Transfer, 0, len(transactions))

	// Process transactions
	for _, tx := range transactions {
		// Skip failed transactions
		if tx.IsError != "0" {
			continue
		}

		// Parse block number
		blockNumber, err := strconv.ParseInt(tx.BlockNumber, 10, 64)
		if err != nil {
			s.logger.Warnw("Failed to parse block number", "err", err, "blockNumber", tx.BlockNumber)
			continue
		}

		// Parse timestamp
		timestamp, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil {
			s.logger.Warnw("Failed to parse timestamp", "err", err, "timestamp", tx.TimeStamp)
			continue
		}

		// Create transfer record
		transfer := &storage.Transfer{
			Hash:         tx.Hash,
			BlockNumber:  blockNumber,
			Timestamp:    time.Unix(timestamp, 0),
			FromAddress:  tx.From,
			ToAddress:    tx.To,
			TokenAddress: "0x0000000000000000000000000000000000000000", // ETH
			Amount:       tx.Value,
		}

		// Add to batch
		transfers = append(transfers, transfer)
	}

	// Store transfers in batch
	if len(transfers) > 0 {
		err = s.store.AddTransfersBatch(ctx, transfers)

		if err != nil {
			s.logger.Errorw("Failed to store ETH transfers batch", "err", err, "count", len(transfers))
			return fmt.Errorf("storing ETH transfers batch: %w", err)
		}

		s.logger.Infow("Stored ETH transfers batch", "count", len(transfers))
	}

	return nil
}

// fetchAndStoreAllERC20Transfers fetches and stores all ERC20 transfers for a specific address
// in a single query.
func (s *TransferService) fetchAndStoreAllERC20Transfers(ctx context.Context, address string, startTime, endTime time.Time) error {
	// Get the last processed block for ERC20 transfers
	lastBlock, err := s.store.GetLastProcessedBlockForERC20(ctx, address)
	if err != nil {
		s.logger.Warnw("Failed to get last processed block for ERC20, starting from block 0",
			"address", address,
			"err", err)
		// Start from block 0 if we couldn't get the last processed block
		lastBlock = 0
	}

	s.logger.Infow("Fetching all ERC20 transfers",
		"address", address,
		"startTime", startTime,
		"endTime", endTime,
		"lastProcessedBlock", lastBlock)

	// Fetch all ERC20 transfers in a single query (empty tokenAddress means all tokens)
	transactions, err := s.etherscanAPI.GetERC20Transfers(ctx, address, "", startTime, endTime, lastBlock)
	if err != nil {
		return fmt.Errorf("fetching ERC20 transfers: %w", err)
	}

	s.logger.Infow("Fetched ERC20 transfers", "address", address, "count", len(transactions))

	// Prepare batch of transfers with preallocated capacity
	transfers := make([]*storage.Transfer, 0, len(transactions))

	// Process transactions
	for _, tx := range transactions {
		// Parse block number
		blockNumber, err := strconv.ParseInt(tx.BlockNumber, 10, 64)
		if err != nil {
			s.logger.Warnw("Failed to parse block number", "err", err, "blockNumber", tx.BlockNumber)
			continue
		}

		// Parse timestamp
		timestamp, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil {
			s.logger.Warnw("Failed to parse timestamp", "err", err, "timestamp", tx.TimeStamp)
			continue
		}

		// Create transfer record
		transfer := &storage.Transfer{
			Hash:         tx.Hash,
			BlockNumber:  blockNumber,
			Timestamp:    time.Unix(timestamp, 0),
			FromAddress:  tx.From,
			ToAddress:    tx.To,
			TokenAddress: tx.ContractAddress,
			Amount:       tx.Value,
		}

		// Add to batch
		transfers = append(transfers, transfer)
	}

	// Store transfers in batch
	if len(transfers) > 0 {
		err = s.store.AddTransfersBatch(ctx, transfers)
		if err != nil {
			s.logger.Errorw("Failed to store ERC20 transfers batch", "err", err, "count", len(transfers))
			return fmt.Errorf("storing ERC20 transfers batch: %w", err)
		}

		s.logger.Infow("Stored ERC20 transfers batch", "count", len(transfers))
	}

	return nil
}
