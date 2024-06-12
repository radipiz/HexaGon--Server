package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"go.bug.st/serial.v1"
	"go.bug.st/serial/enumerator"
)

var (
	selectedPort string
	baudRate     int
	port         int
)

var connectedClients uint16 = 0

func main() {
	flag.StringVar(&selectedPort, "serial", "nil", "Serial port to use to communicate with the Hexagon controller")
	flag.IntVar(&baudRate, "baud", 9600, "Selected Baud Rate for the serial interface")
	flag.IntVar(&port, "port", 3092, "Port for the server")
	flag.Parse()

	if selectedPort == "nil" {
		displayAvailablePorts()
	} else {
		startServer()
	}

}
func startServer() {
	mode := &serial.Mode{
		BaudRate: baudRate,
	}

	serialPort, err := serial.Open(selectedPort, mode)
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}
	defer serialPort.Close()

	log.Printf("Opened serial port %s\n", selectedPort)

	// Start TCP server
	listener, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	serialBuffer := make(chan []byte, 4)
	closeSerial := make(chan bool, 1)
	defer func() { closeSerial <- true }()
	go serialFlush(serialPort, serialBuffer, closeSerial)

	// Await connections
	log.Println("Waiting for TCP connection...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Failed to accept TCP connection: %v", err)
		}
		connectedClients += 1
		log.Printf("TCP connection established. %d Connected clients right now\n", connectedClients)
		go handleConnection(conn, serialBuffer)
	}
}

func displayAvailablePorts() {
	// Find available serial ports
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		log.Fatalf("Failed to list serial ports: %v", err)
	}

	if len(ports) == 0 {
		log.Fatal("No serial ports found")
	}

	for _, port := range ports {
		fmt.Printf("Found port: %s\n", port.Name)
		if port.IsUSB {
			fmt.Printf(" - USB ID     %s:%s\n", port.VID, port.PID)
			fmt.Printf(" - USB serial %s\n", port.SerialNumber)
		}
	}
}

func handleConnection(con net.Conn, serialBuffer chan []byte) {
	defer con.Close()
	pongbytes := []byte("pong")

	readTimeout := 30 * time.Second
	// start the ping sender to make sure the read timeout never happens
	go ping_sender(con, readTimeout)

	for {
		con.SetReadDeadline(time.Now().Add(readTimeout))
		buffer := make([]byte, 128)
		_, err := con.Read(buffer)

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Printf("[%s] Client read timeout, disconnecting\n", con.RemoteAddr().String())
			connectedClients -= 1
			return
		} else if err != nil {
			fmt.Printf("[%s] Read error: %s", con.RemoteAddr().String(), err)
			connectedClients -= 1
			return
		}
		if bytes.HasPrefix(buffer, pongbytes) {
			fmt.Printf("[%s] Received Pong\n", con.RemoteAddr().String())
		} else {
			fmt.Println("Received Data", string(buffer))
			serialBuffer <- buffer
		}
	}
}

func ping_sender(con net.Conn, timeout time.Duration) {
	for {
		time.Sleep(timeout - 2*time.Second)
		con.Write([]byte("ping"))
	}
}

func serialFlush(serial serial.Port, sendBuffer chan []byte, done chan bool) {
	for {
		select {
		case msg := <-sendBuffer:
			_, err := serial.Write(msg)
			if err != nil {
				log.Printf("Error forwarding data from serial to TCP: %v", err)
			}
		case <-done:
			return
		}
	}
}
