BUILD_TAG=$(shell git describe --tags --abbrev=0)
BUILD_NAME?=golurectl

.PHONY: create-fixtures
test-fixtures:
	go test -tags=fixtures -race -json ./... > ./tests/fixtures/test_sample.txt

.PHONY: integration
test-integration:
	go test -tags=integration -race -v ./...

build:
	go build -trimpath -ldflags "-s -w -X main.BuildName=${BUILD_NAME} -X main.BuildTag=${BUILD_TAG}" -o \
	bin/golurectl ./cmd/golurectl

.PHONY: test
test:
	go test -race -v ./...

.PHONY: test
test-cover:
	@go test -race -v -tags=all ./... -coverprofile=coverage.out

.PHONY: lint
lint:
	golangci-lint run --timeout 5m -v ./...