package handlers

import (
	"database/sql"

	"github.com/go-webauthn/webauthn/webauthn"
)

type Handler struct {
	DB       *sql.DB
	WebAuthn *webauthn.WebAuthn
}
