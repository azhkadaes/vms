# VMS manual development; commands work without a Unix recipe shell.
.DEFAULT_GOAL := build
.PHONY: env-setup build run start swagger test lint migration-plan contract-check

env-setup:
	go run ./cmd/vms-dev -mode env-setup
build:
	go run ./cmd/vms-dev -mode build
run:
	go run ./cmd/vms-dev -mode run
start:
	go run ./cmd/vms-dev -mode start
swagger:
	go run ./cmd/vms-dev -mode swagger
test:
	go test ./...
lint:
	go vet ./...
migration-plan:
	go run ./cmd/vms-migrate -mode plan
# Checks this checkout (main or worktree) against ../docs/contracts.
contract-check:
	python ../docs/contracts/check.py --backend . --runtime --consumers
