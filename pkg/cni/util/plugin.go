// Package util provides generic utility routines used within OSM CNI Plugin.
package util

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fsnotify/fsnotify"
)

// CreateFileWatcher creates a file watcher that watches for any changes to the directory
func CreateFileWatcher(dirs ...string) (watcher *fsnotify.Watcher, fileModified chan bool, errChan chan error, err error) {
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}

	fileModified, errChan = make(chan bool), make(chan error)
	go watchFiles(watcher, fileModified, errChan)

	for _, dir := range dirs {
		if IsDirWriteable(dir) != nil {
			continue
		}
		if err = watcher.Add(dir); err != nil {
			if closeErr := watcher.Close(); closeErr != nil {
				err = fmt.Errorf("%s: %w", closeErr.Error(), err)
			}
			return nil, nil, nil, err
		}
	}

	return
}

func watchFiles(watcher *fsnotify.Watcher, fileModified chan bool, errChan chan error) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove) != 0 {
				fileModified <- true
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			errChan <- err
		}
	}
}

// WaitForFileMod waits until a file is modified (returns nil), the context is cancelled (returns context error), or returns error
func WaitForFileMod(ctx context.Context, fileModified chan bool, errChan chan error) error {
	select {
	case <-fileModified:
		return nil
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetPlugins given an unmarshalled CNI config JSON map, return the plugin list asserted as a []interface{}
func GetPlugins(cniConfigMap map[string]any) (plugins []any, err error) {
	plugins, ok := cniConfigMap["plugins"].([]any)
	if !ok {
		err = fmt.Errorf("error reading plugin list from CNI config")
		return
	}
	return
}

// GetPlugin given the raw plugin interface, return the plugin asserted as a map[string]interface{}
func GetPlugin(rawPlugin any) (plugin map[string]any, err error) {
	plugin, ok := rawPlugin.(map[string]any)
	if !ok {
		err = fmt.Errorf("error reading plugin from CNI config plugin list")
		return
	}
	return
}

// MarshalCNIConfig marshal the CNI config map and append a new line
func MarshalCNIConfig(cniConfigMap map[string]any) ([]byte, error) {
	cniConfig, err := json.MarshalIndent(cniConfigMap, "", "  ")
	if err != nil {
		return nil, err
	}
	cniConfig = append(cniConfig, "\n"...)
	return cniConfig, nil
}
