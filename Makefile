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
	@go build -a -o mqueue-consumer consumer/*.go
	#@go build -o mqueue-producer producer/*.go
	#@go build -o mqueue-wakeup wakeup/*.go

copy:
	@echo "---- Copying Application ----"
	@cp -f mqueue-consumer /usr/local/bin/sa-consumer


.PHONY: producer
producer:
	@echo "---- Starting Producer ----"
	@go run producer/*.go

.PHONY: consumer
consumer:
	@echo "---- Starting Consumer ----"
	@go run consumer/*.go

.PHONY: wakeup
wakeup:
	@echo "---- Starting WakeUp service ----"
	@go run wakeup/*.go

.PHONY: systemd-socket-activate-consumer
systemd-socket-activate-consumer: build
	@echo "---- Starting Socket Activation ----"
	systemd-socket-activate -l /var/run/mqueue-consumer.socket \
		-E STREAM=${STREAM} \
		-E GROUP=${GROUP} \
		-E TIMEOUT=${TIMEOUT} \
		./mqueue-consumer

.PHONY: wakeup-consumer
wakeup-consumer:
	@echo "---- Wake up consumer through socket ----"
	@printf WAKEUP | socat UNIX-CONNECT:/var/run/mqueue-consumer.socket -
