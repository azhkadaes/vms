.DEFAULT_GOAL := plan
.PHONY: plan status up
plan:
	cd .. && go run ./cmd/vms-migrate -mode plan
status:
	cd .. && go run ./cmd/vms-migrate -mode status -expect-database $(database)
up:
	cd .. && go run ./cmd/vms-migrate -mode up -expect-database $(database)