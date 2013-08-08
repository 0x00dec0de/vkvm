/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Example of using the rfb package.
//
// Author: Brad Fitzpatrick <brad@danga.com>

package main

import (
	"./rfb/"
	"flag"
	"log"
	"net"
)

var (
	listen = flag.String("listen", ":6900", "listen on [ip]:port")
)

const (
	width  = 640
	height = 480
)

func main() {
	flag.Parse()

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}

	s := rfb.NewServer(width, height)
	go func() {
		err = s.Serve(ln)
		log.Fatalf("rfb server ended with: %v", err)
	}()
	for c := range s.Conns {
		handleConn(c)
	}
}

func handleConn(c *rfb.Conn) {
	err := c.Proxy("127.0.0.1:5900")
	if err != nil {
		log.Fatalf("rfb server error: %v", err)
	}
}
