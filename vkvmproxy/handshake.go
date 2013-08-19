package main

/*
import (
	"./vnc"
	"bytes"
	"crypto/des"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

func (p *Proxy) lHandler(c *vnc.Conn, rw io.ReadWriter) (err error) {
	buffer := new(bytes.Buffer)
	var protocolVersion [12]byte
	if _, err := io.Write(rw, []byte("RFB 003.008\n")); err != nil {
		return err
	}

	// 7.1.1, read the ProtocolVersion message sent by the server.
	if _, err := io.ReadFull(rw, protocolVersion[:]); err != nil {
		return err
	}

	var maxMajor, maxMinor uint8
	_, err = fmt.Sscanf(string(protocolVersion[:]), "RFB %d.%d\n", &maxMajor, &maxMinor)
	if err != nil {
		return err
	}

	if maxMajor < 3 {
		return fmt.Errorf("unsupported major version, less than 3: %d", maxMajor)
	}

	if maxMinor < 8 {
		return fmt.Errorf("unsupported minor version, less than 8: %d", maxMinor)
	}

	if err := binary.Write(buffer, binary.BigEndian, uint8(1)); err != nil {
		return err
	}

	if err := binary.Write(buffer, binary.BigEndian, uint8(2)); err != nil {
		return err
	}

	if err := binary.Write(rw, binary.BigEndian, buffer.Bytes()); err != nil {
		return err
	}

	buffer.Reset()

	var auth uint8
	if err := binary.Read(rw, binary.BigEndian, &auth); err != nil {
		return err
	}
	if auth != 2 {
		return fmt.Errorf("failed auth")
	}

	challenge := make([]uint8, 16)
	response := make([]uint8, 16)

	_, err = rand.Read(challenge)
	if err != nil {
		return err
	}
	if err := binary.Write(rw, binary.BigEndian, challenge); err != nil {
		return err
	}

	if err := binary.Read(rw, binary.BigEndian, &response); err != nil {
		return err
	}

	rconn := &rConn{HostPort: "127.0.0.1:5900", Password: []byte("njkcnjd")}
	err = p.rHandshake(rconn, rw)
	if err != nil {
		return err
	}

	if err := binary.Write(rw, binary.BigEndian, uint32(0)); err != nil {
		return err
	}

	p.Lock()
	p.Targets[c] = rconn
	p.Unlock()

	return nil
}

func (p *Proxy) rHandshake(rc *rConn, rw io.ReadWriter) (err error) {

	c, err := net.Dial("tcp", rc.HostPort)
	if err != nil {
		return err
	}
	rc.c = c

	var protocolVersion [12]byte

	// 7.1.1, read the ProtocolVersion message sent by the server.
	if _, err := io.ReadFull(rc.c, protocolVersion[:]); err != nil {
		return err
	}

	var maxMajor, maxMinor uint8
	_, err = fmt.Sscanf(string(protocolVersion[:]), "RFB %d.%d\n", &maxMajor, &maxMinor)
	if err != nil {
		return err
	}

	if maxMajor < 3 {
		return fmt.Errorf("unsupported major version, less than 3: %d", maxMajor)
	}

	if maxMinor < 8 {
		return fmt.Errorf("unsupported minor version, less than 8: %d", maxMinor)
	}

	// Respond with the version we will support
	if _, err = rc.c.Write([]byte("RFB 003.008\n")); err != nil {
		return err
	}

	// 7.1.2 Security Handshake from server
	var numSecurityTypes uint8
	if err = binary.Read(rc.c, binary.BigEndian, &numSecurityTypes); err != nil {
		return err
	}

	securityTypes := make([]uint8, numSecurityTypes)
	if err = binary.Read(rc.c, binary.BigEndian, &securityTypes); err != nil {
		return err
	}

	if err = binary.Write(rc.c, binary.BigEndian, uint8(2)); err != nil {
		return err
	}

	challenge := make([]uint8, 16)

	if err := binary.Read(rc.c, binary.BigEndian, &challenge); err != nil {
		return err
	}
	pwd := rc.Password
	if len(pwd) > 8 {
		pwd = pwd[:8]
	}
	if len(pwd) < 8 {
		if x := len(pwd); x < 8 {
			for i := 8 - x; i > 0; i-- {
				pwd = append(pwd, byte(0))
			}
		}
	}

	newpwd := make([]byte, 8)
	for i := 0; i < 8; i++ {
		c := pwd[i]
		c = ((c & 0x01) << 7) + ((c & 0x02) << 5) + ((c & 0x04) << 3) + ((c & 0x08) << 1) +
			((c & 0x10) >> 1) + ((c & 0x20) >> 3) + ((c & 0x40) >> 5) + ((c & 0x80) >> 7)
		newpwd[i] = c
	}

	enc, err := des.NewCipher(newpwd)
	if err != nil {
		return err
	}
	response := make([]byte, 16)

	enc.Encrypt(response[:8], challenge[:8])
	enc.Encrypt(response[8:], challenge[8:])
	if err = binary.Write(rc.c, binary.BigEndian, response); err != nil {
		return err
	}

	var ok uint8
	if err = binary.Read(rc.c, binary.BigEndian, &ok); err != nil {
		return err
	}

	return nil
}
*/
