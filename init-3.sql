-- Migration: Add credential flags for WebAuthn backup state tracking

ALTER TABLE credentials
ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS backup_state BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS attestation_type VARCHAR(50) DEFAULT '',
ADD COLUMN IF NOT EXISTS aaguid BYTEA;

-- Add index for faster lookups
CREATE INDEX IF NOT EXISTS idx_credentials_aaguid ON credentials(aaguid);
