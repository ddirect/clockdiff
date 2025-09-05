package socket

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

func Addr(x *net.UDPAddr) unix.Sockaddr {
	res := &unix.SockaddrInet4{
		Port: x.Port,
	}
	copy(res.Addr[:], x.IP.To4())
	return res
}

func AddrToString(sa unix.Sockaddr) string {
	switch v := sa.(type) {
	case *unix.SockaddrInet4:
		ip := net.IP(v.Addr[:])
		return fmt.Sprintf("%s:%d", ip, v.Port)
	case *unix.SockaddrInet6:
		ip := net.IP(v.Addr[:])
		return fmt.Sprintf("[%s]:%d", ip, v.Port)
	case *unix.SockaddrUnix:
		return v.Name
	default:
		panic(fmt.Errorf("unsupported address type %T", v))
	}
}
