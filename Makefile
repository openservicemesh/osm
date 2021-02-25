#!make

TARGETS         := darwin/amd64 linux/amd64 windows/amd64
SHELL           := bash -o pipefail
BINNAME         ?= osm
DIST_DIRS       := find * -type d -exec
CTR_REGISTRY    ?= openservicemesh
CTR_TAG         ?= latest

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = go run github.com/mitchellh/gox

VERSION ?= dev
BUILD_DATE=$$(date +%Y-%m-%d-%H:%M)
GIT_SHA=$$(git rev-parse HEAD)
BUILD_DATE_VAR := github.com/openservicemesh/osm/pkg/version.BuildDate
BUILD_VERSION_VAR := github.com/openservicemesh/osm/pkg/version.Version
BUILD_GITCOMMIT_VAR := github.com/openservicemesh/osm/pkg/version.GitCommit

LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -X main.chartTGZSource=$$(cat -) -s -w"

# These two values are combined and passed to go test
E2E_FLAGS ?= -installType=KindCluster
E2E_FLAGS_DEFAULT := -test.v -ginkgo.v -ginkgo.progress -ctrRegistry $(CTR_REGISTRY) -osmImageTag $(CTR_TAG)

# Installed Go version
# This is the version of Go going to be used to compile this project.
# It will be compared with the minimum requirements for OSM.
GO_VERSION_MAJOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_VERSION_MINOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_VERSION_PATCH = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f3)
ifeq ($(GO_VERSION_PATCH),)
GO_VERSION_PATCH := 0
endif

# Required Go version
# These variables set the minimum required version of Go for this project.
# The minimum version below is defined based on:
#   - required language features
#   - known required bug fixes
#   - CVEs
# Current: 1.15.7
# For more context see https://github.com/openservicemesh/osm/issues/2363
# For CVEs fixed with 1.15.7 see https://groups.google.com/g/golang-announce/c/mperVMGa98w
MIN_REQUIRED_GO_VERSION_MAJOR = 1
MIN_REQUIRED_GO_VERSION_MINOR = 15
MIN_REQUIRED_GO_VERSION_PATCH = 7

GO_VERSION_MESSAGE = Installed Go version is $(GO_VERSION_MAJOR).$(GO_VERSION_MINOR).$(GO_VERSION_PATCH). OSM requires Go version $(MIN_REQUIRED_GO_VERSION_MAJOR).$(MIN_REQUIRED_GO_VERSION_MINOR).$(MIN_REQUIRED_GO_VERSION_PATCH) or higher!

check-go-version: # Ensure the Go version used is what OSM requires
	@echo -e "Installed Go version is $(GO_VERSION_MAJOR).$(GO_VERSION_MINOR).$(GO_VERSION_PATCH)."
	@echo -e "OSM requires Go version $(MIN_REQUIRED_GO_VERSION_MAJOR).$(MIN_REQUIRED_GO_VERSION_MINOR).$(MIN_REQUIRED_GO_VERSION_PATCH) or higher!\n\n"
	@if [ $(GO_VERSION_MAJOR) -gt $(MIN_REQUIRED_GO_VERSION_MAJOR) ]; then \
		exit 0 ;\
	elif [ $(GO_VERSION_MAJOR) -lt $(MIN_REQUIRED_GO_VERSION_MAJOR) ]; then \
		echo '$(GO_VERSION_MESSAGE)';\
		exit 1; \
	elif [ $(GO_VERSION_MINOR) -gt $(MIN_REQUIRED_GO_VERSION_MINOR) ] ; then \
		exit 0; \
	elif [ $(GO_VERSION_MINOR) -lt $(MIN_REQUIRED_GO_VERSION_MINOR) ] ; then \
		echo '$(GO_VERSION_MESSAGE)';\
		exit 1; \
	elif [ $(GO_VERSION_PATCH) -lt $(MIN_REQUIRED_GO_VERSION_PATCH) ] ; then \
		echo '$(GO_VERSION_MESSAGE)';\
		exit 1; \
	fi

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

.PHONY: clean-osm-injector
clean-osm-injector:
	@rm -rf bin/osm-injector

.PHONY: build
build: build-osm-controller build-osm-injector

.PHONY: build-osm-controller
build-osm-controller: check-go-version clean-osm-controller wasm/stats.wasm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-controller/osm-controller -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -X github.com/openservicemesh/osm/pkg/envoy/lds.statsWASMBytes=$$(base64 < wasm/stats.wasm | tr -d \\n) -s -w" ./cmd/osm-controller

.PHONY: build-osm-injector
build-osm-injector: check-go-version clean-osm-injector
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-injector/osm-injector -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w" ./cmd/osm-injector

.PHONY: build-osm
build-osm: check-go-version
	go run scripts/generate_chart/generate_chart.go | CGO_ENABLED=0  go build -v -o ./bin/osm -ldflags ${LDFLAGS} ./cmd/cli

.PHONY: clean-osm
clean-osm:
	@rm -rf bin/osm

.PHONY: chart-readme
chart-readme:
	go run github.com/norwoodj/helm-docs/cmd/helm-docs -c charts -t charts/osm/README.md.gotmpl

.PHONY: chart-checks
chart-checks: chart-readme
	@git diff --exit-code charts/osm/README.md || { echo "----- Please commit the changes made by 'make chart-readme' -----"; exit 1; }

.PHONY: check-mocks
check-mocks:
	@go run ./mockspec/generate.go
	@git diff --exit-code || { echo "----- Please commit the changes made by 'go run ./mockspec/generate.go' -----"; exit 1; }

.PHONY: go-checks
go-checks: go-lint go-fmt go-mod-tidy check-mocks

.PHONY: go-vet
go-vet:
	go vet ./...

.PHONY: go-lint
go-lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run --config .golangci.yml

.PHONY: go-fmt
go-fmt:
	go fmt ./...

.PHONY: go-mod-tidy
go-mod-tidy:
	./scripts/go-mod-tidy.sh

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

.PHONY: test-e2e
test-e2e: docker-build-osm-controller docker-build-osm-injector docker-build-init build-osm docker-build-tcp-echo-server
	go test ./tests/e2e $(E2E_FLAGS_DEFAULT) $(E2E_FLAGS)

.env:
	cp .env.example .env

.PHONY: kind-demo
kind-demo: export CTR_REGISTRY=localhost:5000
kind-demo: .env kind-up clean-osm
	./demo/run-osm-demo.sh

# build-bookbuyer, etc
DEMO_TARGETS = bookbuyer bookthief bookstore bookwarehouse tcp-echo-server tcp-client
DEMO_BUILD_TARGETS = $(addprefix build-, $(DEMO_TARGETS))
.PHONY: $(DEMO_BUILD_TARGETS)
$(DEMO_BUILD_TARGETS): NAME=$(@:build-%=%)
$(DEMO_BUILD_TARGETS):
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/$(NAME)/$(NAME) ./demo/cmd/$(NAME)
	@if [ -f demo/$(NAME).html.template ]; then cp demo/$(NAME).html.template demo/bin/$(NAME); fi

.PHONY: demo-build
demo-build: $(DEMO_BUILD_TARGETS) build-osm-controller build-osm-injector

# docker-build-bookbuyer, etc
DOCKER_DEMO_TARGETS = $(addprefix docker-build-, $(DEMO_TARGETS))
.PHONY: $(DOCKER_DEMO_TARGETS)
$(DOCKER_DEMO_TARGETS): NAME=$(@:docker-build-%=%)
$(DOCKER_DEMO_TARGETS):
	make build-$(NAME)
	docker build -t $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) -f dockerfiles/Dockerfile.$(NAME) demo/bin/$(NAME)

docker-build-init:
	docker build -t $(CTR_REGISTRY)/init:$(CTR_TAG) - < dockerfiles/Dockerfile.init

docker-build-osm-controller: build-osm-controller
	docker build -t $(CTR_REGISTRY)/osm-controller:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-controller bin/osm-controller

docker-build-osm-injector: build-osm-injector
	docker build -t $(CTR_REGISTRY)/osm-injector:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-injector bin/osm-injector

wasm/stats.wasm: wasm/stats.cc wasm/Makefile
	docker run --rm -v $(PWD)/wasm:/work -w /work openservicemesh/proxy-wasm-cpp-sdk:956f0d500c380cc1656a2d861b7ee12c2515a664 /build_wasm.sh

.PHONY: docker-build
docker-build: $(DOCKER_DEMO_TARGETS) docker-build-init docker-build-osm-controller docker-build-osm-injector

# docker-push-bookbuyer, etc
DOCKER_PUSH_TARGETS = $(addprefix docker-push-, $(DEMO_TARGETS) init osm-controller osm-injector)
VERIFY_TAGS = 0
.PHONY: $(DOCKER_PUSH_TARGETS)
$(DOCKER_PUSH_TARGETS): NAME=$(@:docker-push-%=%)
$(DOCKER_PUSH_TARGETS):
	@if [ $(VERIFY_TAGS) != 1 ]; then make docker-build-$(NAME); docker push "$(CTR_REGISTRY)/$(NAME):$(CTR_TAG)" || { echo "Error pushing images to container registry $(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"; exit 1; }; else bash scripts/publish-image.sh $(NAME); fi

.PHONY: docker-push
docker-push: $(DOCKER_PUSH_TARGETS)

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
build-cross:
	go run scripts/generate_chart/generate_chart.go | GO111MODULE=on CGO_ENABLED=0 $(GOX) -ldflags $(LDFLAGS) -parallel=3 -output="_dist/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' ./cmd/cli

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf osm-${VERSION}-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r osm-${VERSION}-{}.zip {} \; && \
		sha256sum osm-* > sha256sums.txt \
	)

.PHONY: release-artifacts
release-artifacts: build-cross dist
