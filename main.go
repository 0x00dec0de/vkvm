package main

import (
	"./vnc"
	"crypto/des"
	"crypto/rand"
	"encoding/binary"
	"log"
	"net"
	"reflect"
)

type ClientAuthVNC byte

func (*ClientAuthVNC) SecurityType() uint8 {
	return 2
}

func (*ClientAuthVNC) Handshake(c net.Conn) error {
	challenge := make([]uint8, 16)

	if err := binary.Read(c, binary.BigEndian, &challenge); err != nil {
		return err
	}

	_, err := rand.Read(challenge)
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
	if err = binary.Write(c, binary.BigEndian, response); err != nil {
		return err
	}
	return nil
}

func main() {

	l, err := net.Listen("tcp", "127.0.0.1:6900")
	if err != nil {
		log.Fatal(err)
	}

	r, err := net.Dial("tcp", "127.0.0.1:5900")
	if err != nil {
		log.Fatal(err)
	}

	msgR := make(chan vnc.ServerMessage, 1)
	c, err := vnc.Client(r, &vnc.ClientConfig{ServerMessageCh: msgR})
	if err != nil {
		log.Fatal(err)
	}
	//	log.Printf("%+v\n", c)
	_ = c
	msgL := make(chan vnc.ClientMessage, 1)
	format := vnc.PixelFormat{BPP: 32, Depth: 24, BigEndian: false, TrueColor: true, RedMax: 255, GreenMax: 255, BlueMax: 255, RedShift: 16, GreenShift: 8, BlueShift: 0}

	go func() {
		err = vnc.NewServer(l, &vnc.ServerConfig{PixelFormat: format, FrameBufferWidth: uint16(640), FrameBufferHeight: uint16(480), DesktopName: "QEMU (devstack)", ClientMessageCh: msgL})
		if err != nil {
			log.Fatal(err)
		}
	}()
	//	defer s.Close()
	for {
		select {
		case msg := <-msgL:
			switch reflect.TypeOf(msg).String() {
			case "*vnc.SetPixelFormatMessage":
				format := msg.(*vnc.SetPixelFormatMessage)
				err := c.SetPixelFormat(&format.PixelFormat)
				if err != nil {
					log.Printf(err.Error())
				}
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

			default:
				log.Printf("%s\n", reflect.TypeOf(msg).String())
			}
		}
	}

}
