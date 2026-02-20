package main

import (
	"log"
	"os"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

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
