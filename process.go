package main

import (
	"clockdiff/pkg/stats"
	"fmt"
	"time"
)

func b2s(b bool) string {
	if b {
		return "*"
	}
	return "-"
}

func Process(ch <-chan Sample, mode string, maxSamples int, maxSpread float64) {
	diff := stats.New[time.Duration](maxSamples, maxSpread)
	roundtrip := stats.New[time.Duration](maxSamples, maxSpread)
	for s := range ch {
		forward := time.Duration(s.RequestRecvTime - s.RequestSendTime)
		backward := time.Duration(s.ResponseRecvTime - s.ResponseSendTime)

		diffSample := (backward - forward) / 2
		rtSample := forward + backward

		diffV := diff.SampleIn(diffSample)
		rtV := roundtrip.SampleIn(rtSample)

		switch mode {
		case "raw":
			fmt.Printf("%5d lost %5d inva %d -> %d -- %d -> %d / %20d%20d%20d\n", s.LostCount, s.InvalidCount,
				s.RequestSendTime, s.RequestRecvTime, s.ResponseSendTime, s.ResponseRecvTime,
				s.RequestRecvTime-s.RequestSendTime, s.ResponseSendTime-s.RequestRecvTime, s.ResponseRecvTime-s.ResponseSendTime)
		case "sample":
			fmt.Printf("%20v -> %20v <- %20v diff %20v rt\n", forward, backward, diffSample, rtSample)
		default:
			fmt.Printf("%s%s%5d lost %5d inva %5d sampl %20v diffM %15v diffSD %15v rtM %15v rtSD\n",
				b2s(diffV),
				b2s(rtV),
				s.LostCount,
				s.InvalidCount,
				diff.SampleCount(),
				diff.Mean(),
				diff.StdDev(),
				roundtrip.Mean(),
				roundtrip.StdDev(),
			)
		}
	}
}
