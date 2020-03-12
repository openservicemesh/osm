package utils

import (
	"regexp"

	"github.com/google/uuid"
)

// NewUUIDStr creates a new string UUID.
func NewUUIDStr() string {
	id := uuid.New()
	return id.String()
}

// IsValidUUID validates a UUID
func IsValidUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}
