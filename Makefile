GOVERSION := $(shell go version | cut -d ' ' -f 3 | cut -d '.' -f 2)

.PHONY: test test-race test-cover-html help
.DEFAULT_GOAL := help

test: ## Run tests
	go test ./...


test-race: ## Run tests with race detector
	go test -race ./...

test-cover-html: ## Generate test coverage report
	go test ./... -coverprofile=buffer_coverage.out -covermode=count
	go tool cover -func=buffer_coverage.out

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
