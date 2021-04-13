package main

import "github.com/pkg/errors"

var errAlreadyExists = errors.Errorf("Meshes already exist in cluster. Cannot enforce single mesh cluster.")
