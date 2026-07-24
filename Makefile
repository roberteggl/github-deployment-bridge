# SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
#
# SPDX-License-Identifier: Apache-2.0

MODULE := github.com/roberteggl/github-deployment-bridge
BIN := bin/bridge
IMG ?= ghcr.io/roberteggl/github-deployment-bridge
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build test lint tidy docker-build helm-lint fmt vet

all: tidy fmt vet test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -o $(BIN) ./cmd/bridge

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

docker-build:
	docker build -t $(IMG):$(VERSION) .

helm-lint:
	helm lint charts/github-deployment-bridge
