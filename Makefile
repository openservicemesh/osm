#!make

TARGETS    := linux/amd64
LDFLAGS    :=
SHELL      := bash

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = $(GOPATH)/bin/gox

HAS_GOX := $(shell command -v gox)

.PHONY: gox
gox:
ifndef HAS_GOX
	 GOBIN=$(GOBIN) go get -u github.com/mitchellh/gox
endif

include .env

.PHONY: clean-cert
clean-cert:
	@rm -rf bin/cert

.PHONY: clean-ads
clean-ads:
	@rm -rf bin/ads

.PHONY: build
build: build-ads

.PHONY: build-cert
build-cert: clean-cert
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/cert ./cmd/cert

.PHONY: build-ads
build-ads: clean-ads
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/ads ./cmd/ads

.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross: build-cross-ads

.PHONY: build-cross-ads
build-cross-ads: gox
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -output="./bin/{{.OS}}-{{.Arch}}/ads" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/ads

.PHONY: build-smc
build-smc:
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/smc ./cmd/smc

.PHONY: docker-build
docker-build: build-cross docker-build-bookbuyer docker-build-bookstore docker-build-ads

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

.PHONY: docker-build-ads
docker-build-ads: build-cross-ads
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/ads -f dockerfiles/Dockerfile.ads .

.PHONY: build-bookstore
build-bookstore:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookstore ./demo/cmd/bookstore/bookstore.go

.PHONY: build-bookbuyer
build-bookbuyer:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookbuyer ./demo/cmd/bookbuyer/bookbuyer.go

.PHONY: docker-build-bookbuyer
docker-build-bookbuyer: build-bookbuyer
	docker build -t $(CTR_REGISTRY)/bookbuyer -f dockerfiles/Dockerfile.bookbuyer .

.PHONY: docker-build-bookstore
docker-build-bookstore: build-bookstore
	docker build -t $(CTR_REGISTRY)/bookstore -f dockerfiles/Dockerfile.bookstore .

.PHONY: docker-build-init
docker-build-init:
	docker build -t $(CTR_REGISTRY)/init -f dockerfiles/Dockerfile.init .

.PHONY: docker-build-envoyproxy
docker-build-envoyproxy:
	docker build -t $(CTR_REGISTRY)/envoyproxy -f dockerfiles/Dockerfile.envoyproxy .

.PHONY: docker-push-ads
docker-push-ads: docker-build-ads
	docker push "$(CTR_REGISTRY)/ads"

.PHONY: docker-push-bookbuyer
docker-push-bookbuyer: docker-build-bookbuyer
	docker push "$(CTR_REGISTRY)/bookbuyer"

.PHONY: docker-push-bookstore
docker-push-bookstore: docker-build-bookstore
	docker push "$(CTR_REGISTRY)/bookstore"

.PHONY: docker-push-init
docker-push-init: docker-build-init
	docker push "$(CTR_REGISTRY)/init"

.PHONY: docker-push-envoypoxy
docker-push-envoyproxy: docker-build-envoyproxy
	docker push "$(CTR_REGISTRY)/envoyproxy"

.PHONY: docker-push
docker-push: docker-push-init docker-push-envoyproxy docker-push-bookbuyer docker-push-bookstore docker-push-ads

.PHONY: generate-crds
generate-crds:
	@./crd/generate-AzureResource.sh

.PHONY: shellcheck
shellcheck:
	shellcheck -x $(shell find . -name '*.sh')

.PHONY: install-git-pre-push-hook
install-git-pre-push-hook:
	./scripts/install-git-pre-push-hook.sh
