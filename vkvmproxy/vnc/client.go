package vnc

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Client struct {
	c *ClientConfig
	//	HostPort string
	//	Password []byte
}

type ClientConfig struct {
	AuthTypes []AuthType
	MaxMsg    int

	Width  int
	Height int

	Messages []Message
}

func NewClient(cfg *ClientConfig) *Client {
	if cfg.AuthTypes == nil {
		cfg.AuthTypes = []AuthType{new(AuthTypeNone)}
	}
	return &Client{
		c: cfg,
	}
}

func (cli *Client) Serve(c net.Conn) (*Conn, error) {
	var err error
	conn := cli.newConn(&c)
	if err = conn.clientVersionHandshake(); err != nil {
		return nil, err
	}
	if err = conn.clientSecurityHandshake(); err != nil {
		return nil, err
	}
	if err = conn.clientInit(); err != nil {
		return nil, err
	}
	go conn.clientServe()
	return conn, err
}

func (c *Conn) clientVersionHandshake() error {
	var protocolVersion [12]byte

	if err := binary.Read(*c.c, binary.BigEndian, &protocolVersion); err != nil {
		return err
	}

	var maxMajor, maxMinor uint8
	_, err := fmt.Sscanf(string(protocolVersion[:]), "RFB %d.%d\n", &maxMajor, &maxMinor)
	if err != nil {
		return err
	}

	if maxMajor < 3 {
		return fmt.Errorf("unsupported major version, less than 3: %d", maxMajor)
	}

	if maxMinor < 8 {
		return fmt.Errorf("unsupported minor version, less than 8: %d", maxMinor)
	}

	if err = binary.Write(*c.c, binary.BigEndian, []byte("RFB 003.008\n")); err != nil {
		return err
	}
	return nil
}

func (c *Conn) clientSecurityHandshake() error {
	var err error
	var numSecurityTypes uint8
	if err = binary.Read(*c.c, binary.BigEndian, &numSecurityTypes); err != nil {
		return err
	}

	if numSecurityTypes == 0 {
		var reasonLength uint32
		if err = binary.Read(*c.c, binary.BigEndian, &reasonLength); err != nil {
			return err
		}
		reasonText := make([]byte, reasonLength)
		if err = binary.Read(*c.c, binary.BigEndian, &reasonText); err != nil {
			return err
		}
		return fmt.Errorf("no security types: %s", reasonText)
	}

	securityTypes := make([]uint8, numSecurityTypes)
	if err = binary.Read(*c.c, binary.BigEndian, &securityTypes); err != nil {
		return err
	}

	clientSecurityTypes := c.cli.c.AuthTypes

	var auth AuthType
FindAuth:
	for _, curAuth := range clientSecurityTypes {
		for _, securityType := range securityTypes {
			if curAuth.Type() == securityType {
				// We use the first matching supported authentication
				auth = curAuth
				break FindAuth
			}
		}
	}

	if auth == nil {
		return fmt.Errorf("no suitable auth schemes found. server supported: %#v", securityTypes)
	}

	// Respond back with the security type we'll use
	if err = binary.Write(*c.c, binary.BigEndian, auth.Type()); err != nil {
		return err
	}

	if err = auth.Handler(c, *c.c); err != nil {
		return err
	}

	// 7.1.3 SecurityResult Handshake
	var securityResult uint32
	if err = binary.Read(*c.c, binary.BigEndian, &securityResult); err != nil {
		return err
	}

	if securityResult == 1 {
		var reasonLength uint32
		if err = binary.Read(*c.c, binary.BigEndian, &reasonLength); err != nil {
			return err
		}
		reasonText := make([]byte, reasonLength)
		if err = binary.Read(*c.c, binary.BigEndian, &reasonText); err != nil {
			return err
		}
		return fmt.Errorf("security handshake failed: %s", reasonText)
	}
	return nil
}

func (c *Conn) clientInit() error {
	var err error
	var sharedFlag uint8 = 1
	if c.Exclusive {
		sharedFlag = 0
	}

	if err = binary.Write(*c.c, binary.BigEndian, sharedFlag); err != nil {
		return err
	}

	var Width uint16
	var Height uint16
	if err = binary.Read(*c.c, binary.BigEndian, &Width); err != nil {
		return err
	}
	c.cli.c.Width = int(Width)
	if err = binary.Read(*c.c, binary.BigEndian, &Height); err != nil {
		return err
	}
	c.cli.c.Height = int(Height)
	if c.PixelFormat, err = readPixelFormat(*c.c); err != nil {
		return err
	}
	var nameLength uint32
	if err = binary.Read(*c.c, binary.BigEndian, &nameLength); err != nil {
		return err
	}

	nameBytes := make([]uint8, nameLength)
	if err = binary.Read(*c.c, binary.BigEndian, &nameBytes); err != nil {
		return err
	}

	c.DesktopName = string(nameBytes)

	return nil
}

func (c *Conn) clientServe() {
	//	defer c.Close()

	typeMap := make(map[uint8]Message)

	defaultMessages := []Message{
		new(FramebufferUpdateMsg),
		new(SetColorMapEntriesMsg),
		new(BellMsg),
		new(ServerCutTextMsg),
	}

	for _, msg := range defaultMessages {
		typeMap[msg.Type()] = msg
	}

	if c.cli.c.Messages != nil {
		for _, msg := range c.cli.c.Messages {
			typeMap[msg.Type()] = msg
		}
	}

	go func() {
		for {
			messageType, err := c.readByte()
			if err != nil {
				break
			}
			msg, ok := typeMap[messageType]
			if !ok {
				// Unsupported message type! Bad!
				break
			}
			parsedMsg, err := msg.Read(c, *c.c)
			if err != nil {
				fmt.Printf("cli.MessageSrv: %T %s\n", msg, err.Error())
				break
			} else {
				fmt.Printf("cli.MessageSrv: %T\n", msg)
			}
			c.MessageSrv <- &parsedMsg
		}
	}()
	for {
		select {
		case msg := <-c.MessageCli:
			m := *msg
			err := m.Write(c, *c.c)
			if err != nil {
				fmt.Printf("cli.MessageCli: %T %s\n", msg, err.Error())
				break
			} else {
				fmt.Printf("cli.MessageCli: %T\n", msg)
			}
		}
	}
}
