package vnc

import (
	"bufio"
	"io"
)

type Message interface {
	Type() uint8
	Read(*Conn, io.Reader, bufio.Reader) (*Message, error)
	Write(*Conn, io.Writer, bufio.Writer) ([]byte, error)
}
