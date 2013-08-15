package vnc

import (
	"bytes"
	"encoding/binary"
)

func serverInitDefault(c *ServerConn) (err error) {
	var sharedFlag uint8
	if err = binary.Read(c.c, binary.BigEndian, &sharedFlag); err != nil {
		return err
	}
	_ = sharedFlag

	buffer := new(bytes.Buffer)
	// 7.3.2 ServerInit
	if err = binary.Write(buffer, binary.BigEndian, c.s.config.Width); err != nil {
		return err
	}

	if err = binary.Write(buffer, binary.BigEndian, c.s.config.Height); err != nil {
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

	return nil
}
