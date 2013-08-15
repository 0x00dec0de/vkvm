package main

import (
	"./vnc"
	"crypto/des"
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

type Node struct {
	HostPort string
	Password []byte
	Rvnc     *vnc.ClientConn
}

type Proxy struct {
	sync.Mutex
	Hosts map[*vnc.ServerConn]*Node
}

var proxy Proxy

type ClientAuthVNC byte

func (*ClientAuthVNC) Type() uint8 {
	return 2
}

func (*ClientAuthVNC) Handler(c *vnc.ServerConn, rw io.ReadWriter) (err error) {
	challenge := make([]uint8, 16)

	if err := binary.Read(rw, binary.BigEndian, &challenge); err != nil {
		return err
	}

	_, err = rand.Read(challenge)
	if err != nil {
		return err
	}

	pwd := []byte("njkcnjd")
	if len(pwd) > 8 {
		pwd = pwd[:8]
	}
	if x := len(pwd); x < 8 {
		for i := 8 - x; i > 0; i-- {
			pwd = append(pwd, byte(0))
		}

	}

	enc, err := des.NewCipher(pwd)
	if err != nil {
		return err
	}
	response := make([]byte, 16)
	enc.Encrypt(response, challenge)
	if err = binary.Write(rw, binary.BigEndian, response); err != nil {
		return err
	}

	nc, err := net.Dial("tcp", "127.0.0.1:5900")
	if err != nil {
		return err
	}
	rvnc, err := vnc.Client(nc, vnc.ClientConfig{})
	if err != nil {
		return err
	}

	proxy.Lock()
	proxy.Hosts[c] = &Node{Rvnc: rvnc, HostPort: "127.0.0.1:5900", Password: pwd}
	proxy.Unlock()
	return nil
}

func main() {
	proxy.Hosts = make(map[*vnc.ServerConn]*Node)

	l, err := net.Listen("tcp", "127.0.0.1:6900")
	if err != nil {
		log.Fatal(err)
	}

	s := vnc.NewServer(&vnc.ServerConfig{Width: 640, Height: 480, DesktopName: "vkvm"})
	s.Serve(l)

	go func() {
		err = s.Serve(l)
		log.Fatalf("vnc server ended with: %v", err)
	}()
	for c := range s.Conns {
		handleConn(c)
	}
}

func handleConn(lvnc *vnc.ServerConn) {
	//	var err error
	var rvnc *vnc.ClientConn

	if p, ok := proxy.Host[lvnc]; ok {
		rvnc = p.Rvnc
	}

	go func() {
		for {
			select {
			case msg := <-lvnc.MessageInp:
				switch reflect.TypeOf(msg).String() {
				case "*vnc.SetPixelFormatMessage":
					format := msg.(*vnc.SetPixelFormatMessage)
					log.Printf("%+v\n", msg)
					rvnc.MessageOut <- SetPixelFormat(format.PixelFormat)
				}
			}
		}
	}()
	select {}
}

/*
	go func() {
		for {
			select {
			case msg := <-rvnc.MessageInp:


/*
		case "*vnc.SetEncodingsMessage":
			encs := msg.(*vnc.SetEncodingsMessage)
			err := c.SetEncodings(encs.Encs)
			if err != nil {
				log.Printf(err.Error())
			}
		case "*vnc.FramebufferUpdateRequestMessage":
			request := msg.(*vnc.FramebufferUpdateRequestMessage)
			incremental := false
			if request.Incremental == 1 {
				incremental = true
			}
			err := c.FramebufferUpdateRequest(incremental, request.X, request.Y, request.Width, request.Height)
			if err != nil {
				log.Printf(err.Error())
			}
		case "*vnc.PointerEventMessage":
		}
	case msg := <-msgR:
		switch reflect.TypeOf(msg).String() {
		case "*vnc.FramebufferUpdateMessage":
			s.FramebufferUpdate(msg)
		default:
			log.Printf("%s\n", reflect.TypeOf(msg).String())
		}
	}
*/
