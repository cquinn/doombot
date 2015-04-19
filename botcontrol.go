package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"reflect"

	"azul3d.org/gfx.v1"
	"azul3d.org/gfx/window.v2"
	"azul3d.org/keyboard.v1"
	"azul3d.org/mouse.v1"
	"github.com/xa4a/go-roomba"
)

const (
	defaultPort    = "/dev/cu.usbserial-DA017N8D"
	velocityChange = 300
	rotationChange = 400
)

var (
	portName = flag.String("port", defaultPort, "Roomba's serial port name")
)

// gfxLoop is responsible for drawing things to the window.
func gfxLoop(w window.Window, r gfx.Renderer) {

	// Get ready to Roomba!
	flag.Parse()
	bot, err := roomba.MakeRoomba(*portName)
	if err != nil {
		log.Fatal("Making Roomba failed")
	}
	err = bot.Start()
	if err != nil {
		log.Fatal("Starting failed")
	}
	err = bot.Safe()
	if err != nil {
		log.Fatal("Entering Safe mode failed")
	}

	fmt.Printf("\nMain Bot: %#v", bot)

	// You can handle window events in a seperate goroutine!
	go func() {
		// Create our events channel with sufficient buffer size.
		events := make(chan window.Event, 256)

		// Notify our channel anytime any event occurs.
		w.Notify(events, window.AllEvents)

		fmt.Printf("\nBot: %#v", bot)

		// Wait for events.
		for event := range events {
			switch event.(type) {
			case keyboard.StateEvent:
				fmt.Printf("\nEvent type %s: %v\n", reflect.TypeOf(event), event)
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

					fmt.Printf("Updating Right:%d Left:%d\n", vr, vl)

					bot.DirectDrive(int16(vr), int16(vl))
				}
			}
		}
	}()

	upArr := image.Rect(100, 0, 100+100, 0+100)
	dnArr := image.Rect(100, 200, 100+100, 200+100)
	lfArr := image.Rect(0, 100, 0+100, 100+100)
	rtArr := image.Rect(200, 100, 200+100, 100+100)

	for {
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
			fmt.Printf("Quitting...\n")
			bot.Stop()         // Motor Stop
			bot.WriteByte(173) // Roomba Stop
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
	window.Run(gfxLoop, nil)
}
