package main

import (
	"net/http"

	"github.com/open-service-mesh/osm/demo/cmd/common"
)

const (
	participantName = "bookbuyer"
)

func main() {

	// This is the bookbuyer.  When it tries to buy books from the bookstore - we expect it to see 200 responses.
	expectedResponseCode := http.StatusOK
	common.GetBooks(participantName, expectedResponseCode)
}
