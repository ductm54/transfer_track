# Transfer Track

A system that tracks outgoing ETH and selected ERC20 token transfers from a set of addresses to a specific list of addresses.

## Features

- Track outgoing ETH transfers from specified source addresses to target addresses
- Track outgoing ERC20 token transfers from specified source addresses to target addresses
- API to check total amounts of each token and ETH sent over a selected time range
- Automatic data refresh when using the API if the last update was more than the configured minimum refresh interval (default: 1 hour)
- Configurable daily data refresh time (default: midnight)
- Configurable source addresses, target addresses, and tokens to track

## Requirements

- Go 1.16+
- PostgreSQL
- Etherscan API key

## Setup

1. Clone the repository
2. Copy `.env.sample` to `.env` and update the values
3. Create a PostgreSQL database
4. Run the migrations: `go run cmd/transfer-track/main.go`

## Configuration

The system can be configured through the API:

- Source addresses: Addresses to track outgoing transfers from
- Target addresses: Addresses to track incoming transfers to
- Tokens: ERC20 tokens to track (ETH is tracked by default)
- Minimum refresh interval: Minimum time interval between data refreshes when using the API
- Daily refresh time: When the daily data refresh should run

## API Endpoints

### Transfers

- `GET /api/transfers`: Get total amounts of each token transferred
  - Query parameters:
    - `start_time`: Start time as Unix epoch timestamp in seconds or RFC3339 format (default: 30 days ago)
    - `end_time`: End time as Unix epoch timestamp in seconds or RFC3339 format (default: now)
  - Response includes:
    - `start_time`: Start time as Unix epoch timestamp in seconds
    - `end_time`: End time as Unix epoch timestamp in seconds
    - `amounts`: Array of token amounts with both raw and normalized values:
      - `total_amount`: Raw amount in wei/smallest token unit
      - `normalized_amount`: Human-readable amount (total_amount / 10^decimals)
- `POST /api/transfers/refresh`: Manually trigger a data refresh

### Source Addresses

- `GET /api/source-addresses`: Get all source addresses
- `POST /api/source-addresses`: Add multiple source addresses
  - Request body: `{ "addresses": [{ "address": "0x...", "label": "Address 1" }, { "address": "0x...", "label": "Address 2" }] }`
- `DELETE /api/source-addresses/:id`: Delete a source address

### Target Addresses

- `GET /api/target-addresses`: Get all target addresses
- `POST /api/target-addresses`: Add multiple target addresses
  - Request body: `{ "addresses": [{ "address": "0x...", "label": "Address 1" }, { "address": "0x...", "label": "Address 2" }] }`
- `DELETE /api/target-addresses/:id`: Delete a target address

### Tokens

- `GET /api/tokens`: Get all tokens
- `POST /api/tokens`: Add a new token
  - Request body: `{ "address": "0x...", "symbol": "TOKEN", "name": "Token Name", "decimals": 18 }`
- `DELETE /api/tokens/:id`: Delete a token

### Configuration

- `GET /api/config`: Get current configuration
- `PUT /config/refresh-interval`: Update minimum refresh interval (in hours)
  - Request body: `{ "hours": 1 }`
- `PUT /config/daily-refresh-time`: Update daily refresh time
  - Request body: `{ "time": "00:00:00" }`

Note: The Etherscan API key can only be set via the environment variable `ETHERSCAN_API_KEY`. The system uses Etherscan API with chain ID support (default: 1 for Ethereum Mainnet).

## Running the Service

```bash
go run cmd/transfer-track/main.go
```

You can specify the chain ID using the `--chain-id` flag or the `CHAIN_ID` environment variable. The default chain ID is 1 (Ethereum Mainnet).

```bash
go run cmd/transfer-track/main.go --chain-id=1
```

## Docker

You can also run the service using Docker:

```bash
docker-compose up -d
```


## License

This project is licensed under the MIT License - see the LICENSE file for details.
