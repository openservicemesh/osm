package bugreport

import (
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	rootNamespaceDirName = "namespaces"
)

var commonNamespaceCmds = [][]string{
	{"osm", "namespace", "list"},
}

func (c *Config) initRootNamespaceDir() error {
	rootNsDir := c.rootNamespaceDirPath()
	if err := os.Mkdir(rootNsDir, 0700); err != nil {
		return errors.Wrapf(err, "Error creating root dir %s for namespaces", rootNsDir)
	}
	return nil
}

func (c *Config) collectMeshedNamespaceReport() {
	for _, nsCmd := range commonNamespaceCmds {
		outPath := path.Join(c.rootNamespaceDirPath(), commandsDirName, strings.Join(nsCmd, "_"))
		if err := runCmdAndWriteToFile(nsCmd, outPath); err != nil {
			c.completionFailure("Error running command: %v", nsCmd)
		}
	}
}

func (c *Config) collectPerNamespaceReport() {
	for _, ns := range c.AppNamespaces {
		for _, nsCmd := range getPerNamespaceCommands(ns) {
			outPath := path.Join(c.rootNamespaceDirPath(), ns, commandsDirName, strings.Join(nsCmd, "_"))
			if err := runCmdAndWriteToFile(nsCmd, outPath); err != nil {
				c.completionFailure("Error running cmd: %v", nsCmd)
			}
		}
		c.completionSuccess("Collected report from Namespace %q", ns)
	}
}

func (c *Config) rootNamespaceDirPath() string {
	return path.Join(c.stagingDir, rootNamespaceDirName)
}

func getPerNamespaceCommands(namespace string) [][]string {
	return [][]string{
		{"kubectl", "get", "events", "-n", namespace},
		{"kubectl", "get", "pods", "-n", namespace, "-l", constants.EnvoyUniqueIDLabelName},
		{"kubectl", "get", "svc", "-n", namespace},
	}
}
