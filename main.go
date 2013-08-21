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

	l, err := net.Listen("tcp", "127.0.0.2:5900")
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
	//	defer lc.Close()
	p.Lock()
	c, ok := p.Targets[lc]
	p.Unlock()
	if !ok {
		return
	}
	rc := c.c
	//defer rc.Close()
	for {
		select {
		case msg := <-lc.MessageCli:
			rc.MessageCli <- msg
			break
		case msg := <-rc.MessageSrv:
			lc.MessageSrv <- msg
			break
		case <-rc.Quit:
			lc.Close()
			return
		case <-lc.Quit:
			rc.Close()
			return
		}
	}
}
