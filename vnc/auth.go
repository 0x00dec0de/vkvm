package vnc

import (
	"net"
)

// A Auth implements a method of authenticating with a remote server or client.
type Auth interface {
	// SecurityType returns the byte identifier sent by the server or client to
	// identify this authentication scheme.
	SecurityType() uint8

	// Handshake is called when the authentication handshake should be
	// performed, as part of the general RFB handshake. (see 7.1.2)
	Handshake(net.Conn) error
}

// AuthNone is the "none" authentication. See 7.1.2
type AuthNone byte

func (*AuthNone) SecurityType() uint8 {
	return 1
}

func (*AuthNone) Handshake(net.Conn) error {
	return nil
}
