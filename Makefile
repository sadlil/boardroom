.PHONY: all build fmt lint dep test clean

all: fmt lint build

build:
	go build -o bin/boardroom ./cmd/boardroom

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

dep:
	go mod download
	go mod tidy -e

test:
	go test ./...

test-v:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

clean:
	go clean -i -r -x
	rm -rf bin/

# Docker targets
DOCKER_IMAGE ?= sadlil/boardroom
DOCKER_TAG ?= latest

docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
