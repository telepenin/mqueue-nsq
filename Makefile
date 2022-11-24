.EXPORT_ALL_VARIABLES:

REDIS_HOST ?= localhost

up:
	@echo "Starting server..."
	@docker-compose up -d

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

reload-systemd-consumer:
	@echo "---- Copying Systemd Files & Reload ----"
	/usr/bin/cp -f consumer/mqueue-consumer.service /etc/systemd/system/; \
	/usr/bin/cp -f consumer/mqueue-consumer.socket /etc/systemd/system/; \
	systemctl daemon-reload;

.PHONY: producer
producer:
	@echo "---- Starting Producer ----"
	@go run producer/*.go

.PHONY: producers
producers:
	bash ./run-producers.sh ${NUMBER}

.PHONY: consumers
consumers:
	bash ./run-consumers.sh ${NUMBER}

.PHONY: consumer
consumer:
	@echo "---- Starting Consumer ----"
	@go run consumer/*.go

.PHONY: wakeup
wakeup:
	@echo "---- Starting WakeUp service ----"
	@go run wakeup/*.go

systemd-socket-activate-consumer: #build
	@echo "---- Starting Socket Activation ----"
	systemd-socket-activate -l /run/mqueue-consumer.socket \
		-E STREAM=${STREAM} \
		-E TIMEOUT=${TIMEOUT} \
		./mqueue-consumer

wakeup-consumer:
	@echo "---- Wake up consumer through socket ----"
	@printf WAKEUP | socat UNIX-CONNECT:/var/run/mqueue-consumer.socket -


