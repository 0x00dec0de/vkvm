package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
)

type Conn struct {
	smu sync.Mutex
	cmu sync.Mutex
	sc  net.Conn
	cc  net.Conn

	Major uint8
	Minor uint8

	Width  uint16
	Height uint16

	SharedFlag uint8

	Name string

	PixelFormat PixelFormat

	br *bufio.Reader
	bw *bufio.Writer

	err error

	scHostPort string
	ccHostPort string
	Password   string

	MsgChan  chan []byte
	MsgDone  chan bool
	InitDone chan bool
}

func (c *Conn) Read() ([]byte, error) {
	var err error
	bbuf := new(bytes.Buffer)
	defer bbuf.Reset()
	buf := new(bytes.Buffer)
	defer buf.Reset()
	var m byte
	if c.sc != nil {
		//		log.Printf("c.smu.Lock()\n")
		//		c.smu.Lock()
		//		log.Printf("c.smu.Unlock()\n")
		//		defer c.smu.Unlock()
		if err = binary.Read(c.sc, binary.BigEndian, &m); err != nil {
			log.Printf(err.Error())
			return nil, err
		}
		bbuf.Write([]byte{m})
		log.Printf("SC: %+v\n", m)
		switch m {
		case 0:
			_, err = io.CopyN(buf, c.sc, 19)
			bbuf.Write(buf.Bytes())
		case 2:
			mn := make([]byte, 3)
			if _, err = io.ReadFull(c.sc, mn); err != nil {
				return nil, err
			}
			_, err = io.CopyN(ioutil.Discard, c.sc, int64(4*binary.BigEndian.Uint16(mn[1:])))
			binary.Write(bbuf, binary.BigEndian, uint8(0))
			binary.Write(bbuf, binary.BigEndian, uint16(1))
			binary.Write(bbuf, binary.BigEndian, int32(0))
		case 3:
			_, err = io.CopyN(buf, c.sc, 9)
			bbuf.Write(buf.Bytes())
		case 4:
			_, err = io.CopyN(buf, c.sc, 7)
			bbuf.Write(buf.Bytes())
		case 5:
			_, err = io.CopyN(buf, c.sc, 5)
			bbuf.Write(buf.Bytes())
		case 6:
			mn := make([]byte, 7)
			if _, err = io.ReadFull(c.sc, mn); err != nil {
				return nil, err
			}
			bbuf.Write(mn)
			_, err = io.CopyN(buf, c.sc, int64(binary.BigEndian.Uint32(mn[3:])))
			bbuf.Write(buf.Bytes())
		}
	}
	if c.cc != nil {
		//		log.Printf("c.cmu.Lock()\n")
		//		c.cmu.Lock()
		//		log.Printf("c.cmu.Unlock()\n")
		//		defer c.cmu.Unlock()
		if err = binary.Read(c.cc, binary.BigEndian, &m); err != nil {
			return nil, err
		}
		log.Printf("CC: %+v\n", m)
		bbuf.Write([]byte{m})
		switch m {
		case 0:
			log.Printf("cc: framebufferupdate\n")
			var Hdr struct {
				Pad    uint8
				Nrects uint16
			}
			var Rect struct {
				X    uint16
				Y    uint16
				W    uint16
				H    uint16
				Type int32
			}
			binary.Read(c.cc, binary.BigEndian, &Hdr)
			binary.Write(bbuf, binary.BigEndian, Hdr)
			log.Printf("nrects: %d\n", int(Hdr.Nrects))
			for i := uint16(0); i < Hdr.Nrects; i++ {
				log.Printf("Rect :%+v\n", Rect)

				binary.Read(c.cc, binary.BigEndian, &Rect)
				binary.Write(bbuf, binary.BigEndian, Rect)
				if Rect.W*Rect.H == uint16(0) {
					//					Rect.W = c.Width
					//				Rect.H = c.Height
					continue
				}

				bytesPerLine := Rect.W * uint16(c.PixelFormat.Bpp/8)
				linesToRead := Rect.W * Rect.H / bytesPerLine
				var bbb int64
				for n := Rect.H; n > 0; n -= linesToRead {
					if linesToRead > n {
						linesToRead = n
					}
					bbb += int64(bytesPerLine * linesToRead)
				}

				if _, err := io.CopyN(buf, c.cc, bbb); err != nil {
					return nil, err
				}
				bbuf.Write(buf.Bytes())
			}
		case 1:
			mn := make([]byte, 5)
			if _, err = io.ReadFull(c.cc, mn); err != nil {
				return nil, err
			}
			bbuf.Write(mn)
			_, err = io.CopyN(buf, c.cc, int64(binary.BigEndian.Uint16(mn[3:])*uint16(6)))
			bbuf.Write(buf.Bytes())
		case 2:
			_, err = io.CopyN(buf, c.cc, 1)
			bbuf.Write(buf.Bytes())
		case 3:
			mn := make([]byte, 7)
			if _, err = io.ReadFull(c.cc, mn); err != nil {
				return nil, err
			}
			_, err = io.CopyN(buf, c.cc, int64(binary.BigEndian.Uint32(mn[3:])))
			bbuf.Write(buf.Bytes())
		}
	}
	return bbuf.Bytes(), err
}

func (c *Conn) Write(b []byte) (n int, err error) {
	if c.cc != nil {
		log.Printf("CC %v\n", b)
		//	c.cmu.Lock()
		//		defer c.cmu.Unlock()
		n, err = c.cc.Write(b)
		log.Printf("cc: %d %d\n", n, len(b))
		//		c.cmu.Unlock()
		return
	}
	if c.sc != nil {
		log.Printf("SC %d\n", b[0])
		//	c.smu.Lock()
		//defer c.smu.Unlock()
		n, err = c.sc.Write(b)
		return
	}
	return
}

func (c *Conn) Close() error {
	if c.sc != nil {
		c.sc.Close()
	}
	if c.cc != nil {
		c.cc.Close()
	}
	return nil
}
