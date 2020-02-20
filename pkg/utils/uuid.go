package utils

import (
	"github.com/google/uuid"
)

func NewUUIDStr() string {
	id := uuid.New()
	return id.String()
}
