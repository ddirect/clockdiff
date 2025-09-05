package packet

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"
)

func NewReceiver[T any](fd int) func() (T, Timestamp, unix.Sockaddr, error) {
	var t T
	tBuf := make([]byte, binary.Size(t))
	ctlBuf := make([]byte, ctlBufSize)

	return func() (T, Timestamp, unix.Sockaddr, error) {
		var zero T
		tN, ctlN, _, from, err := unix.Recvmsg(fd, tBuf, ctlBuf, 0)
		if err != nil {
			return zero, 0, nil, fmt.Errorf("recvfrom: %w", err)
		}

		ts, err := decodeTimestamp(ctlBuf[:ctlN])
		if err != nil {
			return zero, 0, nil, err
		}

		consumed, err := binary.Decode(tBuf[:tN], binary.BigEndian, &t)
		if err != nil {
			return zero, 0, nil, fmt.Errorf("binary.Decode: %w", err)
		}

		if consumed != tN {
			return zero, 0, nil, fmt.Errorf("%d bytes left after reading data", tN-consumed)
		}

		return t, ts, from, nil
	}
}

type RecvPacket[T any] struct {
	Data  T
	Ts    Timestamp
	From  unix.Sockaddr
	Error error
}

func NewAsyncReceiver[T any](fd, chDepth int) <-chan (RecvPacket[T]) {
	ch := make(chan (RecvPacket[T]), chDepth)
	go func() {
		recv := NewReceiver[T](fd)
		for {
			data, ts, from, err := recv()
			ch <- RecvPacket[T]{data, ts, from, err}
		}
	}()
	return ch
}
