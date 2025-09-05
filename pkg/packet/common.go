package packet

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

type (
	Timestamp int64 // unix time in nanoseconds - differences can be cast directly to time.Duration
)

var (
	ErrTimestampNotFound            = errors.New("no timestamp found in control data")
	ErrScmTimestampingNotEnoughData = errors.New("not enough data received for ScmTimestamping")
)

const (
	ctlBufSize = 256 // 64 bytes are enough for a single ScmTimestamping structure plus Cmsghdr (on x86_64)
)

func EnableTimestamping(fd int) error {
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING,
		unix.SOF_TIMESTAMPING_RX_SOFTWARE|
			unix.SOF_TIMESTAMPING_TX_SOFTWARE|
			unix.SOF_TIMESTAMPING_SOFTWARE|
			unix.SOF_TIMESTAMPING_OPT_TSONLY,
	); err != nil {
		return fmt.Errorf("setsockopt: %w", err)
	}
	return nil
}

func decodeTimestamp(buf []byte) (Timestamp, error) {
	for len(buf) > 0 {
		hdr, data, remainder, err := unix.ParseOneSocketControlMessage(buf)
		if err != nil {
			return 0, fmt.Errorf("unix.ParseOneSocketControlMessage: %w", err)
		}

		switch hdr.Level {
		case unix.SOL_SOCKET:
			switch hdr.Type {
			case unix.SCM_TIMESTAMPING:
				if uintptr(len(data)) < unsafe.Sizeof(unix.ScmTimestamping{}) {
					return 0, ErrScmTimestampingNotEnoughData
				}
				scmTs := (*unix.ScmTimestamping)(unsafe.Pointer(unsafe.SliceData(data)))
				return Timestamp(scmTs.Ts[0].Nano()), nil
			}
		}

		buf = remainder
	}
	return 0, ErrTimestampNotFound
}
