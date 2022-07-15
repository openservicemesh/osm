package bugreport

import (
	"fmt"
	"os"

	"github.com/mholt/archiver/v3"
)

func (c *Config) archive(sourcePath string, destinationPath string) error {
	if _, err := os.Stat(destinationPath); err == nil {
		if err := os.Remove(destinationPath); err != nil {
			c.completionFailure("Error removing existing bug report file %s", destinationPath)
			return fmt.Errorf("Error generating bug report: %w", err)
		}
	}
	if err := archiver.Archive([]string{sourcePath}, destinationPath); err != nil {
		c.completionFailure("Error archiving files for bug report")
		return fmt.Errorf("Error generating bug report: %w", err)
	}

	c.completionSuccess("Bug report successfully archived to %s", destinationPath)
	return nil
}
