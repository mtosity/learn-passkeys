package models

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Username  string
	CreatedAt time.Time
}

func (u User) WebAuthnID() []byte {
	// Convert UUID to bytes
	return u.ID[:]
}

func (u User) WebAuthnName() string {
	return u.Username
}

func (u User) WebAuthnDisplayName() string {
	return u.Username
}

func (u User) WebAuthnCredentials() []webauthn.Credential {
	// For now, return empty - we'll load credentials from DB later
	return []webauthn.Credential{}
}

func (u User) WebAuthnIcon() string {
	return ""
}
