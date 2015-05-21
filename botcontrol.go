package main

import (
	"flag"
	//"fmt"
	"encoding/binary"
	"image"
	"log"
	"net"
	"reflect"
	"time"

	"azul3d.org/gfx.v1"
	"azul3d.org/gfx/window.v2"
	"azul3d.org/keyboard.v1"
	"azul3d.org/mouse.v1"
	"github.com/cquinn/doombot/testing"
	"github.com/cquinn/doombot/text"
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
	testMode   = flag.String("testMode", "", "Set to true to use a mock roomba")
	modes      = []string{"Off", "Passive", "Safe", "Full"}
)

type TimeEvent struct{}

func (e *TimeEvent) Time() time.Time {
	return time.Now()
}

type SensorInfo struct {
	voltage  uint
	current  int
	temp     int
	charge   uint
	capacity uint
	// TODO - add enum for mode (careful, can get whacky modes...)
	mode uint
}

func getSensorInfo(bot *roomba.Roomba) (*SensorInfo, error) {
	si := &SensorInfo{}
	log.Println()
	log.Printf("GETTING SENSOR INFO")
	// TODO - check for error return from sensor stuff
	si.voltage, _ = sensorU16(bot, 22)
	log.Printf("Voltage: %dmV", si.voltage)

	si.current, _ = sensorS16(bot, 23)
	log.Printf("Current: %dmA", si.current)

	si.temp, _ = sensorS8(bot, 24)
	log.Printf("Temp: %dC", si.temp)

	si.charge, _ = sensorU16(bot, 25)
	log.Printf("Charge: %dmAh", si.charge)

	si.capacity, _ = sensorU16(bot, 26)
	log.Printf("Capacity: %dmAh", si.capacity)

	si.mode, _ = sensorU8(bot, 35)
	log.Printf("Mode: %d", si.mode)

	log.Println()

	return si, nil
}

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

func sensorS8(bot *roomba.Roomba, sensor byte) (int, error) {
	bytes, err := bot.Sensors(sensor)
	if err != nil {
		return 0, err
	}
	//log.Printf(" s8 bytes %v", bytes)
	val := bytes[0]
	return int(val), nil
}

func sensorS16(bot *roomba.Roomba, sensor byte) (int, error) {
	bytes, err := bot.Sensors(sensor)
	if err != nil {
		return 0, err
	}
	//log.Printf(" s16 bytes %v", bytes)
	val := binary.BigEndian.Uint16(bytes)
	return int(val), nil
}

func sensorU8(bot *roomba.Roomba, sensor byte) (uint, error) {
	bytes, err := bot.Sensors(sensor)
	if err != nil {
		return 0, err
	}
	//log.Printf(" u8 bytes %v", bytes)
	val := bytes[0]
	return uint(val), nil
}

func sensorU16(bot *roomba.Roomba, sensor byte) (uint, error) {
	bytes, err := bot.Sensors(sensor)
	if err != nil {
		return 0, err
	}
	//log.Printf(" u16 bytes %v", bytes)
	val := binary.BigEndian.Uint16(bytes)
	return uint(val), nil
}

// gfxLoop is responsible for drawing things to the window.
func gfxLoop(w window.Window, r gfx.Renderer) {

	log.Printf("gfxLoop")
	flag.Parse()

	// Who we gonna call? Default to local serial unless a remote addr was given
	var bot *roomba.Roomba
	if *testMode == "true" {
		log.Printf("Creating mock doombot")
		bot = testing.MakeTestRoomba()
	} else if *remoteAddr != "" {
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

	mode, _ := sensorU8(bot, 35)
	log.Printf("Mode: %d", mode)
	//log.Printf("Mode: %s", modes[mode])

	// TODO - add sensor 21 to mock roomba
	charging, _ := sensorU8(bot, 21)
	log.Printf("Charging state: %d", charging)
	log.Println()

	log.Printf("Entering Safe mode")
	err = bot.Safe()
	if err != nil {
		log.Fatal("Entering Safe mode failed")
	}
	log.Println()

	charging, _ = sensorU8(bot, 21)
	log.Printf("Charging state: %d", charging)
	/*
		0 Not charging
		1 Reconditioning Charging 2 Full Charging
		3 Trickle Charging
		4 Waiting
		5 Charging Fault Condition
	*/

	// to be populated by a timer and dumped to the UI
	var sensor *SensorInfo

	// Create our events channel with sufficient buffer size.
	events := make(chan window.Event, 256)

	// collect sensor data in a separate goroutine
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			events <- &TimeEvent{}
		}
	}()

	// Handle window events in a seperate goroutine
	go func() {
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
						err = bot.Start()
						err = bot.Safe()
						if err != nil {
							log.Fatal("Entering Safe mode failed")
						}
					} else if w.Keyboard().Down(keyboard.D) {
						log.Printf("Seeking Dock")
						err = bot.WriteByte(143) // Seek Dock
					}
				}
			case *TimeEvent:
				// grab all the sensor info
				log.Printf("Getting latest sensor info")
				sensor, _ = getSensorInfo(bot)
			}
		}
	}()

	// cheesy block direction signals
	upArr := image.Rect(150, 100, 150+50, 100+50)
	dnArr := image.Rect(150, 200, 150+50, 200+50)
	lfArr := image.Rect(100, 150, 100+50, 150+50)
	rtArr := image.Rect(200, 150, 200+50, 150+50)

	//bumpers
	tlBumper := image.Rect(30, 50, 30+80, 50+30)
	trBumper := image.Rect(240, 50, 240+80, 50+30)

	//batteryComponents
	botBat := image.Rect(600, 400, 600+50, 400+30)
	midBat := image.Rect(600, 370, 600+50, 370+30)
	topBat := image.Rect(600, 340, 600+50, 340+30)
	tipBat := image.Rect(610, 330, 610+30, 330+10)
	for {
		//log.Printf("Rendering")
		// Clear the entire area (empty rectangle means "the whole area").
		r.Clear(image.Rect(0, 0, 0, 0), gfx.Color{1, 1, 1, 1})

		//clear d-pad
		r.Clear(upArr, gfx.Color{0.5, 0, 0, 0.1})
		r.Clear(dnArr, gfx.Color{0.5, 0, 0, 0.1})
		r.Clear(lfArr, gfx.Color{0.5, 0, 0, 0.1})
		r.Clear(rtArr, gfx.Color{0.5, 0, 0, 0.1})

		r.Clear(tlBumper, gfx.Color{0.5, 0, 0, 0.1})
		r.Clear(trBumper, gfx.Color{0.5, 0, 0, 0.1})

		//draw battery
		r.Clear(botBat, gfx.Color{0.5, 0, 0, 0.1})
		r.Clear(midBat, gfx.Color{0.5, 0, 0, 0.1})
		r.Clear(topBat, gfx.Color{0.5, 0, 0, 0.1})
		r.Clear(tipBat, gfx.Color{0.5, 0, 0, 0.1})

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
			log.Println()
			log.Printf("Quitting")
			bot.Stop()         // Motor Stop
			bot.WriteByte(173) // Create 2 Stop
			//bot.Power()

			w.Close()
		}

		// And the same thing with the mouse, check if a mouse button is down:
		if w.Mouse().Down(mouse.Left) {
			// Clear a blue rectangle.
			r.Clear(image.Rect(100, 100, 200, 200), gfx.Color{0, 0, 1, 1})
		}

		// Add some text
		text.Runes = append(text.Runes, ev.Rune)
		text.Update()
		b := r.Bounds()
		text.Object.SetPos(lmath.Vec3{0, 0, float64(b.Dy()) / 2.0})
		r.Clear(image.Rect(0, 0, 0, 0), gfx.Color{0.2, 0.2, 0.2, 1})
		r.ClearDepth(image.Rect(0, 0, 0, 0), 1.0)
		r.Draw(image.Rect(0, 0, 0, 0), text.Object, camera)

		// Render the whole frame.
		r.Render()
	}
}

func main() {
	log.Printf("Main")
	window.Run(gfxLoop, nil)
}
