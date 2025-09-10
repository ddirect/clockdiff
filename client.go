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

type TransInfo struct {
	SendTs    packet.Timestamp
	RecvTs    packet.Timestamp
	Processed bool
}

func Client(conf Config) error {
	sampleCh := make(chan Sample, 16)
	defer close(sampleCh)

	go Process(sampleCh, conf.mode, conf.maxSamples, conf.maxSpread)

	addr, err := net.ResolveUDPAddr(conf.network, conf.ep)
	if err != nil {
		return fmt.Errorf("resolve addr: %w", err)
	}

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	remoteAddr := socket.Addr(addr)
	if err = unix.Connect(fd, remoteAddr); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	if err = packet.EnableTimestamping(fd); err != nil {
		return err
	}

	recvCh := packet.NewAsyncReceiver[Data](fd, 16)

	send := packet.NewSender[Data](fd, conf.usePoll)
	tick := time.NewTicker(time.Millisecond * time.Duration(conf.rate)).C

	var (
		id      packetID
		lost    int
		invalid int
	)

	inflight, expired := ttlmap.New[packetID, TransInfo](time.Second, 100*time.Millisecond)

	for {
		select {
		case seq := <-expired:
			for ti := range seq {
				if !ti.Value.Processed {
					lost++
				}
			}
		case <-tick:
			id++
			ts, err := send(Data{PacketID: id}, nil)
			if err != nil {
				return err
			}
			inflight.Set(id, TransInfo{SendTs: ts})
		case recvPkt := <-recvCh:
			if recvPkt.Error != nil {
				log.Print(err)
				continue
			}
			if i1 := inflight.Get(recvPkt.Data.PacketID); i1.Present() {
				i1.Value.RecvTs = recvPkt.Ts
			} else {
				invalid++
			}
			if st := &recvPkt.Data.ServerTime; st.IsPopulated() {
				if i2 := inflight.Get(st.PacketID); i2.Present() && i2.Value.RecvTs > 0 && !i2.Value.Processed {
					i2.Value.Processed = true
					sampleCh <- Sample{
						RequestSendTime:  i2.Value.SendTs,
						RequestRecvTime:  st.RecvTime,
						ResponseSendTime: st.SendTime,
						ResponseRecvTime: i2.Value.RecvTs,
						LostCount:        lost,
						InvalidCount:     invalid,
					}
				} else {
					invalid++
				}
			}
		}
	}
}
