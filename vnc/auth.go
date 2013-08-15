package vnc

// A Auth implements a method of authenticating with a remote server or client.
type AuthType interface {
	// Type returns the byte identifier sent by the server or client to
	// identify this authentication scheme.
	Type() uint8

	// Handshake is called when the authentication handshake should be
	// performed, as part of the general RFB handshake. (see 7.1.2)
	Handler(*ServerConn) error
}

// AuthNone is the "none" authentication. See 7.1.2
type AuthTypeNone byte

func (*AuthTypeNone) Type() uint8 {
	return 1
}

func (*AuthTypeNone) Handler(*ServerConn) error {
	return nil
}
