#!make

TARGETS         := linux/amd64
LDFLAGS         :=
SHELL           := bash -o pipefail
CTR_REGISTRY    := smctest.azurecr.io
CTR_TAG         := latest

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin

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

.PHONY: kind-up
kind-up:
	./scripts/kind-with-registry.sh

.PHONY: kind-reset
kind-reset:
	kind delete cluster --name osm

.PHONY: kind-load
	kind-load: docker-build-init docker-build docker-build-bookthief

.env:
	cp .env.example .env

.PHONY: kind-demo
kind-demo: kind-up demo-build docker-build docker-build-bookthief kind-load
	./bin/osm install

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
	docker build -t $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) -f dockerfiles/Dockerfile.$(NAME) .

# kind-load-bookbuyer, etc
KIND_LOAD_TARGETS = $(addprefix kind-load-, $(DOCKER_TARGETS))
.PHONY: $(KIND_LOAD_TARGETS)
$(KIND_LOAD_TARGETS): NAME=$(@:kind-load-%=%)
$(KIND_LOAD_TARGETS):
	kind load docker-image $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) --name osm

.PHONY: kind-load
kind-load: $(KIND_LOAD_TARGETS)

# docker-push-bookbuyer, etc
DOCKER_PUSH_TARGETS = $(addprefix docker-push-, $(DOCKER_TARGETS))
.PHONY: $(DOCKER_PUSH_TARGETS)
$(DOCKER_PUSH_TARGETS): NAME=$(@:docker-push-%=%)
$(DOCKER_PUSH_TARGETS): docker-build-$(NAME)
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
