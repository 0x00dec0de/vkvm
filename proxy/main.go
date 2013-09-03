package main

import (
	"io"
	"log"
	"net"
	"sync"
)

type Proxy struct {
	sync.Mutex
	Targets map[*Conn]*Conn
}

var p Proxy

func main() {
	p.Targets = make(map[*Conn]*Conn, 1024)
	l, err := net.Listen("tcp", "127.0.0.2:5900")
	if err != nil {
		log.Fatalf(err.Error())
	}
	s := NewServer()

	go s.Serve(l)

	for c := range s.Conns {
		handleConn(c)
	}

}

func handleConn(sc *Conn) {
	var cc *Conn
	var ok bool
	p.Lock()
	if cc, ok = p.Targets[sc]; !ok {
		p.Unlock()
		return
	}
	p.Unlock()
	go func() {
		for {
			log.Printf("AAA\n")
			select {
			case <-sc.MsgChan:
				if n, err := io.Copy(cc, sc); err != nil {
					log.Printf("sc->cc: %d %s\n", n, err.Error())
					return
				}
				sc.MsgDone <- true
			}
		}
	}()

	go func() {
		for {
			select {
			case <-cc.MsgChan:
				if n, err := io.Copy(sc, cc); err != nil {
					log.Printf("cc->sc: %d %s\n", n, err.Error())
					return
				}
				cc.MsgDone <- true
			}
		}
	}()
	select {}
}
