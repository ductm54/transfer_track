-- Create tables for transfer tracking system

-- Source addresses to track outgoing transfers from
CREATE TABLE IF NOT EXISTS source_addresses (
    id SERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    label VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Target addresses to track incoming transfers to
CREATE TABLE IF NOT EXISTS target_addresses (
    id SERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    label VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tokens to track (ERC20)
CREATE TABLE IF NOT EXISTS tokens (
    id SERIAL PRIMARY KEY,
    address VARCHAR(42) NOT NULL UNIQUE,
    symbol VARCHAR(20) NOT NULL,
    name VARCHAR(255),
    decimals INTEGER NOT NULL DEFAULT 18,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Special entry for ETH
INSERT INTO tokens (address, symbol, name, decimals)
VALUES ('0x0000000000000000000000000000000000000000', 'ETH', 'Ethereum', 18)
ON CONFLICT (address) DO NOTHING;

-- Transfers data
CREATE TABLE IF NOT EXISTS transfers (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42) NOT NULL,
    token_address VARCHAR(42) NOT NULL,
    amount NUMERIC(78, 0) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(hash, token_address, from_address, to_address)
);

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS transfers_from_to_token_idx ON transfers(from_address, to_address, token_address);
CREATE INDEX IF NOT EXISTS transfers_timestamp_idx ON transfers(timestamp);

-- System configuration
CREATE TABLE IF NOT EXISTS config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(50) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default configuration
INSERT INTO config (key, value) VALUES
('daily_refresh_time', '00:00:00'),
('min_refresh_interval_hours', '1'),
('last_eth_update', '1970-01-01 00:00:00+00'),
('last_token_update', '1970-01-01 00:00:00+00'),
('etherscan_api_key', '')
ON CONFLICT (key) DO NOTHING;