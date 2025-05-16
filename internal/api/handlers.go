// Package api provides HTTP API handlers for the transfer tracking service.
package api

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/ductm54/transfer-track/internal/service"
	"github.com/ductm54/transfer-track/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler handles API requests.
type Handler struct {
	transferService *service.TransferService
	store           *storage.Storage
	logger          *zap.SugaredLogger
}

// NewHandler creates a new Handler.
func NewHandler(transferService *service.TransferService, store *storage.Storage, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		transferService: transferService,
		store:           store,
		logger:          logger,
	}
}

// RegisterRoutes registers API routes.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Transfer endpoints
		api.GET("/transfers", h.GetTotalAmounts)
		api.POST("/transfers/refresh", h.RefreshTransfers)

		// Source address endpoints
		api.GET("/source-addresses", h.GetSourceAddresses)
		api.POST("/source-addresses", h.AddSourceAddress)
		api.DELETE("/source-addresses/:id", h.DeleteSourceAddress)

		// Target address endpoints
		api.GET("/target-addresses", h.GetTargetAddresses)
		api.POST("/target-addresses", h.AddTargetAddress)
		api.DELETE("/target-addresses/:id", h.DeleteTargetAddress)

		// Token endpoints
		api.GET("/tokens", h.GetTokens)
		api.POST("/tokens", h.AddToken)
		api.DELETE("/tokens/:id", h.DeleteToken)

		// Config endpoints
		api.GET("/config", h.GetConfig)
		api.PUT("/config/refresh-interval", h.UpdateRefreshInterval)
		api.PUT("/config/daily-refresh-time", h.UpdateDailyRefreshTime)
	}
}

// parseTimeParam parses a time parameter from a string.
// It supports both Unix timestamp and RFC3339 formats.
// If the string is empty, it returns the defaultTime.
func parseTimeParam(timeStr string, defaultTime time.Time) (time.Time, error) {
	if timeStr == "" {
		return defaultTime, nil
	}

	// Try to parse as epoch timestamp (unix seconds)
	epoch, err := strconv.ParseInt(timeStr, 10, 64)

	if err != nil {
		return time.Time{}, fmt.Errorf("parsing timestamp as integer: %w", err)
	}

	return time.Unix(epoch, 0), nil
}

// normalizeAmount converts a string amount to a normalized amount based on decimals.
// It returns the normalized amount as a string.
func normalizeAmount(amount string, decimals int) string {
	// Create a big.Float from the string amount
	bigAmount, success := new(big.Float).SetString(amount)
	if !success {
		return "0"
	}

	// Calculate divisor (10^decimals)
	divisor := new(big.Float).SetInt(new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(int64(decimals)),
		nil,
	))

	// Divide amount by divisor
	normalizedAmount := new(big.Float).Quo(bigAmount, divisor)

	// Convert to string with appropriate precision
	return normalizedAmount.Text('f', decimals)
}

// refreshDataIfNeeded checks if data should be refreshed and refreshes it if needed.
func (h *Handler) refreshDataIfNeeded(ctx context.Context) {
	shouldRefresh, err := h.transferService.ShouldRefreshData(ctx)
	if err != nil {
		h.logger.Warnw("Error checking if data should be refreshed", "err", err)
		return
	}

	if !shouldRefresh {
		return
	}

	h.logger.Infow("Auto-refreshing data before getting total amounts")

	// Create a new context with timeout for the refresh operation
	refreshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = h.transferService.FetchAndStoreTransfers(refreshCtx)
	if err != nil {
		h.logger.Errorw("Error refreshing data", "err", err)
		// Continue with potentially stale data
	}
}

// GetTotalAmounts handles the request to get total amounts.
func (h *Handler) GetTotalAmounts(c *gin.Context) {
	// Parse time range parameters
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	// Default to last 30 days if not specified
	defaultStartTime := time.Now().AddDate(0, -1, 0) // 1 month ago
	defaultEndTime := time.Now()

	startTime, err := parseTimeParam(startTimeStr, defaultStartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid start_time format, expected Unix timestamp (seconds since epoch) or RFC3339",
		})
		return
	}

	endTime, err := parseTimeParam(endTimeStr, defaultEndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid end_time format, expected Unix timestamp (seconds since epoch) or RFC3339",
		})
		return
	}

	// Refresh data if needed
	h.refreshDataIfNeeded(c)

	// Get total amounts
	amounts, err := h.store.GetTotalAmounts(c, startTime, endTime)
	if err != nil {
		h.logger.Errorw("Error getting total amounts", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total amounts"})
		return
	}

	// Calculate normalized amounts
	for i := range amounts {
		amounts[i].NormalizedAmount = normalizeAmount(amounts[i].TotalAmount, amounts[i].Decimals)
	}

	// Create response with timestamps
	response := gin.H{
		"start_time": startTime.Unix(),
		"end_time":   endTime.Unix(),
		"amounts":    amounts,
	}

	c.JSON(http.StatusOK, response)
}

// RefreshTransfers handles the request to refresh transfers.
func (h *Handler) RefreshTransfers(c *gin.Context) {
	err := h.transferService.FetchAndStoreTransfers(c)
	if err != nil {
		h.logger.Errorw("Error refreshing transfers", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh transfers"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transfers refreshed successfully"})
}

// GetSourceAddresses handles the request to get source addresses.
func (h *Handler) GetSourceAddresses(c *gin.Context) {
	addresses, err := h.store.GetSourceAddresses(c)
	if err != nil {
		h.logger.Errorw("Error getting source addresses", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get source addresses"})

		return
	}

	c.JSON(http.StatusOK, addresses)
}

// AddAddressRequest represents a request to add a single address.
type AddAddressRequest struct {
	Address string `json:"address" binding:"required"`
	Label   string `json:"label"`
}

// AddAddressesRequest represents a request to add multiple addresses.
type AddAddressesRequest struct {
	Addresses []struct {
		Address string `json:"address" binding:"required"`
		Label   string `json:"label"`
	} `json:"addresses" binding:"required"`
}

// addAddresses is a generic function to add addresses (source or target).
// It takes a function to add a single address and returns the added addresses.
func (h *Handler) addAddresses(
	c *gin.Context,
	addFunc any,
	addressType string,
) {
	// Create a wrapper function that converts the specific return type to any
	var addFuncWrapper func(ctx context.Context, address, label string) (any, error)

	// Type switch to handle different function signatures
	switch typedAddFunc := addFunc.(type) {
	case func(ctx context.Context, address, label string) (*storage.SourceAddress, error):
		addFuncWrapper = func(ctx context.Context, address, label string) (any, error) {
			return typedAddFunc(ctx, address, label)
		}
	case func(ctx context.Context, address, label string) (*storage.TargetAddress, error):
		addFuncWrapper = func(ctx context.Context, address, label string) (any, error) {
			return typedAddFunc(ctx, address, label)
		}
	default:
		h.logger.Errorw("Invalid function type passed to addAddresses", "type", fmt.Sprintf("%T", addFunc))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Try to bind as array first
	var reqMulti AddAddressesRequest
	if err := c.ShouldBindJSON(&reqMulti); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Preallocate with the capacity of the number of addresses
	addedAddresses := make([]any, 0, len(reqMulti.Addresses))

	for _, addr := range reqMulti.Addresses {
		address, err := addFuncWrapper(c, addr.Address, addr.Label)
		if err != nil {
			h.logger.Warnw(fmt.Sprintf("Error adding %s address", addressType),
				"address", addr.Address, "err", err)
			// Continue with other addresses
			continue
		}

		addedAddresses = append(addedAddresses, address)
	}

	if len(addedAddresses) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to add any %s addresses", addressType),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"addresses": addedAddresses})
}

// AddSourceAddress handles the request to add a source address or multiple source addresses.
func (h *Handler) AddSourceAddress(c *gin.Context) {
	h.addAddresses(c, h.store.AddSourceAddress, "source")
}

// deleteAddress is a generic function to delete an address (source or target).
// It takes a function to delete a single address by ID.
func (h *Handler) deleteAddress(
	c *gin.Context,
	deleteFunc func(ctx context.Context, id int64) error,
	addressType string,
) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	err = deleteFunc(c, id)
	if err != nil {
		h.logger.Errorw(fmt.Sprintf("Error deleting %s address", addressType),
			"err", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete %s address", addressType),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%s address deleted successfully", addressType),
	})
}

// DeleteSourceAddress handles the request to delete a source address.
func (h *Handler) DeleteSourceAddress(c *gin.Context) {
	h.deleteAddress(c, h.store.DeleteSourceAddress, "Source")
}

// GetTargetAddresses handles the request to get target addresses.
func (h *Handler) GetTargetAddresses(c *gin.Context) {
	addresses, err := h.store.GetTargetAddresses(c)
	if err != nil {
		h.logger.Errorw("Error getting target addresses", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get target addresses"})
		return
	}

	c.JSON(http.StatusOK, addresses)
}

// AddTargetAddress handles the request to add a target address or multiple target addresses.
func (h *Handler) AddTargetAddress(c *gin.Context) {
	h.addAddresses(c, h.store.AddTargetAddress, "target")
}

// DeleteTargetAddress handles the request to delete a target address.
func (h *Handler) DeleteTargetAddress(c *gin.Context) {
	h.deleteAddress(c, h.store.DeleteTargetAddress, "Target")
}

// AddTokenRequest represents a request to add a token.
type AddTokenRequest struct {
	Address  string `json:"address" binding:"required"`
	Symbol   string `json:"symbol" binding:"required"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
}

// GetTokens handles the request to get tokens.
func (h *Handler) GetTokens(c *gin.Context) {
	tokens, err := h.store.GetTokens(c)
	if err != nil {
		h.logger.Errorw("Error getting tokens", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tokens"})

		return
	}

	c.JSON(http.StatusOK, tokens)
}

// AddToken handles the request to add a token.
func (h *Handler) AddToken(c *gin.Context) {
	var req AddTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	// Default decimals to 18 if not specified
	if req.Decimals == 0 {
		req.Decimals = 18
	}

	token, err := h.store.AddToken(c, req.Address, req.Symbol, req.Name, req.Decimals)
	if err != nil {
		h.logger.Errorw("Error adding token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add token"})

		return
	}

	c.JSON(http.StatusCreated, token)
}

// DeleteToken handles the request to delete a token.
func (h *Handler) DeleteToken(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})

		return
	}

	err = h.store.DeleteToken(c, id)
	if err != nil {
		h.logger.Errorw("Error deleting token", "err", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token deleted successfully"})
}

// GetConfig handles the request to get configuration.
func (h *Handler) GetConfig(c *gin.Context) {
	// Get refresh interval
	refreshInterval, err := h.transferService.GetRefreshInterval(c)
	if err != nil {
		h.logger.Warnw("Error getting refresh interval", "err", err)

		refreshInterval = 1 // Default to 1 hour
	}

	// Get daily refresh time
	dailyRefreshTime, err := h.transferService.GetDailyRefreshTime(c)
	if err != nil {
		h.logger.Warnw("Error getting daily refresh time", "err", err)

		dailyRefreshTime = "00:00:00" // Default to midnight
	}

	config := gin.H{
		"min_refresh_interval_hours": refreshInterval,
		"daily_refresh_time":         dailyRefreshTime,
	}

	c.JSON(http.StatusOK, config)
}

// UpdateRefreshIntervalRequest represents a request to update the refresh interval.
type UpdateRefreshIntervalRequest struct {
	Hours int `json:"hours" binding:"required"`
}

// UpdateRefreshInterval handles the request to update the refresh interval.
func (h *Handler) UpdateRefreshInterval(c *gin.Context) {
	var req UpdateRefreshIntervalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	err := h.transferService.UpdateRefreshInterval(c, req.Hours)
	if err != nil {
		h.logger.Errorw("Error updating refresh interval", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update refresh interval"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Refresh interval updated successfully"})
}

// UpdateDailyRefreshTimeRequest represents a request to update the daily refresh time.
type UpdateDailyRefreshTimeRequest struct {
	Time string `json:"time" binding:"required"`
}

// UpdateDailyRefreshTime handles the request to update the daily refresh time.
func (h *Handler) UpdateDailyRefreshTime(c *gin.Context) {
	var req UpdateDailyRefreshTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	err := h.transferService.UpdateDailyRefreshTime(c, req.Time)
	if err != nil {
		h.logger.Errorw("Error updating daily refresh time", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update daily refresh time"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Daily refresh time updated successfully"})
}
