package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/version"
)

func getLatestReleaseVersion() (string, error) {
	url := "https://api.github.com/repos/openservicemesh/osm/releases/latest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrapf(err, "unable to create GET request for latest release version from %s", url)
	}

	req.Header.Add("Accept", "application/vnd.github.v3+json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "unable to fetch latest release version from %s", url)
	}
	//nolint: errcheck
	//#nosec G307
	defer resp.Body.Close()

	latestReleaseVersionInfo := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&latestReleaseVersionInfo); err != nil {
		return "", errors.Wrapf(err, "unable to decode latest release version information from %s", url)
	}

	latestVersion, ok := latestReleaseVersionInfo["tag_name"]
	if !ok {
		return "", errors.Errorf("tag_name key not found in latest release version information from %s", url)
	}
	return fmt.Sprint(latestVersion), nil
}

func outputLatestReleaseVersion(out io.Writer, latestRelease string, currentRelease string) error {
	latest, err := version.ParseSemantic(latestRelease)
	if err != nil {
		return err
	}
	current, err := version.ParseSemantic(currentRelease)
	if err != nil {
		return err
	}
	if current.LessThan(latest) {
		fmt.Fprintf(out, "\nOSM %s is now available. Please see https://github.com/openservicemesh/osm/releases/latest.\nWARNING: upgrading could introduce breaking changes. Please review the release notes.\n\n", latestRelease)
	}
	return nil
}
