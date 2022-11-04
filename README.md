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