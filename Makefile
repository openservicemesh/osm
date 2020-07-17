#!make

TARGETS    := darwin/amd64 linux/amd64
LDFLAGS    :=
SHELL      := bash -o pipefail
BINNAME    ?= osm
DIST_DIRS  := find * -type d -exec

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = $(GOPATH)/bin/gox

.PHONY: gox
gox:
ifndef HAS_GOX
	 GOBIN=$(GOBIN) go get -u github.com/mitchellh/gox
endif

check-env:
ifndef CTR_REGISTRY
	$(error CTR_REGISTRY environment variable is not defined; see the .env.example file for more information; then source .env)
endif
ifndef CTR_TAG
	$(error CTR_TAG environment variable is not defined; see the .env.example file for more information; then source .env)
endif

.PHONY: clean-cert
clean-cert:
	@rm -rf bin/cert

.PHONY: clean-osm-controller
clean-osm-controller:
	@rm -rf bin/osm-controller

.PHONY: build
build: build-osm-controller

.PHONY: build-osm-controller
build-osm-controller: clean-osm-controller
	@mkdir -p $(shell pwd)/bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-controller ./cmd/ads

.PHONY: build-osm
build-osm:
	@mkdir -p $(shell pwd)/bin
	go run scripts/generate_chart/generate_chart.go | CGO_ENABLED=0  go build -v -o ./bin/osm -ldflags "-X main.chartTGZSource=$$(cat -)" ./cmd/cli

.PHONY: docker-build
docker-build: docker-build-osm-controller docker-build-bookbuyer docker-build-bookstore docker-build-bookwarehouse

.PHONY: go-vet
go-vet:
	go vet ./...

.PHONY: go-lint
go-lint:
	golint ./cmd ./pkg
	golangci-lint run --tests --enable-all # --disable gochecknoglobals --disable gochecknoinit

.PHONY: go-fmt
go-fmt:
	go fmt ./...

.PHONY: go-test
go-test:
	./scripts/go-test.sh

.PHONY: go-test-coverage
go-test-coverage:
	./scripts/test-w-coverage.sh

.PHONY: docker-build-osm-controller
docker-build-osm-controller: check-env build-osm-controller
	docker build --build-arg $(HOME)/go/ -t $(CTR_REGISTRY)/osm-controller:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-controller .

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
docker-build-bookbuyer: check-env build-bookbuyer
	docker build -t $(CTR_REGISTRY)/bookbuyer:$(CTR_TAG) -f dockerfiles/Dockerfile.bookbuyer .

.PHONY: docker-build-bookthief
docker-build-bookthief: check-env build-bookthief
	docker build -t $(CTR_REGISTRY)/bookthief:$(CTR_TAG) -f dockerfiles/Dockerfile.bookthief .

.PHONY: docker-build-bookstore
docker-build-bookstore: check-env build-bookstore
	docker build -t $(CTR_REGISTRY)/bookstore:$(CTR_TAG) -f dockerfiles/Dockerfile.bookstore .

.PHONY: docker-build-bookwarehouse
docker-build-bookwarehouse: check-env build-bookwarehouse
	docker build -t $(CTR_REGISTRY)/bookwarehouse:$(CTR_TAG) -f dockerfiles/Dockerfile.bookwarehouse .

.PHONY: docker-build-init
docker-build-init: check-env
	docker build -t $(CTR_REGISTRY)/init:$(CTR_TAG) -f dockerfiles/Dockerfile.init .

.PHONY: docker-push-osm-controller
docker-push-osm-controller: check-env docker-build-osm-controller
	docker push "$(CTR_REGISTRY)/osm-controller:$(CTR_TAG)"

.PHONY: docker-push-bookbuyer
docker-push-bookbuyer: check-env docker-build-bookbuyer
	docker push "$(CTR_REGISTRY)/bookbuyer:$(CTR_TAG)"

.PHONY: docker-push-bookthief
docker-push-bookthief: check-env docker-build-bookthief
	docker push "$(CTR_REGISTRY)/bookthief:$(CTR_TAG)"

.PHONY: docker-push-bookstore
docker-push-bookstore: check-env docker-build-bookstore
	docker push "$(CTR_REGISTRY)/bookstore:$(CTR_TAG)"

.PHONY: docker-push-bookwarehouse
docker-push-bookwarehouse: check-env docker-build-bookwarehouse
	docker push "$(CTR_REGISTRY)/bookwarehouse:$(CTR_TAG)"

.PHONY: docker-push-init
docker-push-init: check-env docker-build-init
	docker push "$(CTR_REGISTRY)/init:$(CTR_TAG)"

.PHONY: docker-push
docker-push: docker-push-init docker-push-bookbuyer docker-push-bookthief docker-push-bookstore docker-push-osm-controller docker-push-bookwarehouse

.PHONY: generate-crds
generate-crds:
	@./crd/generate-AzureResource.sh

.PHONY: shellcheck
shellcheck:
	shellcheck -x $(shell find . -name '*.sh')

.PHONY: install-git-pre-push-hook
install-git-pre-push-hook:
	./scripts/install-git-pre-push-hook.sh

# -------------------------------------------
#  release targets below
# -------------------------------------------

.PHONY: build-cross
build-cross: $(GOX)
	@mkdir -p $(shell pwd)/_dist
	go run scripts/generate_chart/generate_chart.go | GO111MODULE=on CGO_ENABLED=0 $(GOX) -ldflags "-X main.chartTGZSource=$$(cat -)" -parallel=3 -output="_dist/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' ./cmd/cli

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf osm-${VERSION}-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r osm-${VERSION}-{}.zip {} \; \
	)

.PHONY: release-artifacts
release-artifacts: build-cross dist
