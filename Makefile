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
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-controller/osm-controller -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w" ./cmd/osm-controller

.PHONY: build-osm
build-osm:
	go run scripts/generate_chart/generate_chart.go | CGO_ENABLED=0  go build -v -o ./bin/osm -ldflags ${LDFLAGS} ./cmd/cli

.PHONY: clean-osm
clean-osm:
	@rm -rf bin/osm

.PHONY: chart-readme
chart-readme:
	go run github.com/norwoodj/helm-docs/cmd/helm-docs -c charts -t charts/osm/README.md.gotmpl

.PHONY: go-checks
go-checks: go-lint go-fmt go-mod-tidy

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
test-e2e: docker-build-osm-controller docker-build-init build-osm docker-build-tcp-echo-server
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
demo-build: $(DEMO_BUILD_TARGETS) build-osm-controller

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

.PHONY: docker-build
docker-build: $(DOCKER_DEMO_TARGETS) docker-build-init docker-build-osm-controller

# docker-push-bookbuyer, etc
DOCKER_PUSH_TARGETS = $(addprefix docker-push-, $(DEMO_TARGETS) init osm-controller)
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
