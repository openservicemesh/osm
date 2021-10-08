#!make

TARGETS         := darwin/amd64 linux/amd64 windows/amd64
BINNAME         ?= osm
DIST_DIRS       := find * -type d -exec
CTR_REGISTRY    ?= openservicemesh
CTR_TAG         ?= latest
CTR_DIGEST_FILE ?= /tmp/osm_image_digest_$(CTR_TAG).txt
VERIFY_TAGS     ?= false

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

.PHONY: clean-osm-controller
clean-osm-controller:
	@rm -rf bin/osm-controller

.PHONY: clean-osm-injector
clean-osm-injector:
	@rm -rf bin/osm-injector

.PHONY: clean-osm-crds
clean-osm-crds:
	@rm -rf bin/osm-crds

.PHONY: clean-osm-bootstrap
clean-osm-bootstrap:
	@rm -rf bin/osm-bootstrap

.PHONY: build
build: build-osm-controller build-osm-injector build-osm-crds build-osm-bootstrap

.PHONY: build-osm-controller
build-osm-controller: clean-osm-controller pkg/envoy/lds/stats.wasm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-controller/osm-controller -ldflags ${LDFLAGS} ./cmd/osm-controller

.PHONY: build-osm-injector
build-osm-injector: clean-osm-injector
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-injector/osm-injector -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w" ./cmd/osm-injector

.PHONY: build-osm-crds
build-osm-crds: clean-osm-crds
	cp -R ./cmd/osm-bootstrap/crds ./bin/osm-crds

.PHONY: build-osm-bootstrap
build-osm-bootstrap: clean-osm-bootstrap
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/osm-bootstrap/osm-bootstrap -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w" ./cmd/osm-bootstrap

.PHONY: build-osm
build-osm: cmd/cli/chart.tgz
	CGO_ENABLED=0  go build -v -o ./bin/osm -ldflags ${LDFLAGS} ./cmd/cli

cmd/cli/chart.tgz: scripts/generate_chart/generate_chart.go $(shell find charts/osm)
	helm dependency update charts/osm
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

.PHONY: check-codegen
check-codegen:
	@./codegen/gen-crd-client.sh
	@git diff --exit-code || { echo "----- Please commit the changes made by './codegen/gen-crd-client.sh' -----"; exit 1; }

.PHONY: go-checks
go-checks: go-lint go-fmt go-mod-tidy check-mocks check-codegen

.PHONY: go-vet
go-vet:
	go vet ./...

.PHONY: go-lint
go-lint: cmd/cli/chart.tgz pkg/envoy/lds/stats.wasm
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.41.1 golangci-lint run --config .golangci.yml

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
test-e2e: docker-build-osm-controller docker-build-osm-injector docker-build-osm-crds docker-build-osm-bootstrap docker-build-init build-osm docker-build-tcp-echo-server
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
ifeq ($(OS), windows)
	EXT=.exe
endif
$(DEMO_BUILD_TARGETS):
	GOOS=$(OS) GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/$(NAME)/$(NAME)$(EXT) ./demo/cmd/$(NAME)
	@if [ -f demo/$(NAME).html.template ]; then cp demo/$(NAME).html.template demo/bin/$(NAME); fi

.PHONE: build-bookwatcher
build-bookwatcher:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookwatcher/bookwatcher ./demo/cmd/bookwatcher

.PHONY: demo-build
demo-build: $(DEMO_BUILD_TARGETS) build-osm-controller build-osm-injector build-osm-crds build-osm-bootstrap

# docker-build-bookbuyer, etc
DOCKER_DEMO_TARGETS = $(addprefix docker-build-, $(DEMO_TARGETS))
.PHONY: $(DOCKER_DEMO_TARGETS)
$(DOCKER_DEMO_TARGETS): NAME=$(@:docker-build-%=%)
$(DOCKER_DEMO_TARGETS):
	make OS=linux build-$(NAME)
	docker build -t $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) -f dockerfiles/Dockerfile.$(NAME) demo/bin/$(NAME)


# docker-build-windows-bookbuyer, etc
# This command can be used to push the images as well if the ARGS is set to --push
# see https://docs.docker.com/engine/reference/commandline/buildx_build/#push
# The reason for that is that on linux we can't load a Windows image so we need to build and push with one command.
DOCKER_WINDOWS_DEMO_TARGETS = $(addprefix docker-build-windows-, $(DEMO_TARGETS))
.PHONY: $(DOCKER_WINDOWS_DEMO_TARGETS)
$(DOCKER_WINDOWS_DEMO_TARGETS): OS = windows
$(DOCKER_WINDOWS_DEMO_TARGETS): NAME=$(@:docker-build-windows-%=%)
$(DOCKER_WINDOWS_DEMO_TARGETS):
	make OS=windows build-$(NAME)
	@if ! docker buildx ls | grep -q "img-builder "; then  echo "Creating buildx img-builder"; docker buildx create --name img-builder; fi
	docker buildx build --builder img-builder --platform "windows/amd64" -t $(CTR_REGISTRY)/$(NAME)-windows:$(CTR_TAG) $(ARGS) -f dockerfiles/Dockerfile.$(NAME).windows demo/bin/$(NAME)

docker-build-init:
	docker build -t $(CTR_REGISTRY)/init:$(CTR_TAG) - < dockerfiles/Dockerfile.init

docker-build-osm-controller: build-osm-controller
	docker build -t $(CTR_REGISTRY)/osm-controller:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-controller bin/osm-controller

docker-build-osm-injector: build-osm-injector
	docker build -t $(CTR_REGISTRY)/osm-injector:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-injector bin/osm-injector

docker-build-osm-crds: build-osm-crds
	docker build -t $(CTR_REGISTRY)/osm-crds:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-crds bin/osm-crds

docker-build-osm-bootstrap: build-osm-bootstrap
	docker build -t $(CTR_REGISTRY)/osm-bootstrap:$(CTR_TAG) -f dockerfiles/Dockerfile.osm-bootstrap bin/osm-bootstrap

pkg/envoy/lds/stats.wasm: wasm/stats.cc wasm/Makefile
	docker run --rm -v $(PWD)/wasm:/work -w /work openservicemesh/proxy-wasm-cpp-sdk:956f0d500c380cc1656a2d861b7ee12c2515a664 /build_wasm.sh
	@mv -f wasm/stats.wasm $@

.PHONY: docker-build
docker-build: $(DOCKER_DEMO_TARGETS) docker-build-init docker-build-osm-controller docker-build-osm-injector docker-build-osm-crds docker-build-osm-bootstrap

.PHONY: embed-files
embed-files: cmd/cli/chart.tgz pkg/envoy/lds/stats.wasm

.PHONY: embed-files-test
embed-files-test:
	./scripts/generate-dummy-embed.sh

.PHONY: build-ci
build-ci: embed-files
	go build -v ./...

.PHONY: clean-image-digest
clean-image-digest:
	@rm -f "$(CTR_DIGEST_FILE)"

trivy-scan-images:
	wget https://github.com/aquasecurity/trivy/releases/download/v0.18.0/trivy_0.18.0_Linux-64bit.tar.gz
	tar zxvf trivy_0.18.0_Linux-64bit.tar.gz

	# Show all vulnerabilities in logs
	./trivy $(CTR_REGISTRY)/osm-controller:$(CTR_TAG)
	./trivy $(CTR_REGISTRY)/osm-injector:$(CTR_TAG)
	./trivy $(CTR_REGISTRY)/init:$(CTR_TAG)
	./trivy $(CTR_REGISTRY)/osm-bootstrap:$(CTR_TAG)
	./trivy $(CTR_REGISTRY)/osm-crds:$(CTR_TAG)

	# Exit if vulnerability exists
	./trivy --exit-code 1 --ignore-unfixed --severity MEDIUM,HIGH,CRITICAL "$(CTR_REGISTRY)/osm-controller:$(CTR_TAG)" || exit 1
	./trivy --exit-code 1 --ignore-unfixed --severity MEDIUM,HIGH,CRITICAL "$(CTR_REGISTRY)/osm-injector:$(CTR_TAG)" || exit 1
	./trivy --exit-code 1 --ignore-unfixed --severity MEDIUM,HIGH,CRITICAL "$(CTR_REGISTRY)/init:$(CTR_TAG)" || exit 1
	./trivy --exit-code 1 --ignore-unfixed --severity MEDIUM,HIGH,CRITICAL "$(CTR_REGISTRY)/osm-bootstrap:$(CTR_TAG)" || exit 1
	./trivy --exit-code 1 --ignore-unfixed --severity MEDIUM,HIGH,CRITICAL "$(CTR_REGISTRY)/osm-crds:$(CTR_TAG)" || exit 1

# OSM control plane components
DOCKER_PUSH_CONTROL_PLANE_TARGETS = $(addprefix docker-push-, init osm-controller osm-injector osm-crds osm-bootstrap)
.PHONY: $(DOCKER_PUSH_CONTROL_PLANE_TARGETS)
$(DOCKER_PUSH_CONTROL_PLANE_TARGETS): NAME=$(@:docker-push-%=%)
$(DOCKER_PUSH_CONTROL_PLANE_TARGETS):
	scripts/publish-image.sh "$(NAME)" "linux" "$(CTR_REGISTRY)" "$(CTR_TAG)"
	@docker images --digests | grep "$(CTR_REGISTRY)/$(NAME)\s*$(CTR_TAG)" >> "$(CTR_DIGEST_FILE)"


# Linux demo applications
DOCKER_PUSH_LINUX_TARGETS = $(addprefix docker-push-, $(DEMO_TARGETS))
.PHONY: $(DOCKER_PUSH_LINUX_TARGETS)
$(DOCKER_PUSH_LINUX_TARGETS): NAME=$(@:docker-push-%=%)
$(DOCKER_PUSH_LINUX_TARGETS):
	scripts/publish-image.sh "$(NAME)" "linux" "$(CTR_REGISTRY)" "$(CTR_TAG)"


# Windows demo applications
DOCKER_PUSH_WINDOWS_TARGETS = $(addprefix docker-push-windows-, $(DEMO_TARGETS))
.PHONY: $(DOCKER_PUSH_WINDOWS_TARGETS)
$(DOCKER_PUSH_WINDOWS_TARGETS): NAME=$(@:docker-push-windows-%=%)
$(DOCKER_PUSH_WINDOWS_TARGETS):
	scripts/publish-image.sh "$(NAME)" "windows" "$(CTR_REGISTRY)" "$(CTR_TAG)"


.PHONY: docker-control-plane-push
docker-control-plane-push: clean-image-digest $(DOCKER_PUSH_CONTROL_PLANE_TARGETS)

.PHONY: docker-linux-push
docker-linux-push: docker-control-plane-push $(DOCKER_PUSH_LINUX_TARGETS)

.PHONY: docker-windows-push
docker-windows-push: docker-control-plane-push $(DOCKER_PUSH_WINDOWS_TARGETS)

.PHONY: docker-push
docker-push: docker-linux-push docker-windows-push

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
