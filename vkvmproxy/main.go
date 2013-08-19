package main

import (
	"./vnc"
	"log"
	"net"
	"sync"
)

type Proxy struct {
	sync.Mutex
	Targets map[*vnc.Conn]*rConn
}

type rConn struct {
	HostPort string
	Password []byte
	c        *vnc.Conn
}

func (c *rConn) Close() {
	c.c.Close()
	return
}

func main() {
	var p Proxy
	p.Targets = make(map[*vnc.Conn]*rConn)
	l, err := net.Listen("tcp", "127.0.0.1:6900")
	if err != nil {
		log.Fatalf(err.Error())
	}

	s := vnc.NewServer(&vnc.ServerConfig{})

	go func() {
		err = s.Serve(l)
		log.Fatalf("vnc proxy ended with: %s", err.Error())
	}()

}

func (p *Proxy) handleConn(lc *vnc.Conn) {
	var rc *rConn
	p.Lock()
	rc, ok := p.Targets[lc]
	p.Unlock()
	if !ok {
		return
	}
	defer rc.Close()
	defer lc.Close()
	for {
		select {
		case msg := <-lc.MessageSrv:
			log.Printf("%+v\n", msg)

		}
	}
}
