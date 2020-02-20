package utils

import (
	"github.com/google/uuid"
)

// NewUUIDStr creates a new string UUID.
func NewUUIDStr() string {
	id := uuid.New()
	return id.String()
}
