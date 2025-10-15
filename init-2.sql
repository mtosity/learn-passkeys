-- Migration: Add transports to credentials and create challenges table

-- Add transports column to credentials table
ALTER TABLE credentials ADD COLUMN IF NOT EXISTS transports TEXT[];

-- Create index for faster credential lookups by user
CREATE INDEX IF NOT EXISTS idx_credentials_user_id ON credentials(user_id);

-- Create challenges table for registration and authentication flows
CREATE TABLE IF NOT EXISTS challenges (
    challenge BYTEA PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- 'registration' or 'authentication'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Create index for cleanup of expired challenges
CREATE INDEX IF NOT EXISTS idx_challenges_expires_at ON challenges(expires_at);
