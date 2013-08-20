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
	c         *vnc.Conn
	challenge [16]byte
	password  []byte
}

var p Proxy

func main() {
	p.Targets = make(map[*vnc.Conn]*rConn)

	l, err := net.Listen("tcp", "127.0.0.1:6900")
	if err != nil {
		log.Fatalf(err.Error())
	}

	s := vnc.NewServer(&vnc.ServerConfig{AuthTypes: []vnc.AuthType{new(ServerAuthTypeVNC)}})
	go func() {
		err = s.Serve(l)
		log.Fatalf("vnc proxy ended with: %s", err.Error())
	}()

	for c := range s.Conns {
		p.handleConn(c)
	}

}

func (p *Proxy) handleConn(lc *vnc.Conn) {
	p.Lock()
	c, ok := p.Targets[lc]
	p.Unlock()
	if !ok {
		return
	}
	rc := c.c
	for {
		select {
		case msg := <-lc.MessageCli:
			//			log.Printf("%+v\n", msg)
			rc.MessageCli <- msg
		case msg := <-rc.MessageSrv:
			//		log.Printf("%+v\n", msg)
			lc.MessageSrv <- msg
		}
	}
}
