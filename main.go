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

//type Proxy map[*vnc.ServerConn]*vnc.ClientConn

type ClientAuthVNC byte

func (*ClientAuthVNC) Type() uint8 {
	return 2
}

func (*ClientAuthVNC) Handler(c *vnc.ServerConn) (err error) {
	challenge := make([]uint8, 16)

	if err := binary.Read(c.c, binary.BigEndian, &challenge); err != nil {
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
	if err = binary.Write(c.c, binary.BigEndian, response); err != nil {
		return err
	}
	return nil
}

func main() {

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

func handleConn(c *vnc.ServerConn) {
	var err error
	for {
		select {
		case msg := <-c.MessageCh:
			switch reflect.TypeOf(msg).String() {
			case "*vnc.CloseMessage":
				return
			case "*vnc.AuthMessage":
				auth := msg.(*vnc.AuthMessage)
				if err = auth.Handshake(c.c); err != nil {
					if err = binary.Write(c.c, binary.BigEndian, uint32(1)); err != nil {
						return err
					}
					c.writeErrorReason(err)
					return
				}
				// 7.1.3 SecurityResult Handshake
				if err = binary.Write(c.c, binary.BigEndian, uint32(0)); err != nil {
					return
				}
			case "*vnc.InitMessage":
				var sharedFlag uint8
				if c.config.Exclusive {
					sharedFlag = 0
				}

				if err = binary.Read(c.c, binary.BigEndian, &sharedFlag); err != nil {
					return err
				}

				buffer := new(bytes.Buffer)
				// 7.3.2 ServerInit
				if err = binary.Write(buffer, binary.BigEndian, c.config.Width); err != nil {
					return err
				}

				if err = binary.Write(buffer, binary.BigEndian, c.config.Height); err != nil {
					return err
				}

				// Write the pixel format
				var format []byte
				if format, err = writePixelFormat(&c.PixelFormat); err != nil {
					return err
				}
				if err = binary.Write(buffer, binary.BigEndian, format); err != nil {
					return err
				}

				padding := []uint8{0, 0, 0}
				if err = binary.Write(buffer, binary.BigEndian, padding); err != nil {
					return err
				}

				nameBytes := []uint8(c.DesktopName)
				nameLen := uint8(cap(nameBytes))
				if err = binary.Write(buffer, binary.BigEndian, nameLen); err != nil {
					return err
				}

				if err = binary.Write(buffer, binary.BigEndian, nameBytes); err != nil {
					return err
				}

				if err = binary.Write(c.c, binary.BigEndian, buffer.Bytes()); err != nil {
					return err
				}

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
				s.FramebufferUpdate(msg)
			default:
				log.Printf("%s\n", reflect.TypeOf(msg).String())
			}
		}
	}

}
