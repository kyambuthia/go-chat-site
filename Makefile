.PHONY: setup setup-client test test-go test-client lint lint-go lint-client typecheck-client build build-go build-client cover-go check

setup: setup-client

setup-client:
	cd client && npm ci

test: test-go test-client

test-go:
	go test ./server/...

test-client:
	cd client && npm test

lint: lint-go lint-client typecheck-client

lint-go:
	@fmt_out=$$(gofmt -l $$(find server -name '*.go' -type f)); \
	if [ -n "$$fmt_out" ]; then \
		echo "gofmt check failed for:"; \
		echo "$$fmt_out"; \
		exit 1; \
	fi
	go vet ./server/...

lint-client:
	cd client && npm run lint

typecheck-client:
	cd client && npm run typecheck

build: build-go build-client

build-go:
	go build ./server/...

build-client:
	cd client && npm run build

cover-go:
	go test ./server/... -coverprofile=coverage.out
	go tool cover -func=coverage.out

check: lint test build
