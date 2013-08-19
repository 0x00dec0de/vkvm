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
	*vnc.Conn
	challenge [16]byte
}

func main() {
	var p Proxy
	p.Targets = make(map[*vnc.Conn]*rConn)

	/*
		///
		d, err := net.Dial("tcp", "127.0.0.1:5900")
		if err != nil {
			log.Fatalf(err.Error())
		}

		c := vnc.NewClient(&vnc.ClientConfig{AuthTypes: []vnc.AuthType{new(ClientAuthTypeVNC)}})
		go func() {
			err = c.Serve(d)
			if err != nil {
				log.Fatalf("vnc proxy ended with: %s", err.Error())
			}
		}()
		///
	*/
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
	log.Printf("conn: %+v\n", lc)
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
