#!make

SHELL:=bash

include .env

.PHONY: clean-sds
clean-sds:
	@rm -rf bin/sds

.PHONY: clean-eds
clean-eds:
	@rm -rf bin/eds

.PHONY: clean-rds
clean-rds:
	@rm -rf bin/rds

.PHONY: build
build: build-sds build-eds build-rds

.PHONY: build-sds
build-sds: clean-sds
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/sds ./cmd/sds

.PHONY: build-eds
build-eds: clean-eds
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0 go build -v -o ./bin/eds ./cmd/eds

.PHONY: build-rds
build-rds: clean-rds
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0 go build -v -o ./bin/rds ./cmd/rds

.PHONY: docker-build
docker-build: build docker-build-sds docker-build-eds docker-build-rds docker-build-bookbuyer docker-build-bookstore

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
.PHONY: docker-build-eds
docker-build-eds: build-eds
	@mkdir -p ./bin/
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/eds -f dockerfiles/Dockerfile.eds .

.PHONY: docker-build-sds
docker-build-sds: build-sds sds-root-tls
	@mkdir -p ./bin/
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/sds -f dockerfiles/Dockerfile.sds .

.PHONY: docker-build-rds
docker-build-rds: build-rds
	@mkdir -p ./bin/
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/rds -f dockerfiles/Dockerfile.rds .

.PHONY: build-counter
build-counter:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	CGO_ENABLED=0 go build -o ./demo/bin/counter ./demo/counter.go

.PHONY: docker-build-bookbuyer
docker-build-bookbuyer:
	docker build -t $(CTR_REGISTRY)/bookbuyer -f dockerfiles/Dockerfile.bookbuyer .

.PHONY: docker-build-bookstore
docker-build-bookstore: build-counter
	docker build -t $(CTR_REGISTRY)/bookstore -f dockerfiles/Dockerfile.bookstore .

.PHONY: docker-build-init
docker-build-init:
	docker build -t $(CTR_REGISTRY)/init -f dockerfiles/Dockerfile.init .

.PHONY: docker-push-eds
docker-push-eds: docker-build-eds
	docker push "$(CTR_REGISTRY)/eds"

.PHONY: docker-push-sds
docker-push-sds: docker-build-sds
	docker push "$(CTR_REGISTRY)/sds"

.PHONY: docker-push-rds
docker-push-rds: docker-build-rds
	docker push "$(CTR_REGISTRY)/rds"

.PHONY: docker-push-bookbuyer
docker-push-bookbuyer: docker-build-bookbuyer
	docker push "$(CTR_REGISTRY)/bookbuyer"

.PHONY: docker-push-bookstore
docker-push-bookstore: docker-build-bookstore
	docker push "$(CTR_REGISTRY)/bookstore"

.PHONY: docker-push-init
docker-push-init: docker-build-init
	docker push "$(CTR_REGISTRY)/init"

.PHONY: docker-push
docker-push: docker-push-eds docker-push-sds docker-push-rds docker-push-init docker-push-bookbuyer docker-push-bookstore

.PHONY: sds-root-tls
sds-root-tls:
	@mkdir -p $(shell pwd)/bin
	$(shell openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj '/CN=httpbin.example.com/O=Exmaple Company Name LTD./C=US' -keyout bin/key.pem -out bin/cert.pem)

.PHONY: generate-crds
generate-crds:
	@./crd/generate-AzureResource.sh
