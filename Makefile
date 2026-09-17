BINARY := static_studio
CMD := ./cmd/studio
DIST := ./bin
WEB_DIST := web/dist
PKGS = ./cmd/... ./internal/... ./web

.DEFAULT_GOAL := help
.PHONY: help build run test race lint fmt tidy check clean web-install web-dev web-build web-check web-stub

help: ## show available targets
	@grep -E '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf " %-8s %s\n", $$1, $$2}'

build: web-build ## build the binary with the frontend embedded into ./bin
	go build -o $(DIST)/$(BINARY) $(CMD)

build-dev: web-stub ## build without the frontend (fast)
	go build -o $(DIST)/$(BINARY) $(CMD)

run: web-stub ## run the server
	go run $(CMD)

test: web-stub ## run tests
	go test $(PKGS)

race: web-stub ## run tests with the race detector, no cache
	go test -race -count=1 $(PKGS)

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

web-install: ## install frontend deps
	cd web && npm ci

web-dev: ## run the vite dev server
	cd web && npm run dev

web-build: ## build the frontend into web/dist
	cd web && npm run build

web-check: ##t typecheck, lint and test the frontend
	cd web && npm run type-check && npm run lint && npm run test:unit

$(WEB_DIST)/index.html:
	@mkdir -p $(WEB_DIST)
	@printf '<!doctype html><meta charset="utf-8"><title>static_studio</title><p>Frontend not build. Run <code>make web-build</code>.\n'> $@

web-stub: $(WEB_DIST)/index.html ## create a stub index.html so the embed pattern resolves
