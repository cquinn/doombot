package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	//"log"
	"net"
	"sync"
	"time"

	"github.com/golang/glog"
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
	glog.Infof("Opening serial port: %s\n", *serialDev)
	c := &serial.Config{Name: *serialDev, Baud: 115200, ReadTimeout: time.Second * 2}
	serConn, err := serial.OpenPort(c)
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *netPort))
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	defer ln.Close()

	for {
		glog.Infof("Listening on: %v\n", *netPort)
		netConn, err := ln.Accept()
		if err != nil {
			glog.Error(err)
			continue
		}

		glog.Infof("Connection on: %v\n", *netPort)

		waitGroup := &sync.WaitGroup{}
		waitGroup.Add(1)
		stopper := make(chan bool, 2)

		waitGroup.Add(1)
		go copyWithAbort(netConn, serConn, stopper, waitGroup)

		waitGroup.Add(1)
		go copyWithAbort(serConn, netConn, stopper, waitGroup)

		waitGroup.Wait()
	}
}

func copyWithAbort(dst io.WriteCloser, src io.ReadCloser, stopper chan bool, waitGroup *sync.WaitGroup) {
	//src.SetReadDeadline(time.Now().Add(1000 * time.Millisecond)) // 1000 ms max wait
	glog.Infof("Copying: %v <= %v\n", dst, src)
	defer waitGroup.Done()
	for {
		// Check for a stop signal
		//var stop bool
		select {
		case <-stopper:
			// We've been told to stop: close our source.
			glog.Infof("Stopping: %v <= %v; closing %v\n", dst, src, src)
			src.Close()
			return
		default:
		}
		// copy bytes, timeout after deadline
		copied, err := io.Copy(dst, src)
		glog.Info("  copied %d %v <= %v exited %s, looping\n", copied, dst, src, err)
		// just a timeout, keep trying
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			continue
		}
		// nothing to read: assume src has been closed
		if copied == 0 {
			return
		}
	}

}
