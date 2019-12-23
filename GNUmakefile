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


### docker targets
.PHONY: build-counter
build-counter:
	rm -rf $(shell pwd)/demo/bin
	mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/counter ./demo/counter.go

.PHONY: docker-build-bookbuyer
docker-build-bookbuyer:
	docker build -t delqn.azurecr.io/diplomat/bookbuyer -f dockerfiles/Dockerfile.bookbuyer .

.PHONY: docker-build-bookstore
docker-build-bookstore: build-counter
	docker build -t delqn.azurecr.io/diplomat/bookstore -f dockerfiles/Dockerfile.bookstore .

.PHONY: docker-build-init
docker-build-init:
	docker build -t delqn.azurecr.io/diplomat/init -f dockerfiles/Dockerfile.init .

.PHONY: docker-push-eds
docker-push-eds: docker-build-eds
	docker push delqn.azurecr.io/diplomat/eds

.PHONY: docker-push-bookbuyer
docker-push-bookbuyer: docker-build-bookbuyer
	docker push delqn.azurecr.io/diplomat/bookbuyer

.PHONY: docker-push-bookstore
docker-push-bookstore: docker-build-bookstore
	docker push delqn.azurecr.io/diplomat/bookstore

.PHONY: docker-push-init
docker-push-init: docker-build-init
	docker push delqn.azurecr.io/diplomat/init

.PHONY: docker-push
docker-push: docker-push-eds docker-push-sds docker-push-client docker-push-init docker-push-bookbuyer docker-push-bookstore
