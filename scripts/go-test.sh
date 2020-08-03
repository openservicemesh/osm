#!/bin/bash

go test -race -v ./...; echo $?
