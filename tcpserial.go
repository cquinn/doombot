package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/tarm/serial"
)

const (
	defaultSerial = "/dev/cu.usbserial-DA017N8D"
	defaultPort   = 9090
)

var (
	serialPort = flag.String("serial", defaultSerial, "Local serial port name")
	netPort    = flag.Int("port", defaultPort, "Network port to listen on")
)

// Mostly untested program to pipe bytes between a given serial port and a network port.
func main() {
	flag.Parse()
	c := &serial.Config{Name: *serialPort, Baud: 115200}
	serConn, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *netPort))
	if err != nil {
		log.Fatal(err)
	}

	for {
		netConn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			for {
				io.Copy(netConn, serConn)
			}
		}()
		go func() {
			for {
				io.Copy(serConn, netConn)
			}
		}()
	}
}
