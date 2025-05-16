// Package etherscan provides a client for interacting with the Etherscan API.
package etherscan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

const (
	baseURL               = "https://api.etherscan.io/v2/api"
	moduleAccount         = "account"
	actionTxList          = "txlist"
	actionTokenTx         = "tokentx"
	defaultStartBlock     = 0
	defaultEndBlock       = 999999999
	defaultOffset         = 10000
	defaultPage           = 1
	maxRequestsPerSecond  = 5
	requestIntervalMs     = 1000 / maxRequestsPerSecond
	defaultRequestTimeout = 10 * time.Second
	defaultChainID        = 1 // Ethereum Mainnet
)

// Client represents an Etherscan API client.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	logger     *zap.SugaredLogger
	lastReq    time.Time
	chainID    int
}

// NewClient creates a new Etherscan API client.
func NewClient(apiKey string, logger *zap.SugaredLogger) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: defaultRequestTimeout},
		baseURL:    baseURL,
		logger:     logger,
		chainID:    defaultChainID,
	}
}

// NewClientWithChainID creates a new Etherscan API client with a specific chain ID.
func NewClientWithChainID(apiKey string, logger *zap.SugaredLogger, chainID int) *Client {
	client := NewClient(apiKey, logger)
	client.chainID = chainID

	return client
}

// Response represents the standard response format from Etherscan API.
type Response struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

// ETHTransaction represents an Ethereum transaction from Etherscan API.
type ETHTransaction struct {
	BlockNumber     string `json:"blockNumber"`
	TimeStamp       string `json:"timeStamp"`
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	GasPrice        string `json:"gasPrice"`
	Gas             string `json:"gas"`
	IsError         string `json:"isError"`
	TxReceiptStatus string `json:"txreceipt_status"`
}

// ERC20Transaction represents an ERC20 token transfer from Etherscan API.
type ERC20Transaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	ContractAddress   string `json:"contractAddress"`
	TransactionIndex  string `json:"transactionIndex"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           string `json:"gasUsed"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	Confirmations     string `json:"confirmations"`
}

// fetchTransactions.go contains helper functions to fetch transactions from Etherscan API.

// fetchTransactions is a helper function to fetch ETH transactions from Etherscan API.
// It handles pagination, filtering by timestamp, and rate limiting.
func (c *Client) fetchETHTransactions(
	ctx context.Context,
	params url.Values,
	startTime, endTime time.Time,
) ([]ETHTransaction, error) {
	// Preallocate with a reasonable initial capacity
	allTransactions := make([]ETHTransaction, 0, defaultOffset)
	page := defaultPage
	offset := defaultOffset

	for {
		params.Set("page", strconv.Itoa(page))
		params.Set("offset", strconv.Itoa(offset))

		var transactions []ETHTransaction
		err := c.doRequest(ctx, params, &transactions)

		if err != nil {
			return nil, err
		}

		// Filter by timestamp with preallocated capacity
		filteredTxs := make([]ETHTransaction, 0, len(transactions))

		for _, tx := range transactions {
			timestamp, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
			if err != nil {
				c.logger.Warnw("Failed to parse timestamp", "err", err, "timestamp", tx.TimeStamp)
				continue
			}

			txTime := time.Unix(timestamp, 0)
			if (txTime.After(startTime) || txTime.Equal(startTime)) && (txTime.Before(endTime) || txTime.Equal(endTime)) {
				filteredTxs = append(filteredTxs, tx)
			}
		}

		allTransactions = append(allTransactions, filteredTxs...)

		// If we got less than the requested offset, we've reached the end
		if len(transactions) < offset {
			break
		}

		page++
		c.rateLimit() // Rate limit between pagination requests
	}

	return allTransactions, nil
}

// fetchERC20Transactions is a helper function to fetch ERC20 transactions from Etherscan API.
// It handles pagination, filtering by timestamp, and rate limiting.
func (c *Client) fetchERC20Transactions(
	ctx context.Context,
	params url.Values,
	startTime, endTime time.Time,
) ([]ERC20Transaction, error) {
	// Preallocate with a reasonable initial capacity
	allTransactions := make([]ERC20Transaction, 0, defaultOffset)
	page := defaultPage
	offset := defaultOffset

	for {
		params.Set("page", strconv.Itoa(page))
		params.Set("offset", strconv.Itoa(offset))

		var transactions []ERC20Transaction
		err := c.doRequest(ctx, params, &transactions)
		if err != nil {
			return nil, err
		}

		// Filter by timestamp with preallocated capacity
		filteredTxs := make([]ERC20Transaction, 0, len(transactions))

		for _, tx := range transactions {
			timestamp, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
			if err != nil {
				c.logger.Warnw("Failed to parse timestamp", "err", err, "timestamp", tx.TimeStamp)
				continue
			}

			txTime := time.Unix(timestamp, 0)
			if (txTime.After(startTime) || txTime.Equal(startTime)) && (txTime.Before(endTime) || txTime.Equal(endTime)) {
				filteredTxs = append(filteredTxs, tx)
			}
		}

		allTransactions = append(allTransactions, filteredTxs...)

		// If we got less than the requested offset, we've reached the end
		if len(transactions) < offset {
			break
		}

		page++
		c.rateLimit() // Rate limit between pagination requests
	}

	return allTransactions, nil
}

// GetETHTransfers fetches ETH transfers for a specific address.
func (c *Client) GetETHTransfers(
	ctx context.Context, address string, startTime, endTime time.Time, startBlock int64,
) ([]ETHTransaction, error) {
	c.rateLimit()

	// If startBlock is not provided, use default
	if startBlock <= 0 {
		startBlock = defaultStartBlock
	}

	endBlock := defaultEndBlock

	c.logger.Infow("Fetching ETH transfers",
		"address", address,
		"startBlock", startBlock,
		"endBlock", endBlock,
		"startTime", startTime,
		"endTime", endTime,
		"chainID", c.chainID)

	params := url.Values{}
	params.Add("module", moduleAccount)
	params.Add("action", actionTxList)
	params.Add("address", address)
	params.Add("startblock", strconv.FormatInt(startBlock, 10))
	params.Add("endblock", strconv.Itoa(endBlock))
	params.Add("sort", "asc")
	params.Add("apikey", c.apiKey)
	params.Add("chainid", strconv.Itoa(c.chainID))

	return c.fetchETHTransactions(ctx, params, startTime, endTime)
}

// GetERC20Transfers fetches ERC20 token transfers for a specific address and token.
func (c *Client) GetERC20Transfers(ctx context.Context, address string, tokenAddress string, startTime, endTime time.Time, startBlock int64) ([]ERC20Transaction, error) {
	c.rateLimit()

	// If startBlock is not provided, use default
	if startBlock <= 0 {
		startBlock = defaultStartBlock
	}

	endBlock := defaultEndBlock

	c.logger.Infow("Fetching ERC20 transfers",
		"address", address,
		"token", tokenAddress,
		"startBlock", startBlock,
		"endBlock", endBlock,
		"startTime", startTime,
		"endTime", endTime,
		"chainID", c.chainID)

	params := url.Values{}
	params.Add("module", moduleAccount)
	params.Add("action", actionTokenTx)
	params.Add("address", address)
	params.Add("startblock", strconv.FormatInt(startBlock, 10))
	params.Add("endblock", strconv.Itoa(endBlock))
	params.Add("sort", "asc")
	params.Add("apikey", c.apiKey)
	params.Add("chainid", strconv.Itoa(c.chainID))

	// Add token address if specified
	if tokenAddress != "" {
		params.Add("contractaddress", tokenAddress)
	}

	return c.fetchERC20Transactions(ctx, params, startTime, endTime)
}

// doRequest performs an HTTP request to the Etherscan API.
func (c *Client) doRequest(ctx context.Context, params url.Values, result any) error {
	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warnw("Failed to close response body", "err", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	if response.Status != "1" {
		return fmt.Errorf("etherscan API error: %s", response.Message)
	}

	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("unmarshaling result: %w", err)
	}

	return nil
}

// rateLimit ensures we don't exceed the rate limit.
func (c *Client) rateLimit() {
	elapsed := time.Since(c.lastReq)
	if elapsed < time.Duration(requestIntervalMs)*time.Millisecond {
		time.Sleep(time.Duration(requestIntervalMs)*time.Millisecond - elapsed)
	}

	c.lastReq = time.Now()
}
