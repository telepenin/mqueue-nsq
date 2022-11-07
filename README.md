## How to prepare

- start redis in docker

```bash
$ make up
```

## Run producer

```bash
$ STREAM=foo make producer
```

Events will be generated to the stream `foo`

## Run consumers with different groups

```bash
$ STREAM=foo GROUP=1 make consumer
$ STREAM=foo GROUP=2 make consumer
```

Each consumer will receive the same events from the stream `foo`.

## Run consumers with the same group

```bash
$ STREAM=foo GROUP=1 make consumer
$ STREAM=foo GROUP=1 make consumer
```

Each consumer will receive different events from the stream `foo`.
Pretty close how to load balancer works in round-robin mode.

## Run consumers with the same group with timeout

```bash
$ STREAM=foo GROUP=1 TIMEOUT=30 make consumer
```

Consumer will wait new messages for 30 seconds. If no messages will be received, consumer will exit.

## Run socket activation (systemd) consumer

```bash
$ STREAM=foo GROUP=1 TIMEOUT=30 make systemd-socket-activate-consumer
```

Starts the consumer with systemd socket activation. The consumer will be started by systemd when a new data is received
on the socket /var/run/sa-consumer.sock.

## Run wakeup service

```bash
$ TARGETS=foo:/var/run/sa-consumer.sock make wakeup
```

Where is:
- `foo` - stream name
- `/var/run/sa-consumer.sock` - socket path

Targets could be combined:

```bash
TARGETS=foo:/var/run/sa-consumer.sock,bar:/var/run/sa-consumer.sock,foo:/var/run/other-sa-consumer.sock make wakeup
```