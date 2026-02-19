package main

import "log"

type config struct {
	port string
	quit chan struct{}
}

func main() {
	cfg := config{
		port: ":6379", quit: make(chan struct{}),
	}
	log.Fatal(server(&cfg))
}
