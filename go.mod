module mqueue

go 1.17

require (
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/dustin/go-humanize v1.0.0
	github.com/nsqio/go-nsq v1.1.0
	github.com/nsqio/nsq v1.2.1
	github.com/pkg/errors v0.8.1
	github.com/thanhpk/randstr v1.0.4
	github.com/tjarratt/babble v0.0.0-20210505082055-cbca2a4833c1
	github.com/vmihailenco/msgpack/v5 v5.3.5
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.23.0
)

require (
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/bmizerany/perks v0.0.0-20141205001514-d9a9656a3a4b // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/nsqio/go-diskqueue v1.0.0 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.24.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
)

replace github.com/nsqio/go-nsq => github.com/telepenin/go-nsq v1.1.1
