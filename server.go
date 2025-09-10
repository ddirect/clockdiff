package main

import (
	"clockdiff/pkg/packet"
	"clockdiff/pkg/socket"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ddirect/container/ttlmap"
	"golang.org/x/sys/unix"
)

func Server(conf Config) error {
	addr, err := net.ResolveUDPAddr(conf.network, conf.ep)
	if err != nil {
		return fmt.Errorf("net.ResolveUDPAddr: %w", err)
	}

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	localAddr := socket.Addr(addr)
	if err = unix.Bind(fd, localAddr); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	if err = packet.EnableTimestamping(fd); err != nil {
		return err
	}

	log.Printf("listening on %s", socket.AddrToString(localAddr))

	store, expired := ttlmap.New[string, Timing](time.Minute, time.Second)

	send := packet.NewSender[Data](fd, conf.usePoll)
	recvCh := packet.NewAsyncReceiver[Data](fd, 16)

	for {
		select {
		case clients := <-expired:
			for client := range clients {
				log.Printf("client %s expired", client.Key())
			}

		case recvPkt := <-recvCh:
			if recvPkt.Error != nil {
				log.Print(err)
				continue
			}

			timing, found := store.GetOrCreate(socket.AddrToString(recvPkt.From))
			if found {
				recvPkt.Data.ServerTime = timing.Value
			} else {
				log.Printf("new client %s", timing.Key())
			}

			sendTs, err := send(recvPkt.Data, recvPkt.From)
			if err != nil {
				log.Print(err)
				continue
			}

			timing.Value = Timing{
				PacketID: recvPkt.Data.PacketID,
				RecvTime: recvPkt.Ts,
				SendTime: sendTs,
			}
		}
	}
}
