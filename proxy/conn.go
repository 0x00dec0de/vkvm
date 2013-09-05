package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
)

type Conn struct {
	sc net.Conn
	cc net.Conn

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

	MsgChan  chan byte
	MsgDone  chan bool
	InitDone chan bool
}

func (c *Conn) Read(b []byte) (n int, err error) {
	var m []byte
	if m, err = c.br.Peek(1); err != nil {
		return
	}
	if c.sc != nil {
		switch m[0] {
		case 0:
			buf := make([]byte, 20)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			return
		case 2:
			var mn []byte
			var nEncs uint16
			if mn, err = c.br.Peek(4); err != nil {
				return
			}
			nEncs = binary.BigEndian.Uint16(mn[2:])
			buf := make([]byte, 4+4*nEncs)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			bbuf := new(bytes.Buffer)
			binary.Write(bbuf, binary.BigEndian, []uint8{uint8(2), uint8(0)})
			binary.Write(bbuf, binary.BigEndian, uint16(1))
			binary.Write(bbuf, binary.BigEndian, int32(0))
			n = copy(b, bbuf.Bytes())
			return
		case 3:
			buf := make([]byte, 10)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			return
		case 4:
			buf := make([]byte, 8)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			return
		case 5:
			buf := make([]byte, 6)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			return
		case 6:
			var mn []byte
			if mn, err = c.br.Peek(8); err != nil {
				return
			}
			var Len uint32
			Len = binary.BigEndian.Uint32(mn[4:8])
			buf := make([]byte, 8+Len)
			n, err = io.ReadFull(c.br, buf)
			copy(b, buf)
			return
		}
	}
	if c.cc != nil {
		switch m[0] {
		case 0:
			var Hdr struct {
				Type   uint8
				Pad    uint8
				Nrects uint16
			}
			var Rect struct {
				X   uint16
				Y   uint16
				W   uint16
				H   uint16
				Enc int32
			}
			buf := new(bytes.Buffer)
			binary.Read(c.cc, binary.BigEndian, &Hdr)
			binary.Write(buf, binary.BigEndian, Hdr)
			//var byteOrder binary.ByteOrder = binary.LittleEndian
			//			if c.PixelFormat.BigEndian == 1 {
			//			byteOrder = binary.BigEndian
			//	}
			log.Printf("hdr: %+v\n", Hdr)
			for i := uint16(0); i < Hdr.Nrects; i++ {
				binary.Read(c.cc, binary.BigEndian, &Rect)
				binary.Write(buf, binary.BigEndian, Rect)
				log.Printf("rect: %+v\n", Rect)
				bb := make([]byte, Rect.W*Rect.H*uint16(c.PixelFormat.Bpp/8))
				if _, err = io.ReadFull(c.cc, bb); err != nil {
					return
				}
				buf.Write(bb)
			}
			n = copy(b, buf.Bytes())
			buf.Reset()
			return
		case 1:
			var mn []byte
			var nColors uint16
			if mn, err = c.br.Peek(6); err != nil {
				return
			}
			nColors = binary.BigEndian.Uint16(mn[4:])
			buf := make([]byte, 6+nColors*6)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			return
		case 2:
			buf := make([]byte, 1)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			return
		case 3:
			var mn []byte
			var Len uint32
			if mn, err = c.br.Peek(6); err != nil {
				return
			}
			Len = binary.BigEndian.Uint32(mn[4:])
			buf := make([]byte, 8+Len)
			n, err = io.ReadFull(c.br, buf)
			n = copy(b, buf)
			return
		}
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	if len(b) > 4096 {
		c.bw = bufio.NewWriterSize(c.bw, len(b))
	}
	n, err = c.bw.Write(b)
	c.bw.Flush()
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
