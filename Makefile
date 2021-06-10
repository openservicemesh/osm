#!make

TARGETS         := darwin/amd64 linux/amd64 windows/amd64
BINNAME         ?= osm
DIST_DIRS       := find * -type d -exec
CTR_REGISTRY    ?= openservicemesh
CTR_TAG         ?= latest

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = go run github.com/mitchellh/gox

VERSION ?= dev
BUILD_DATE ?=
GIT_SHA=$$(git rev-parse HEAD)
BUILD_DATE_VAR := github.com/openservicemesh/osm/pkg/version.BuildDate
BUILD_VERSION_VAR := github.com/openservicemesh/osm/pkg/version.Version
BUILD_GITCOMMIT_VAR := github.com/openservicemesh/osm/pkg/version.GitCommit

LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w"

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

.PHONY: clean-init-osm-controller
clean-init-osm-controller:
	@rm -rf bin/init-osm-controller

.PHONY: clean-osm-controller
clean-osm-controller:
	@rm -rf bin/osm-controller

.PHONY: clean-osm-injector
clean-osm-injector:
	@rm -rf bin/osm-injector

.PHONY: build
build: build-init-osm-controller build-osm-controller build-osm-injector

.PHONY: build-init-osm-controller
build-init-osm-controller: clean-init-osm-controller
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/init-osm-controller/init-osm-controller -ldflags ${LDFLAGS} ./cmd/init-osm-controller

.PHONY: build-osm-controller
build-osm-controller: clean-osm-controller pkg/envoy/lds/stats.wasm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-controller/osm-controller -ldflags ${LDFLAGS} ./cmd/osm-controller

.PHONY: build-osm-injector
build-osm-injector: clean-osm-injector
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-injector/osm-injector -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w" ./cmd/osm-injector

.PHONY: build-osm
build-osm: cmd/cli/chart.tgz
	CGO_ENABLED=0  go build -v -o ./bin/osm -ldflags ${LDFLAGS} ./cmd/cli

cmd/cli/chart.tgz: scripts/generate_chart/generate_chart.go $(wildcard charts/osm/**/*)
	go run $< > $@

.PHONY: clean-osm
clean-osm:
	@rm -rf bin/osm

.PHONY: codegen
codegen:
	./codegen/gen-crd-client.sh

.PHONY: chart-readme
chart-readme:
	go run github.com/norwoodj/helm-docs/cmd/helm-docs -c charts -t charts/osm/README.md.gotmpl

.PHONY: chart-check-readme
chart-check-readme: chart-readme
	@git diff --exit-code charts/osm/README.md || { echo "----- Please commit the changes made by 'make chart-readme' -----"; exit 1; }

.PHONY: helm-lint
helm-lint:
	@helm lint charts/osm/ || { echo "----- Schema validation failed for OSM chart values -----"; exit 1; }

.PHONY: chart-checks
chart-checks: chart-check-readme helm-lint

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
go-lint: cmd/cli/chart.tgz pkg/envoy/lds/stats.wasm
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
go-test-coverage: embed-files
	./scripts/test-w-coverage.sh

.PHONY: kind-up
kind-up:
	./scripts/kind-with-registry.sh

.PHONY: kind-reset
kind-reset:
	kind delete cluster --name osm

.PHONY: test-e2e
test-e2e: docker-build-init-osm-controller docker-build-osm-controller docker-build-osm-injector docker-build-init build-osm docker-build-tcp-echo-server
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
demo-build: $(DEMO_BUILD_TARGETS) build-init-osm-controller build-osm-controller build-osm-injector

# docker-build-bookbuyer, etc
DOCKER_DEMO_TARGETS = $(addprefix docker-build-, $(DEMO_TARGETS))
.PHONY: $(DOCKER_DEMO_TARGETS)
$(DOCKER_DEMO_TARGETS): NAME=$(@:docker-build-%=%)
$(DOCKER_DEMO_TARGETS):
	make build-$(NAME)
	docker build -t $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) -f dockerfiles/Dockerfile.$(NAME) demo/bin/$(NAME)

docker-build-init:
	docker build -t $(CTR_REGISTRY)/init:$(CTR_TAG) - < dockerfiles/Dockerfile.init

docker-build-init-osm-controller: build-init-osm-controller
	docker build -t $(CTR_REGISTRY)/init-osm-controller:$(CTR_TAG) -f dockerfiles/Dockerfile.init-osm-controller bin/init-osm-controller

docker-build-osm-controller: build-osm-controller
	docker build -t $(CTR_REGISTRY)/osm-controller:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-controller bin/osm-controller

docker-build-osm-injector: build-osm-injector
	docker build -t $(CTR_REGISTRY)/osm-injector:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-injector bin/osm-injector

pkg/envoy/lds/stats.wasm: wasm/stats.cc wasm/Makefile
	docker run --rm -v $(PWD)/wasm:/work -w /work openservicemesh/proxy-wasm-cpp-sdk:956f0d500c380cc1656a2d861b7ee12c2515a664 /build_wasm.sh
	@mv wasm/stats.wasm $@
.PHONY: docker-build
docker-build: $(DOCKER_DEMO_TARGETS) docker-build-init docker-build-init-osm-controller  docker-build-osm-controller docker-build-osm-injector

.PHONY: embed-files
embed-files: cmd/cli/chart.tgz pkg/envoy/lds/stats.wasm

.PHONY: embed-files-test
embed-files-test:
	./scripts/generate-dummy-embed.sh

.PHONY: build-ci
build-ci: embed-files
	go build -v ./...

# docker-push-bookbuyer, etc
DOCKER_PUSH_TARGETS = $(addprefix docker-push-, $(DEMO_TARGETS) init init-osm-controller osm-controller osm-injector)
VERIFY_TAGS = 0
.PHONY: $(DOCKER_PUSH_TARGETS)
$(DOCKER_PUSH_TARGETS): NAME=$(@:docker-push-%=%)
$(DOCKER_PUSH_TARGETS):
	@if [ $(VERIFY_TAGS) != 1 ]; then make docker-build-$(NAME) && docker push "$(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"; else bash scripts/publish-image.sh $(NAME); fi

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
build-cross: cmd/cli/chart.tgz
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -ldflags $(LDFLAGS) -parallel=3 -output="_dist/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' ./cmd/cli

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
