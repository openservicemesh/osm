package main

import (
	"net/http"

	"github.com/open-service-mesh/osm/demo/cmd/common"
)

const (
	participantName = "bookthief"
)

func main() {

	// This is the book thief.  When it tries to get books from the bookstore - it will see 404 responses!
	expectedResponseCode := http.StatusNotFound
	common.GetBooks(participantName, expectedResponseCode)
}
