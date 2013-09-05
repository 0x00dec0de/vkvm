package main

import (
	"code.google.com/p/go.net/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

type Proxy struct {
	sync.Mutex
	Targets map[*Conn]*Conn
}

var p Proxy

var srv *Server

func VNCServer(c *websocket.Conn) {
	srv.addConn(c)
}

func main() {
	p.Targets = make(map[*Conn]*Conn, 1024)
	l, err := net.Listen("tcp", "127.0.0.2:5900")
	if err != nil {
		log.Fatalf(err.Error())
	}
	srv = NewServer()

	go srv.Serve(l)

	http.Handle("/", websocket.Handler(VNCServer))
	err = http.ListenAndServe(":5901", nil)
	if err != nil {
		log.Fatalf(err.Error())
	}

	for c := range srv.Conns {
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
			select {
			case <-cc.MsgChan:
				if n, err := io.Copy(sc, cc); err != nil {
					log.Printf("cc->sc: %d %s\n", n, err.Error())
					return
				} else {
					log.Printf("parsed msg len: %d\n", n)
					sc.MsgDone <- true

				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-sc.MsgChan:
				if n, err := io.Copy(cc, sc); err != nil {
					log.Printf("sc->cc: %d %s\n", n, err.Error())
					return
				} else {
					log.Printf("parsed msg len: %d\n", n)
					cc.MsgDone <- true
				}
			}
		}
	}()
	select {}
}
