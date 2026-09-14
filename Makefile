BINARY := static_studio
CMD := ./cmd/studio
DIST := ./bin

.DEFAULT_GOAL := help
.PHONY: help build run test race lint fmt tidy check clean

help: ## show available targets
	@grep -E '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf " %-8s %s\n", $$1, $$2}'

build: ## build the binary into ./bin
	go build -o $(DIST)/$(BINARY) $(CMD)

run: ## run the server
	go run $(CMD)

test: ## run tests
	go test ./...

race: ## run tests with the race detector, no cache
	go test -race -count=1 ./...

lint: ##run golangci-lint
	golangci-lint run

fmt: ## format the code
	golangci-lint fmt

tidy:
	go mod tidy

check: tidy fmt lint race ## everything CI will run
	@echo "green"

clean: ## remove build artifacts and the test cache
	rm -rf $(DIST)
	go clean -testcache
