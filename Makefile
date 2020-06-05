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

.PHONY: build-osm
build-osm:
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/osm -ldflags "-X main.chartTGZSource=$$(go run scripts/generate_chart/generate_chart.go)" ./cmd/osm

.PHONY: docker-build
docker-build: build-cross docker-build-bookbuyer docker-build-bookstore docker-build-ads docker-build-bookwarehouse

.PHONY: go-vet
go-vet:
	go vet ./cmd ./pkg

.PHONY: go-lint
go-lint:
	golint ./cmd ./pkg
	golangci-lint run --tests --enable-all # --disable gochecknoglobals --disable gochecknoinit

.PHONY: go-fmt
go-fmt:
	./scripts/go-fmt.sh

.PHONY: go-test
go-test:
	./scripts/go-test.sh

.PHONY: go-test-coverage
go-test-coverage:
	./scripts/test-w-coverage.sh

.PHONY: docker-build-ads
docker-build-ads: build-cross-ads
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/ads:$(CTR_TAG) -f dockerfiles/Dockerfile.ads .

.PHONY: build-bookstore
build-bookstore:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookstore ./demo/cmd/bookstore/bookstore.go

.PHONY: build-bookwarehouse
build-bookwarehouse:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookwarehouse ./demo/cmd/bookwarehouse/bookwarehouse.go

.PHONY: build-bookbuyer
build-bookbuyer:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookbuyer ./demo/cmd/bookbuyer/bookbuyer.go

.PHONY: build-bookthief
build-bookthief:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookthief ./demo/cmd/bookthief/bookthief.go

.PHONY: docker-build-bookbuyer
docker-build-bookbuyer: build-bookbuyer
	docker build -t $(CTR_REGISTRY)/bookbuyer:$(CTR_TAG) -f dockerfiles/Dockerfile.bookbuyer .

.PHONY: docker-build-bookthief
docker-build-bookthief: build-bookthief
	docker build -t $(CTR_REGISTRY)/bookthief:$(CTR_TAG) -f dockerfiles/Dockerfile.bookthief .

.PHONY: docker-build-bookstore
docker-build-bookstore: build-bookstore
	docker build -t $(CTR_REGISTRY)/bookstore:$(CTR_TAG) -f dockerfiles/Dockerfile.bookstore .

.PHONY: docker-build-bookwarehouse
docker-build-bookwarehouse: build-bookwarehouse
	docker build -t $(CTR_REGISTRY)/bookwarehouse:$(CTR_TAG) -f dockerfiles/Dockerfile.bookwarehouse .

.PHONY: docker-build-init
docker-build-init:
	docker build -t $(CTR_REGISTRY)/init:$(CTR_TAG) -f dockerfiles/Dockerfile.init .

.PHONY: docker-push-ads
docker-push-ads: docker-build-ads
	docker push "$(CTR_REGISTRY)/ads:$(CTR_TAG)"

.PHONY: docker-push-bookbuyer
docker-push-bookbuyer: docker-build-bookbuyer
	docker push "$(CTR_REGISTRY)/bookbuyer:$(CTR_TAG)"

.PHONY: docker-push-bookthief
docker-push-bookthief: docker-build-bookthief
	docker push "$(CTR_REGISTRY)/bookthief:$(CTR_TAG)"

.PHONY: docker-push-bookstore
docker-push-bookstore: docker-build-bookstore
	docker push "$(CTR_REGISTRY)/bookstore:$(CTR_TAG)"

.PHONY: docker-push-bookwarehouse
docker-push-bookwarehouse: docker-build-bookwarehouse
	docker push "$(CTR_REGISTRY)/bookwarehouse:$(CTR_TAG)"

.PHONY: docker-push-init
docker-push-init: docker-build-init
	docker push "$(CTR_REGISTRY)/init:$(CTR_TAG)"

.PHONY: docker-push
docker-push: docker-push-init docker-push-bookbuyer docker-push-bookthief docker-push-bookstore docker-push-ads docker-push-bookwarehouse

.PHONY: generate-crds
generate-crds:
	@./crd/generate-AzureResource.sh

.PHONY: shellcheck
shellcheck:
	shellcheck -x $(shell find . -name '*.sh')

.PHONY: install-git-pre-push-hook
install-git-pre-push-hook:
	./scripts/install-git-pre-push-hook.sh
