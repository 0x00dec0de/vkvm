package main

import (
	"log"
	"net"
	"sync"
)

type Proxy struct {
	conns chan *lConn
	Conns <-chan *lConn
	sync.Mutex
	Targets map[*lConn]*rConn
}

type lConn struct {
	c net.Conn
}

type rConn struct {
	HostPort string
	Password []byte
	c        net.Conn
}

func (p *Proxy) newlConn(c net.Conn) *lConn {
	return &lConn{
		c: c,
	}
}

func (p *lConn) Close() error {
	return p.c.Close()
}

func main() {
	var p Proxy
	p.Targets = make(map[*lConn]*rConn)
	l, err := net.Listen("tcp", "127.0.0.1:6900")
	if err != nil {
		log.Fatalf(err.Error())
	}

	go func() {
		err = p.Serve(l)
		log.Fatalf("vnc proxy ended with: %s", err.Error())
	}()

	for c := range p.Conns {
		p.handleConn(c)
	}

}

func (p *Proxy) handleConn(c *lConn) {
	log.Printf("%+v\n", c)
}

func (p *Proxy) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		conn := p.newlConn(c)
		if err := p.lHandshake(conn); err != nil {
			conn.Close()
			log.Printf(err.Error())
		}
		select {
		case p.conns <- conn:
		default:
			// client is behind; doesn't get this updated.
		}
		//		go conn.serve()
	}
	panic("unreachable")
}
