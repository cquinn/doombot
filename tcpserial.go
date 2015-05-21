package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/tarm/serial"
)

const (
	defaultDev  = "/dev/cu.usbserial-DA017N8D"
	defaultPort = 9003
)

var (
	serialDev = flag.String("serial", defaultDev, "Local serial device")
	netPort   = flag.Int("port", defaultPort, "Network port to listen on")
)

// Mostly untested program to pipe bytes between a given serial port and a network port.
func main() {
	flag.Parse()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *netPort))
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		log.Printf("Listening on: %d", *netPort)
		netConn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		log.Printf("Connection on: %d", *netPort)

		log.Printf("Opening serial port: %s", *serialDev)
		c := &serial.Config{Name: *serialDev, Baud: 115200, ReadTimeout: time.Second * 5}
		serConn, err := serial.OpenPort(c)
		if err != nil {
			log.Fatal(err)
		}

		waitGroup := &sync.WaitGroup{}
		stopper := make(chan bool, 2)

		waitGroup.Add(1)
		go copyWithAbort(netConn, serConn, stopper, waitGroup)

		waitGroup.Add(1)
		go copyWithAbort(serConn, netConn, stopper, waitGroup)

		waitGroup.Wait()
	}
}

func copyWithAbort(dst io.WriteCloser, src io.ReadCloser, stopper chan bool, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	_, netSrc := src.(*net.TCPConn)

	log.Printf("Copying %s => %s", reflect.TypeOf(src), reflect.TypeOf(dst))
	for {
		// Check for a stop signal
		//var stop bool
		select {
		case <-stopper:
			// We've been told to stop: close our source.
			log.Printf("Stopping %s => %s; closing src", reflect.TypeOf(src), reflect.TypeOf(dst))
			src.Close()
			return
		default:
		}
		// copy bytes, timeout after deadline
		//src.SetReadDeadline(time.Now().Add(5 * time.Second))
		copied, err := io.Copy(dst, src)
		log.Printf("Copied %v => %v %d bytes, err:%v", reflect.TypeOf(src), reflect.TypeOf(dst), copied, err)
		// just a timeout, keep trying
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			log.Printf("  looping: timeout")
			continue
		}
		// nothing to read: assume src has been closed
		if netSrc && copied == 0 {
			log.Printf("  done: net src has no data")
			stopper <- true
			return
		}

		// nothing to read: assume src has been closed
		if err != nil {
			log.Printf("  done: err:%v", err)
			stopper <- true
			return
		}
	}

}
