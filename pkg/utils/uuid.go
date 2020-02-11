package utils

import (
	"github.com/google/uuid"
)

func NewUuidStr() string {
	id := uuid.New()
	return id.String()
}
