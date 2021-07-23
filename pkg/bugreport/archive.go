package bugreport

import (
	"compress/flate"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
)

func (c *Config) archive(sourcePath string, destinationPath string) error {
	z := archiver.Zip{
		CompressionLevel:       flate.DefaultCompression,
		MkdirAll:               true,
		SelectiveCompression:   true,
		ContinueOnError:        true,
		OverwriteExisting:      true,
		ImplicitTopLevelFolder: false,
	}
	if err := z.Archive([]string{sourcePath}, destinationPath); err != nil {
		c.completionFailure("Error archiving files for bug report")
		return errors.Wrap(err, "Error generating bug report")
	}

	c.completionSuccess("Bug report successfully archived to %s", destinationPath)
	return nil
}
