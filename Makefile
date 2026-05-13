ifneq (,$(filter migrate-%,$(MAKECMDGOALS)))
ifneq (,$(wildcard .env))
include .env
export
endif
endif

MIGRATE := go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate
MIGRATIONS_PATH := migrations
MOCKERY_VERSION := v2.53.6
MOCKERY_BIN := $(shell go env GOPATH)/bin/mockery

.PHONY: test tidy run-server run-agent install-mockery mockery-version mocks migrate-up migrate-down migrate-version migrate-force

test:
	go test ./...

tidy:
	go mod tidy

run-server:
	go run ./cmd/server

run-agent:
	go run ./cmd/agent

install-mockery:
	go install github.com/vektra/mockery/v2@$(MOCKERY_VERSION)

mockery-version:
	$(MOCKERY_BIN) --version

mocks:
	$(MOCKERY_BIN)

migrate-up:
	@test -n "$(DATABASE_DSN)" || (echo "DATABASE_DSN is required"; exit 1)
	@$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_DSN)" up

migrate-down:
	@test -n "$(DATABASE_DSN)" || (echo "DATABASE_DSN is required"; exit 1)
	@$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_DSN)" down 1

migrate-version:
	@test -n "$(DATABASE_DSN)" || (echo "DATABASE_DSN is required"; exit 1)
	@$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_DSN)" version

migrate-force:
	@test -n "$(DATABASE_DSN)" || (echo "DATABASE_DSN is required"; exit 1)
	@test -n "$(VERSION)" || (echo "VERSION is required, example: make migrate-force VERSION=1"; exit 1)
	@$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_DSN)" force $(VERSION)
