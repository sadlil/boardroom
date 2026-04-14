.PHONY: all build fmt lint dep test clean css css-watch

all: fmt lint css build

build: css
	go build -o bin/boardroom ./cmd/boardroom

fmt:
	go fmt ./...
	goimports -w .

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
	rm -rf bin/ node_modules/

# CSS build targets (uses Tailwind CLI to generate static CSS)
css:
	@if command -v npx >/dev/null 2>&1; then \
		npx tailwindcss -i ./ui/tailwind.css -o ./ui/styles.css --minify; \
	else \
		echo "⚠️  npx not found. Run 'npm install' first, or install Node.js."; \
	fi

css-watch:
	npx tailwindcss -i ./ui/tailwind.css -o ./ui/styles.css --watch

# Docker targets
DOCKER_IMAGE ?= sadlil/boardroom
DOCKER_TAG ?= latest

docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
