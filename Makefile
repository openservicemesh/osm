#!make

TARGETS         := darwin/amd64 linux/amd64
SHELL           := bash -o pipefail
BINNAME         ?= osm
DIST_DIRS       := find * -type d -exec
CTR_REGISTRY    ?= openservicemesh
CTR_TAG         ?= latest

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = $(GOPATH)/bin/gox

HAS_GOX := $(shell command -v gox)

CLI_VERSION ?= dev
BUILD_DATE=$$(date +%Y-%m-%d-%H:%M)
GIT_SHA=$$(git rev-parse --short HEAD)
BUILD_DATE_VAR := main.BuildDate
BUILD_VERSION_VAR := main.BuildVersion
BUILD_GITCOMMIT_VAR := main.GitCommit

LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(CLI_VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -X main.chartTGZSource=$$(cat -)"

$(GOX):
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
	go run scripts/generate_chart/generate_chart.go | CGO_ENABLED=0  go build -v -o ./bin/osm -ldflags ${LDFLAGS} ./cmd/cli

.PHONY: clean-osm
clean-osm:
	@rm -rf bin/osm

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

.PHONY: kind-up
kind-up:
	./scripts/kind-with-registry.sh

.PHONY: kind-reset
kind-reset:
	kind delete cluster --name osm

.env:
	cp .env.example .env

.PHONY: kind-demo
kind-demo: export CTR_REGISTRY=localhost:5000
kind-demo: .env kind-up clean-osm
	./demo/run-osm-demo.sh


# build-bookbuyer, etc
DEMO_TARGETS = bookbuyer bookthief bookstore bookwarehouse
DEMO_BUILD_TARGETS = $(addprefix build-, $(DEMO_TARGETS))
.PHONY: $(DEMO_BUILD_TARGETS)
$(DEMO_BUILD_TARGETS): NAME=$(@:build-%=%)
$(DEMO_BUILD_TARGETS):
	@mkdir -p $(shell pwd)/demo/bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/$(NAME) ./demo/cmd/$(NAME)/$(NAME).go

.PHONY: demo-build
demo-build: $(DEMO_BUILD_TARGETS) build-osm-controller

# docker-build-bookbuyer, etc
DOCKER_TARGETS = bookbuyer bookthief bookstore bookwarehouse init osm-controller
DOCKER_BUILD_TARGETS = $(addprefix docker-build-, $(DOCKER_TARGETS))
.PHONY: $(DOCKER_BUILD_TARGETS)
$(DOCKER_BUILD_TARGETS): NAME=$(@:docker-build-%=%)
$(DOCKER_BUILD_TARGETS):
	if [[ "$(NAME)" != "init" ]] ; then make build-$(NAME); fi
	docker build -t $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) -f dockerfiles/Dockerfile.$(NAME) .

# docker-push-bookbuyer, etc
DOCKER_PUSH_TARGETS = $(addprefix docker-push-, $(DOCKER_TARGETS))
.PHONY: $(DOCKER_PUSH_TARGETS)
$(DOCKER_PUSH_TARGETS): NAME=$(@:docker-push-%=%)
$(DOCKER_PUSH_TARGETS):
	make docker-build-$(NAME)
	docker push "$(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"

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
	go run scripts/generate_chart/generate_chart.go | GO111MODULE=on CGO_ENABLED=0 $(GOX) -ldflags $(LDFLAGS) -parallel=3 -output="_dist/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' ./cmd/cli

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf osm-${CLI_VERSION}-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r osm-${CLI_VERSION}-{}.zip {} \; && \
		sha256sum osm-* > sha256sums.txt \
	)

.PHONY: release-artifacts
release-artifacts: build-cross dist
