package main

import (
	"code.google.com/p/go.net/websocket"
	_ "flag"
	_ "io"
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
	l, err := net.Listen("tcp", "127.0.0.1:5910")
	if err != nil {
		log.Fatalf(err.Error())
	}
	srv = NewServer()

	go srv.Serve(l)

	go func() {
		http.Handle("/", websocket.Handler(VNCServer))
		err = http.ListenAndServe("127.0.0.2:8080", nil)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}()

	for c := range srv.Conns {
		handleConn(c)
	}
	select {}
}

func handleConn(sc *Conn) {
	var wg sync.WaitGroup
	var cc *Conn
	var ok bool
	p.Lock()
	if cc, ok = p.Targets[sc]; !ok {
		p.Unlock()
		return
	}
	p.Unlock()
	defer sc.Close()
	defer cc.Close()
	go func() {
		wg.Add(1)
		defer wg.Done()
		for {
			select {
			case <-sc.MsgClose:
				p.Lock()
				delete(p.Targets, sc)
				p.Unlock()
				return
			}
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		for {
			select {
			case <-cc.MsgClose:
				p.Lock()
				delete(p.Targets, sc)
				p.Unlock()
				return
			}
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		for {
			select {
			case buf := <-sc.MsgChan:
				if n, err := cc.Write(buf); err != nil {
					log.Printf("sc->cc: %d %s\n", n, err.Error())
					return
				} else {
					log.Printf("sc->cc: parsed msg len: %d\n", n)
					sc.MsgDone <- true
				}
			}
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		for {
			select {
			case buf := <-cc.MsgChan:
				if n, err := sc.Write(buf); err != nil {
					log.Printf("cc->sc: %d %s\n", n, err.Error())
					return
				} else {
					log.Printf("cc->sc: parsed msg len: %d\n", n)
					cc.MsgDone <- true
				}
			}
		}
	}()

	wg.Wait()
}
