package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

func server(cfg *config) error {
	listen, err := net.Listen("tcp", cfg.port)
	if err != nil {
		return fmt.Errorf("error listening on %s: err=%w", cfg.port, err)
	}
	defer func() { _ = listen.Close() }()

	log.Printf("listening on port: %s", cfg.port)
	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					log.Println("connection has been closed")
					return
				}
				log.Fatalf("error accepting connection: %s", err)
			}
			log.Printf("connected to: %s", conn.RemoteAddr())
			go basicReadLoop(cfg, conn)
		}
	}()
	<-cfg.quit
	return nil
}

func basicReadLoop(cfg *config, conn net.Conn) {
	defer func() { _ = conn.Close() }()
	reader := bufio.NewReader(conn)
	for {
		request, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Printf("conn closed: %s", conn.RemoteAddr())
				return
			}
			log.Printf("error reading from connection: %s", err)
			continue
		}
		request = bytes.TrimSpace(request)

		log.Printf("received: %s", request)
		switch {
		case bytes.Equal(request, []byte("ping")):
			_, _ = conn.Write([]byte("pong\n"))
			continue
		case bytes.Equal(request, []byte("quit")):
			cfg.quit <- struct{}{}
			return
		}
		_, _ = conn.Write(fmt.Appendf(nil, "echo: %s\n", request))
	}
}
