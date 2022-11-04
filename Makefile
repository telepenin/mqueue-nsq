.EXPORT_ALL_VARIABLES:

REDIS_HOST ?= localhost

.PHONY: up
up:
	@echo "Starting server..."
	@docker-compose up -d

.PHONY: down
down:
	@echo "Stopping server..."
	@docker-compose down

.PHONY: build
build:
	@echo "---- Building Application ----"
	@go build -o consumer consumer/*.go
	@go build -o producer producer/*.go

.PHONY: producer
producer:
	@echo "---- Starting Producer ----"
	@go run producer/*.go

.PHONY: consumer
consumer:
	@echo "---- Starting Consumer ----"
	@export GROUP=group1
	@go run consumer/*.go

.PHONY: wakeup
wakeup:
	@echo "---- Starting WakeUp service ----"
	@go run wakeup/*.go

.PHONY: creategroup
creategroup:
	@echo "---- Creating Group ----"
