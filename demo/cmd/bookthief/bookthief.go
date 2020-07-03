package main

import (
	"github.com/open-service-mesh/osm/demo/cmd/common"
)

const (
	participantName    = "bookthief"
	httpStatusNotFound = "404"
)

func main() {
	// This is the book thief.  When it tries to get books from the bookstore - it will see 404 responses!
	expectedResponseCode := common.GetExpectedResponseCodeFromEnvVar(common.BookthiefExpectedResponseCodeEnvVar, httpStatusNotFound)
	common.GetBooks(participantName, expectedResponseCode)
}
