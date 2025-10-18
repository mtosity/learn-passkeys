package models

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Username    string
	CreatedAt   time.Time
	Credentials []webauthn.Credential //additional fields for application logic
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
	return u.Credentials
}

func (u User) WebAuthnIcon() string {
	return ""
}
