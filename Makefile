GO ?= go
.PHONY: fmt test vet run worker migrate compose-up compose-app build
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
migrate:
	$(GO) run ./cmd/migrate
compose-up:
	docker compose up -d postgres redis
compose-app:
	docker compose --profile app up -d --build
build:
	$(GO) build ./cmd/api-server ./cmd/worker ./cmd/migrate
