package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

// confirm displays a prompt `s` to the user and returns a bool indicating yes / no
// If the lowercased, trimmed input begins with anything other than 'y', it returns false
// It accepts an int `tries` representing the number of attempts before returning false
func confirm(stdin io.Reader, stdout io.Writer, s string, tries int) (bool, error) {
	r := bufio.NewReader(stdin)

	for ; tries > 0; tries-- {
		fmt.Fprintf(stdout, "%s [y/n]: ", s)

		res, err := r.ReadString('\n')
		if err != nil {
			return false, err
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(res)) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			continue
		}
	}

	return false, nil
}

func annotateErrorMessageWithOsmNamespace(errMsgFormat string, args ...interface{}) error {
	osmNamespaceErrorMsg := fmt.Sprintf(
		"Note: The command failed when run in the OSM namespace [%s].\n"+
			"Use the global flag --osm-namespace if [%s] is not the intended OSM namespace.",
		settings.Namespace(), settings.Namespace())

	return annotateErrorMessageWithActionableMessage(osmNamespaceErrorMsg, errMsgFormat, args...)
}

func annotateErrorMessageWithActionableMessage(actionableMessage string, errMsgFormat string, args ...interface{}) error {
	if !strings.HasSuffix(errMsgFormat, "\n") {
		errMsgFormat += "\n"
	}

	if !strings.HasSuffix(errMsgFormat, "\n\n") {
		errMsgFormat += "\n"
	}

	return errors.Errorf(errMsgFormat+actionableMessage, args...)
}
