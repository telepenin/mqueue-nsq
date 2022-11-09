## How to prepare

- start redis in docker

```bash
make up
```

## Run producer

```bash
STREAM=foo make producer
```

Events will be generated to the stream `foo`

## Run consumers with different groups

```bash
STREAM=foo GROUP=1 make consumer
STREAM=foo GROUP=2 make consumer
```

Each consumer will receive the same events from the stream `foo`.

## Run consumers with the same group

```bash
STREAM=foo GROUP=1 make consumer
STREAM=foo GROUP=1 make consumer
```

Each consumer will receive different events from the stream `foo`.
Pretty close how to load balancer works in round-robin mode.

## Run consumers with the same group with timeout

```bash
STREAM=foo GROUP=1 TIMEOUT=30 make consumer
```

Consumer will wait new messages for 30 seconds. If no messages will be received, consumer will exit.

## Run socket activation (systemd) consumer for development

It uses systemd-socket-activate to listen on a socket and launch consumer when new data arriving in socket.

```bash
STREAM=foo GROUP=1 TIMEOUT=30 make systemd-socket-activate-consumer
```

Starts the consumer with systemd socket activation. The consumer will be started by systemd when a new data is received
on the socket /var/run/sa-consumer.sock.

## Run wakeup service

```bash
TARGETS=foo:/var/run/sa-consumer.sock make wakeup
```

Where is:

- `foo` - stream name
- `/var/run/sa-consumer.sock` - socket path

Targets could be combined:

```bash
TARGETS=foo:/var/run/sa-consumer.sock,bar:/var/run/sa-consumer.sock,foo:/var/run/other-sa-consumer.sock make wakeup
```

## Run with systemd

```bash
cp systemd/sa-consumer.service /etc/systemd/system/
cp systemd/sa-consumer.socket /etc/systemd/system/
systemctl daemon-reload
systemctl enable sa-consumer.socket
systemctl enable sa-consumer.service
systemctl start sa-consumer.socket

TARGETS=foo:/var/run/mqueue-consumer.socket make wakeup
```

Services are ready to receive new messages, to generate a new one - run producer.

Show logs of consumer:

```bash
journalctl -u mq-consumer.service
```
