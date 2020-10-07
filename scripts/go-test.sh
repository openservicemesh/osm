#!/bin/bash

go test -failfast -race -v ./...; echo $?
