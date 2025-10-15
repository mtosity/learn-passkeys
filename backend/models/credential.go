package models

import (
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	ID         []byte
	UserID     uuid.UUID
	PublicKey  []byte
	SignCount  uint32
	Transports []string
	CreatedAt  time.Time
}
