package main

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

// TestCommandEntryPoint tests that all CLI subcommands can be
// added under the root CLI command.
func TestCommandEntryPoint(t *testing.T) {
	assert := tassert.New(t)

	cmd := initCommands()
	assert.NotNil(cmd)
}
