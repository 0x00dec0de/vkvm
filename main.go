package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"crypto/des"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	//	"log/syslog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	lbase64  = flag.String("lb", ":443", "listen for base64 websocket conns")
	lbinary  = flag.String("ln", ":17523", "listen for binary websocket conns")
	lvnc     = flag.String("lvnc", ":5900", "listen for vnc conns")
	tlscrt   = flag.String("tlscrt", "", "TLS path for certificate")
	tlskey   = flag.String("tlskey", "", "TLS path for key")
	authurl  = flag.String("authurl", "", "http address for external auth")
	authdata = flag.String("authdata", "", "http form values used in http auth")

//	l, _     = syslog.NewLogger(syslog.LOG_DEBUG|syslog.LOG_DAEMON, log.LstdFlags)
)

type Conn struct {
	c         net.Conn
	srvMsg    [][]byte
	retries   int
	challenge []byte
	password  []byte
	Dst       string
	Src       string
}

type Proxy struct {
	sync.Mutex
	conns map[*Conn]*Conn
}

var p Proxy

func main() {
	flag.Parse()

	http.Handle("/websockify", websocket.Server{Handler: wsHandler,
		Handshake: func(ws *websocket.Config, req *http.Request) error {
			ws.Protocol = []string{"base64"}
			//			ws.Protocol = []string{"binary"}
			return nil
		}})

	p.conns = make(map[*Conn]*Conn, 4096)
	log.Fatal(http.ListenAndServeTLS(*lbase64, *tlscrt, *tlskey, nil))
}

func getConn(srv *Conn) (cli *Conn, err error) {
	p.Lock()
	defer p.Unlock()
	if cli, ok := p.conns[srv]; !ok {
		return nil, fmt.Errorf("failed to get conn")
	} else {
		return cli, nil
	}

	return nil, fmt.Errorf("Something strange")
}

func reconnect(srv *Conn) (cli *Conn, err error) {
	var res *http.Response
	srv.retries += 1

	buf := new(bytes.Buffer)
	enc := base64.NewEncoder(base64.StdEncoding, buf)
	enc.Write(srv.challenge)
	enc.Close()

	v := url.Values{}
	v.Set("hash", buf.String())
	buf.Reset()

	v.Set("ip", srv.Src)
	if *authdata != "" {
		parts := strings.Split(*authdata, "&")
		for _, pv := range parts {
			p := strings.Split(pv, "=")
			if len(p) == 2 {
				v.Set(p[0], p[1])
			}
		}
	}
	if res, err = http.PostForm(*authurl, v); err != nil || res == nil {
		return nil, fmt.Errorf("failed to get auth data")
	}

	if res.StatusCode != 200 || res.Body == nil {
		if res.Body != nil {
			io.Copy(buf, res.Body)
		}
		defer buf.Reset()
		return nil, fmt.Errorf("failed to get auth data: code %d body %s", res.StatusCode, buf.String())
	}
	_, err = io.Copy(buf, res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth data: %s", err.Error())
	}
	log.Printf("http auth: %s\n", buf.Bytes())
	res.Body.Close()

	data := strings.Split(buf.String(), " ")
	if len(data) < 2 {
		return nil, fmt.Errorf("failed to get auth data data invalid")
	}
	buf.Reset()

	srv.Dst = data[0]
	srv.password = []byte(data[1])

	c, err := net.Dial("tcp", srv.Dst)
	if err != nil {
		//if c != nil {
		//	c.Close()
		//}
		return nil, err
	}
	cli = &Conn{c: c, Dst: srv.Dst, password: srv.password}

	var protocolVersion [12]byte

	if err := binary.Read(cli.c, binary.BigEndian, &protocolVersion); err != nil {
		return nil, err
	}

	var maxMajor, maxMinor uint8
	_, err = fmt.Sscanf(string(protocolVersion[:]), "RFB %d.%d\n", &maxMajor, &maxMinor)
	if err != nil {
		return nil, err
	}

	if maxMajor < 3 {
		return nil, fmt.Errorf("unsupported major version, less than 3: %d", maxMajor)
	}

	if maxMinor < 8 {
		return nil, fmt.Errorf("unsupported minor version, less than 8: %d", maxMinor)
	}

	if err = binary.Write(cli.c, binary.BigEndian, []byte("RFB 003.008\n")); err != nil {
		return nil, err
	}

	var numSecurityTypes uint8
	if err = binary.Read(cli.c, binary.BigEndian, &numSecurityTypes); err != nil {
		return nil, err
	}

	if numSecurityTypes == 0 {
		var reasonLength uint32
		if err = binary.Read(cli.c, binary.BigEndian, &reasonLength); err != nil {
			return nil, err
		}
		reasonText := make([]byte, reasonLength)
		if err = binary.Read(cli.c, binary.BigEndian, &reasonText); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no security types: %s", reasonText)
	}

	securityTypes := make([]uint8, numSecurityTypes)
	if err = binary.Read(cli.c, binary.BigEndian, &securityTypes); err != nil {
		return nil, err
	}

	auth := false
	for _, t := range securityTypes {
		if t == uint8(2) {
			auth = true
			break
		}
	}

	if !auth {
		return nil, fmt.Errorf("no suitable auth schemes found.")
	}

	// Respond back with the security type we'll use
	if err = binary.Write(cli.c, binary.BigEndian, uint8(2)); err != nil {
		return nil, err
	}

	if err = cliAuth(cli); err != nil {
		return nil, err
	}

	// 7.1.3 SecurityResult Handshake
	var securityResult uint32
	if err = binary.Read(cli.c, binary.BigEndian, &securityResult); err != nil {
		return nil, err
	}

	if securityResult == 1 {
		var reasonLength uint32
		if err = binary.Read(cli.c, binary.BigEndian, &reasonLength); err != nil {
			return nil, err
		}
		reasonText := make([]byte, reasonLength)
		if err = binary.Read(cli.c, binary.BigEndian, &reasonText); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("security handshake failed: %s", reasonText)
	}

	if srv.retries != 1 {
		if err = binary.Write(cli.c, binary.BigEndian, uint8(1)); err != nil {
			return nil, err
		}
	}
	return cli, nil
}

func cliAuth(cli *Conn) (err error) {
	challenge := make([]uint8, 16)

	if err := binary.Read(cli.c, binary.BigEndian, &challenge); err != nil {
		return err
	}
	pwd := cli.password
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
	if err = binary.Write(cli.c, binary.BigEndian, response); err != nil {
		return err
	}
	return nil
}

func srvAuth(srv *Conn) (err error) {
	w := base64.NewEncoder(base64.StdEncoding, srv.c)
	r := base64.NewDecoder(base64.StdEncoding, srv.c)

	challenge := make([]uint8, 16)
	response := make([]uint8, 16)

	challenge = []byte("clodo.ruclodo.ru")

	if err := binary.Write(w, binary.BigEndian, challenge); err != nil {
		return err
	}
	w.Close()
	if err := binary.Read(r, binary.BigEndian, &response); err != nil {
		return err
	}
	srv.challenge = response
	return nil
}

func cliHandshake(srv *Conn) (err error) {
	w := base64.NewEncoder(base64.StdEncoding, srv.c)
	r := base64.NewDecoder(base64.StdEncoding, srv.c)

	var protocolVersion [12]byte
	// Respond with the version we will support
	if err := binary.Write(w, binary.BigEndian, []byte("RFB 003.008\n")); err != nil {
		return err
	}
	w.Close()
	if _, err := io.ReadFull(r, protocolVersion[:]); err != nil {
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

	if maxMinor < 3 {
		return fmt.Errorf("unsupported minor version, less than 3: %d", maxMinor)
	}

	var securityType uint8
	if maxMinor >= 7 {
		if err := binary.Write(w, binary.BigEndian, []uint8{uint8(1), uint8(2)}); err != nil {
			return err
		}
		w.Close()
		if err := binary.Read(r, binary.BigEndian, &securityType); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.BigEndian, uint32(2)); err != nil {
			return err
		}
		w.Close()
		securityType = 2
	}
	if err := srvAuth(srv); err != nil {
		e := err
		if err = binary.Write(w, binary.BigEndian, uint32(1)); err != nil {
			return err
		}
		w.Close()
		if maxMinor >= 8 {
			reasonLen := uint32(len(e.Error()))
			reason := []byte(e.Error())
			buf := new(bytes.Buffer)
			defer buf.Reset()
			if err = binary.Write(buf, binary.BigEndian, reasonLen); err != nil {
				return err
			}
			if err = binary.Write(buf, binary.BigEndian, &reason); err != nil {
				return err
			}
			w.Write(buf.Bytes())
			w.Close()
		}
		return e
	}
	if err := binary.Write(w, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	w.Close()
	return nil
}

func wsHandler(ws *websocket.Conn) {
	cliClose := make(chan bool, 0)
	srvClose := make(chan bool, 0)
	wsClosed := false
	var err error
	var cli *Conn
	var srv *Conn

	//TODO: retunr right rfb auth error
	defer ws.Close()

	srv = &Conn{c: ws}
	defer srv.c.Close()
	defer delete(p.conns, srv)
	defer log.Printf("exited\n")

	if err := cliHandshake(srv); err != nil {
		log.Printf(err.Error())
		return
	}
	srv.Src, _, _ = net.SplitHostPort(ws.Request().RemoteAddr)
	cli, err = reconnect(srv)
	if cli != nil {
		defer cli.c.Close()
	}
	if err != nil {
		log.Printf(err.Error())
		return
	}

	go func() {
		sbuf := make([]byte, 32*1024)
		dbuf := make([]byte, 32*1024)
		for {
			n, e := srv.c.Read(sbuf)
			if e != nil {
				break
			}
			n, e = base64.StdEncoding.Decode(dbuf, sbuf[0:n])
			if e != nil {
				break
			}
			n, e = cli.c.Write(dbuf[0:n])
			if e != nil {
				break
			}
		}
		wsClosed = true
		srvClose <- true
		return
	}()

	go func() {
		sbuf := make([]byte, 32*1024)
		dbuf := make([]byte, 64*1024)
		for {

			n, e := cli.c.Read(sbuf)
			if e != nil && !wsClosed {
				cli, err = reconnect(srv)
				if err != nil {
					break
				}
			}
			if wsClosed {
				break
			}
			base64.StdEncoding.Encode(dbuf, sbuf[0:n])
			n = base64.StdEncoding.EncodedLen(len(sbuf[0:n]))
			_, err := srv.c.Write(dbuf[0:n])
			if err != nil {
				break
			}
		}
		cliClose <- true
	}()

	for {

		select {
		case <-srvClose:
			return
		case <-cliClose:
			return

		}
	}
}
