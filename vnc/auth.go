package vnc

import (
	"io"
)

type AuthType interface {
	Type() uint8
	Handler(*Conn, io.ReadWriter) error
}

const (
	authInvalid uint8 = iota
	authNone
	authVNC
)

const (
	authSuccess = iota
	authFailure
)

type AuthTypeNone byte

func (*AuthTypeNone) Type() uint8 {
	return authNone
}

func (*AuthTypeNone) Handler(*Conn, io.ReadWriter) error {
	return nil
}
