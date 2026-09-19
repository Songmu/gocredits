VERSION = $(shell godzil show-version)
CURRENT_REVISION = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-s -w -X github.com/Songmu/gocredits.revision=$(CURRENT_REVISION)"
u := $(if $(update),-u)

.PHONY: deps
deps:
	go get ${u}

.PHONY: devel-deps
devel-deps:
	go install github.com/Songmu/godzil/cmd/godzil@latest

.PHONY: test
test:
	go test

.PHONY: build
build:
	go build -ldflags=$(BUILD_LDFLAGS) ./cmd/gocredits

.PHONY: install
install:
	go install -ldflags=$(BUILD_LDFLAGS) ./cmd/gocredits

.PHONY: prepare-release
prepare-release:
	go get
	go mod tidy
	go run ./cmd/gocredits -w .
	git add go.mod CREDITS $(wildcard go.sum)

DIST_DIR = dist
.PHONY: crossbuild
crossbuild: devel-deps
	go mod tidy -diff
	rm -rf $(DIST_DIR)
	godzil crossbuild -pv=v$(VERSION) -build-ldflags=$(BUILD_LDFLAGS) \
      -os=linux,darwin,windows -d=$(DIST_DIR) ./cmd/*
	cd $(DIST_DIR) && shasum -a 256 $$(find * -type f -maxdepth 0) > SHA256SUMS

.PHONY: upload
upload:
	ghr v$(VERSION) $(DIST_DIR)
