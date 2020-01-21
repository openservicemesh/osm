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

.PHONY: clean-cds
clean-cds:
	@rm -rf bin/cds

.PHONY: clean-lds
clean-lds:
	@rm -rf bin/lds

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
build: build-sds build-eds build-cds build-rds build-lds

.PHONY: build-cds
build-cds: clean-cds
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/cds ./cmd/cds

.PHONY: build-lds
build-lds: clean-lds
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/lds ./cmd/lds

.PHONY: build-smc
build-smc:
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0  go build -v -o ./bin/smc ./cmd/smc

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

.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross: build-cross-eds build-cross-sds build-cross-cds build-cross-rds build-cross-lds

.PHONY: build-cross-cds
build-cross-cds: gox
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -output="./bin/{{.OS}}-{{.Arch}}/cds" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/cds

.PHONY: build-cross-lds
build-cross-lds: gox
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -output="./bin/{{.OS}}-{{.Arch}}/lds" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/lds

.PHONY: build-cross-eds
build-cross-eds: gox
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -output="./bin/{{.OS}}-{{.Arch}}/eds" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/eds

.PHONY: build-cross-sds
build-cross-sds: gox
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -output="./bin/{{.OS}}-{{.Arch}}/sds" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/sds

.PHONY: build-cross-rds
build-cross-rds: gox
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -output="./bin/{{.OS}}-{{.Arch}}/rds" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/rds

.PHONY: docker-build
docker-build: build-cross docker-build-sds docker-build-eds docker-build-bookbuyer docker-build-bookstore docker-build-cds docker-build-rds docker-build-lds

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
.PHONY: docker-build-cds
docker-build-cds: build-cross-cds
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/cds -f dockerfiles/Dockerfile.cds .

.PHONY: docker-build-lds
docker-build-lds: build-cross-lds
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/lds -f dockerfiles/Dockerfile.lds .

.PHONY: docker-build-eds
docker-build-eds: build-cross-eds
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/eds -f dockerfiles/Dockerfile.eds .

.PHONY: docker-build-sds
docker-build-sds: build-cross-sds
	@mkdir -p ./bin/
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/sds -f dockerfiles/Dockerfile.sds .

.PHONY: docker-build-rds
docker-build-rds: build-cross-rds
	@mkdir -p ./bin/
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/rds -f dockerfiles/Dockerfile.rds .

.PHONY: build-counter
build-counter:
	@rm -rf $(shell pwd)/demo/bin
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/counter ./demo/counter.go

.PHONY: docker-build-bookbuyer
docker-build-bookbuyer:
	docker build -t $(CTR_REGISTRY)/bookbuyer -f dockerfiles/Dockerfile.bookbuyer .

.PHONY: docker-build-bookstore
docker-build-bookstore: build-counter
	docker build -t $(CTR_REGISTRY)/bookstore -f dockerfiles/Dockerfile.bookstore .

.PHONY: docker-build-init
docker-build-init:
	docker build -t $(CTR_REGISTRY)/init -f dockerfiles/Dockerfile.init .

.PHONY: docker-push-cds
docker-push-cds: docker-build-cds
	docker push "$(CTR_REGISTRY)/cds"

.PHONY: docker-push-lds
docker-push-lds: docker-build-lds
	docker push "$(CTR_REGISTRY)/lds"

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
docker-push: docker-push-eds docker-push-sds docker-push-init docker-push-bookbuyer docker-push-bookstore docker-push-cds docker-push-rds docker-push-lds

.PHONY: sds-root-tls
sds-root-tls:
	@mkdir -p $(shell pwd)/bin
	@./scripts/gen-proxy-certificate.sh

.PHONY: generate-crds
generate-crds:
	@./crd/generate-AzureResource.sh

.PHONY: shellcheck
shellcheck:
	shellcheck -x $(shell find . -name '*.sh')
