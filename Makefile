VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)
LDFLAGS := -X github.com/jinkp/bbkit/internal/version.Version=$(VERSION)

ifeq ($(OS),Windows_NT)
BIN_EXT := .exe
else
BIN_EXT :=
endif

BIN_DIR := bin
BUILD_TARGET := ./cmd/bbk
BUILD_OUTPUT := $(BIN_DIR)/bbk$(BIN_EXT)
RELEASE_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build build-all test lint clean install

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_OUTPUT) $(BUILD_TARGET)

build-all:
	@mkdir -p $(BIN_DIR)
	@for target in $(RELEASE_PLATFORMS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o "$(BIN_DIR)/bbk-$$os-$$arch$$ext" $(BUILD_TARGET); \
	done

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)

install:
	go install -ldflags "$(LDFLAGS)" $(BUILD_TARGET)
