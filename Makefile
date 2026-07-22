GO ?= go
.PHONY: fmt test vet run worker compose-up build
fmt:
	$(GO)fmt -w $$(find . -name '*.go' -not -path './vendor/*')
test:
	$(GO) test ./...
vet:
	$(GO) vet ./...
run:
	$(GO) run ./cmd/api-server
worker:
	$(GO) run ./cmd/worker
compose-up:
	docker compose up -d
build:
	$(GO) build ./cmd/api-server ./cmd/worker
