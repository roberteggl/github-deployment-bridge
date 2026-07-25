# SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
#
# SPDX-License-Identifier: Apache-2.0

MODULE := github.com/roberteggl/github-deployment-bridge
BIN := bin/bridge
IMG ?= ghcr.io/roberteggl/github-deployment-bridge
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build test lint govulncheck tidy docker-build helm-lint fmt vet changelog

all: tidy fmt vet lint govulncheck test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" -o $(BIN) ./cmd/bridge

test:
	go test ./...

lint:
	golangci-lint run ./...

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$$(git rev-parse HEAD 2>/dev/null || echo unknown) -t $(IMG):$(VERSION) .

helm-lint:
	helm lint charts/github-deployment-bridge \
		--set github.appId=1 \
		--set github.installationId=1 \
		--set github.privateKey=ci-fake-key

changelog:
	git cliff --unreleased
