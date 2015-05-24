package main

/*
Super quick and dirty code to control the GPIO 12 PWN signal from a RasberryPi
Relies on the raspi_pwn branch of github.com/hybridgroup/gobot/platforms/raspi
so it has to be compiled /installed locally (ie, dont 'go get' it)
Also relies on pi_blaster being installed and active on the PI
(https://github.com/sarfata/pi-blaster)
*/

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/hybridgroup/gobot/platforms/raspi"
)

const (
	defaultPort = 9004
	minTilt     = 17 // ~ int(255 * 0.06) min duty cycle for PWN signal
	maxTilt     = 52 // ~ int(255 * 0.21) max duty cycle for PWN signal
)

var (
	netPort = flag.Int("port", defaultPort, "Network port to listen on")
)

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
		handleConn(netConn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	raspiAdaptor := raspi.NewRaspiAdaptor("raspi")
	setTilt(raspiAdaptor, 50)
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		message = strings.TrimSpace(strings.ToUpper(message))
		log.Printf("Got '%s'", message)
		switch {
		case strings.HasPrefix(message, "QUIT"):
			return
		case err != nil:
			return
		case strings.HasPrefix(message, "TILT"):
			var newTilt int
			_, _ = fmt.Sscanf(message, "TILT %d", &newTilt)
			setTilt(raspiAdaptor, newTilt)
		}
	}
}

func setTilt(r *raspi.RaspiAdaptor, newTilt int) {
	log.Printf("Tilting to '%d'", newTilt)
	if newTilt > 100 {
		newTilt = 100
	}
	if newTilt < 0 {
		newTilt = 0
	}
	actualTilt := uint8(minTilt + (maxTilt-minTilt)*newTilt/100)
	log.Printf("PWM tilt to '%d'", actualTilt)
	// hardcoded RasPi GPIO pin 12 cause yolo and it's late and thunderdome and all that
	r.PwmWrite("12", actualTilt)
}
