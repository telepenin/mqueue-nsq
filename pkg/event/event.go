package event

import (
	"encoding"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

var (
	_ encoding.BinaryMarshaler   = Event{}
	_ encoding.BinaryUnmarshaler = Event{}
)

type Event struct {
	Payload interface{}
}

func (e *Event) String() string {
	return fmt.Sprintf("%v", e.Payload)
}

func (e Event) MarshalBinary() ([]byte, error) {
	return msgpack.Marshal(e.Payload)
}

func (e Event) UnmarshalBinary(data []byte) error {
	return msgpack.Unmarshal(data, e.Payload)
}
