package main

import (
	"flag"
	//"fmt"
	"image"
	"log"
	"net"
	"reflect"

	"azul3d.org/gfx.v1"
	"azul3d.org/gfx/window.v2"
	"azul3d.org/keyboard.v1"
	"azul3d.org/mouse.v1"
	"github.com/xa4a/go-roomba"
)

const (
	defaultSerial  = "/dev/cu.usbserial-DA017N8D"
	velocityChange = 300
	rotationChange = 400
)

var (
	serialPort = flag.String("serial", defaultSerial, "Local serial port name.")
	remoteAddr = flag.String("remote", "", "Remote Roomba's network address and port.")
)

func makeRemoteRoomba(remoteAddr string) (*roomba.Roomba, error) {
	// from MakeRoomba()...
	roomba := &roomba.Roomba{PortName: remoteAddr, StreamPaused: make(chan bool, 1)}
	conn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		return nil, err
	}
	roomba.S = conn
	return roomba, nil
}

// gfxLoop is responsible for drawing things to the window.
func gfxLoop(w window.Window, r gfx.Renderer) {

	log.Printf("gfxLoop")
	flag.Parse()

	// Who we gonna call? Default to local serial unless a remote addr was given
	var bot *roomba.Roomba
	if *remoteAddr != "" {
		log.Printf("Connecting to remote Doombot @ %s", *remoteAddr)
		var err error
		bot, err = makeRemoteRoomba(*remoteAddr)
		if err != nil {
			log.Fatalf("Connecting to remote Doombot @ %s failed", *remoteAddr)
		}
	} else {
		log.Printf("Connecting to local serial Doombot @ %s", *serialPort)
		var err error
		bot, err = roomba.MakeRoomba(*serialPort)
		if err != nil {
			log.Fatalf("Connecting to local serial Doombot @ %s failed", *serialPort)
		}
	}

	// Start the Doombot & put it into Safe mode
	log.Println()
	log.Printf("Starting Doombot %s", bot.PortName)
	err := bot.Start()
	if err != nil {
		log.Fatal("Starting failed")
	}
	log.Printf("Entering Safe mode")
	err = bot.Safe()
	if err != nil {
		log.Fatal("Entering Safe mode failed")
	}
	log.Println()

	// Handle window events in a seperate goroutine
	go func() {
		// Create our events channel with sufficient buffer size.
		events := make(chan window.Event, 256)

		// Notify our channel anytime any event occurs.
		w.Notify(events, window.AllEvents)

		log.Printf("Event handler gorouting running")

		// Wait for events.
		for event := range events {
			switch event.(type) {
			case keyboard.StateEvent:
				log.Println()
				log.Printf("Event type %s: %v", reflect.TypeOf(event), event)
				ke := event.(keyboard.StateEvent)

				motionChange := ke.Key == keyboard.ArrowUp || ke.Key == keyboard.ArrowDown ||
					ke.Key == keyboard.ArrowLeft || ke.Key == keyboard.ArrowRight

				if motionChange {
					velocity := 0
					if w.Keyboard().Down(keyboard.ArrowUp) {
						velocity = velocityChange
					}
					if w.Keyboard().Down(keyboard.ArrowDown) {
						velocity = -velocityChange
					}
					rotation := 0
					if w.Keyboard().Down(keyboard.ArrowLeft) {
						rotation = rotationChange
					}
					if w.Keyboard().Down(keyboard.ArrowRight) {
						rotation = -rotationChange
					}

					// compute left and right wheel velocities
					vr := velocity + (rotation / 2)
					vl := velocity - (rotation / 2)

					log.Printf("Updating Right:%d Left:%d", vr, vl)
					bot.DirectDrive(int16(vr), int16(vl))

				} else {
					if w.Keyboard().Down(keyboard.R) {
						log.Printf("Resetting to safe mode")
						err = bot.Safe()
						if err != nil {
							log.Fatal("Entering Safe mode failed")
						}
					}
				}
			}
		}
	}()

	// cheesy block direction signals
	upArr := image.Rect(100, 0, 100+100, 0+100)
	dnArr := image.Rect(100, 200, 100+100, 200+100)
	lfArr := image.Rect(0, 100, 0+100, 100+100)
	rtArr := image.Rect(200, 100, 200+100, 100+100)

	for {
		//log.Printf("Rendering")
		// Clear the entire area (empty rectangle means "the whole area").
		r.Clear(image.Rect(0, 0, 0, 0), gfx.Color{1, 1, 1, 1})

		// The keyboard is monitored for you, simply check if a key is down:
		if w.Keyboard().Down(keyboard.ArrowUp) {
			// Clear a red rectangle.
			r.Clear(upArr, gfx.Color{1, 0, 0, 1})
		}
		if w.Keyboard().Down(keyboard.ArrowDown) {
			// Clear a red rectangle.
			r.Clear(dnArr, gfx.Color{1, 0, 0, 1})
		}
		if w.Keyboard().Down(keyboard.ArrowLeft) {
			// Clear a red rectangle.
			r.Clear(lfArr, gfx.Color{1, 0, 0, 1})
		}
		if w.Keyboard().Down(keyboard.ArrowRight) {
			// Clear a red rectangle.
			r.Clear(rtArr, gfx.Color{1, 0, 0, 1})
		}

		if w.Keyboard().Down(keyboard.Q) {
			log.Printf("Quitting")
			bot.Stop() // Motor Stop
			//bot.WriteByte(173) // Roomba Stop
			w.Close()
		}

		// And the same thing with the mouse, check if a mouse button is down:
		if w.Mouse().Down(mouse.Left) {
			// Clear a blue rectangle.
			r.Clear(image.Rect(100, 100, 200, 200), gfx.Color{0, 0, 1, 1})
		}

		// Render the whole frame.
		r.Render()
	}
}

func main() {
	log.Printf("Main")
	window.Run(gfxLoop, nil)
}
