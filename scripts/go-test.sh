#!/bin/bash

go test -v $(go list ./... | grep -v /vendor/); echo $?
