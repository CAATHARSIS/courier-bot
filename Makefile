.DEFAULT_GOAL := run

MIG ?=

run:
	@cd cmd/app && go run main.go -local

migrate:
ifndef MIG
	@echo "Error: enter migration name: make migrate MIG=create_users_table"
	@exit 1
endif
	@migrate create -ext sql -dir migrations -seq $(MIG)

help:
	@echo "Available commands:"
	@echo "  make run                         - run application"
	@echo "  make migrate MIG=...  - create migraton with name = MIG"
	@echo "  Example: make migrate MIG=create_users_table"
