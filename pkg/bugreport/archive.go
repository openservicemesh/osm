package bugreport

import (
	"os"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
)

func (c *Config) archive(sourcePath string, destinationPath string) error {
	if _, err := os.Stat(destinationPath); err == nil {
		if err := os.Remove(destinationPath); err != nil {
			c.completionFailure("Error removing existing bug report file %s", destinationPath)
			return errors.Wrap(err, "Error generating bug report")
		}
	}
	if err := archiver.Archive([]string{sourcePath}, destinationPath); err != nil {
		c.completionFailure("Error archiving files for bug report")
		return errors.Wrap(err, "Error generating bug report")
	}

	c.completionSuccess("Bug report successfully archived to %s", destinationPath)
	return nil
}
