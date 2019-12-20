#!make

SHELL:=bash

.PHONY: clean-sds
clean-sds:
	rm -rf bin/sds

.PHONY: clean-eds
clean-eds:
	rm -rf bin/eds

.PHONY: build
build: build-sds build-eds

.PHONY: build-sds
build-sds: clean-sds
	mkdir -p $(shell pwd)/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ./bin/sds ./cmd/sds

.PHONY: build-eds
build-eds: clean-eds
	mkdir -p $(shell pwd)/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ./bin/eds ./cmd/eds

.PHONY: docker-build
docker-build: build docker-build-sds docker-build-eds docker-build-bookbuyer docker-build-bookstore

.PHONY: go-vet
go-vet:
	go vet ./cmd ./pkg

.PHONY: go-lint
go-lint:
	golint ./cmd ./pkg

.PHONY: go-fmt
go-fmt:
	./scripts/go-fmt.sh

.PHONY: go-test
go-test:
	./scripts/go-test.sh

