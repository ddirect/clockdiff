package main

import (
	"flag"
	"log"
)

func main() {
	if err := mainErr(); err != nil {
		log.Print(err)
	}
}

type Config struct {
	rate       int
	maxSamples int
	maxSpread  float64
	ep         string
	mode       string
	network    string
	usePoll    bool
}

func mainErr() error {
	conf := Config{
		network: "udp",
	}
	var serve bool
	flag.BoolVar(&serve, "serve", false, "server mode")
	flag.IntVar(&conf.rate, "rate", 100, "sending rate (ms)")
	flag.IntVar(&conf.maxSamples, "max-samples", 1000, "maximum number of samples")
	flag.StringVar(&conf.ep, "ep", ":12510", "endpoint to connect to or local endpoint in server mode")
	flag.StringVar(&conf.mode, "mode", "diff", "log mode")
	flag.Float64Var(&conf.maxSpread, "max-spread", 3, "max spread of samples to be considered valid (after max-samples), as a factor of the standard deviation")
	flag.BoolVar(&conf.usePoll, "wait-tx-timestamps", false, "use ppoll to wait for TX timestamps")

	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if serve {
		return Server(conf)
	}

	return Client(conf)
}
