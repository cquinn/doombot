# doombot
Hacky code for playing with Roombas and Raspberry Pis for thunderdome

Using the github.com/xa4a/go-roomba library for talking to a Roomba, although also check out github.com/saljam/roomba and github.com/zagaberoo/roboderp.

botcontrol.go allows roomba navigation control via keyboard events. It can talk to the serial port, or a remote tcp socket.

tcpserial.go is a daemon that pipes bytes between a serial and a tcp port.
