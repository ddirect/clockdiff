package main

import (
	"clockdiff/pkg/packet"
)

type packetID uint16

type Timing struct {
	PacketID packetID
	RecvTime packet.Timestamp
	SendTime packet.Timestamp
}

func (d *Timing) IsPopulated() bool {
	return d.RecvTime > 0 && d.SendTime > 0
}

type Data struct {
	PacketID   packetID
	ServerTime Timing
}

type Sample struct {
	RequestSendTime  packet.Timestamp
	RequestRecvTime  packet.Timestamp
	ResponseSendTime packet.Timestamp
	ResponseRecvTime packet.Timestamp
	LostCount        int
	InvalidCount     int
}
