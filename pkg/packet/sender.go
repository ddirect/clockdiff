package packet

import (
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

const (
	ethtoolSuggestion = " - use 'ethtool -T <INTERFACE>' to check if your network interface supports TX software timestamping"
)

var (
	ErrNoTimestamp = errors.New("ppoll timed out: no timestamp read from kernel" + ethtoolSuggestion)
)

//go:noinline
func NewSender[T any](fd int, usePoll bool) func(T, unix.Sockaddr) (Timestamp, error) {
	var poll func() error

	if usePoll {
		poll = func() error {
		again:
			// poll for error (POLLERR); this doesn't need to be added to the requested events since it's always enabled
			// NOTE: no allocation occurrs while generating the Ppoll arguments
			n, err := unix.Ppoll([]unix.PollFd{{Fd: int32(fd)}}, &unix.Timespec{Nsec: 100 * 1e6}, nil)
			if err != nil {
				if err == unix.EINTR {
					goto again
				}
				return fmt.Errorf("ppoll: %w", err)
			}
			if n < 1 {
				return ErrNoTimestamp
			}
			return nil
		}
	}

	ctlBuf := make([]byte, ctlBufSize)
	var tBuf []byte
	var succededOnce bool

	return func(t T, to unix.Sockaddr) (Timestamp, error) {
		var err error
		if tBuf, err = binary.Append(tBuf[:0], binary.BigEndian, &t); err != nil {
			return 0, fmt.Errorf("binary.Append on data: %w", err)
		}

		if err = unix.Sendto(fd, tBuf, 0, to); err != nil {
			return 0, fmt.Errorf("sendto: %w", err)
		}

		if poll != nil {
			if err = poll(); err != nil {
				return 0, err
			}
		}

		var ctlN int
		// NOTE: no allocations are done for this dummy slice, which avoids an extra call to GetsockoptInt
		if _, ctlN, _, _, err = unix.Recvmsg(fd, make([]byte, 1), ctlBuf, unix.MSG_ERRQUEUE); err != nil {
			format := "recvmsg errqueue: %w"
			if !usePoll && (err == unix.EAGAIN || err == unix.EWOULDBLOCK) {
				if succededOnce {
					format += " - try calling with -wait-tx-timestamps"
				} else {
					format += ethtoolSuggestion + "; if it does, try calling with -wait-tx-timestamps"
				}
			}
			return 0, fmt.Errorf(format, err)
		}

		ts, err := decodeTimestamp(ctlBuf[:ctlN])
		if err != nil {
			return 0, err
		}
		succededOnce = true

		return ts, nil
	}
}
