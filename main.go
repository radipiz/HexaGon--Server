package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

type CommandToSerial struct {
	message      []byte
	responseChan chan []byte
}

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

	serialPort.SetReadTimeout(300 * time.Millisecond)
	defer serialPort.Close()

	log.Printf("Opened serial port %s\n", selectedPort)

	// Start TCP server
	listener, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	serialSend := make(chan CommandToSerial, 4)
	closeSerial := make(chan bool, 1)
	defer func() { closeSerial <- true }()
	go serialFlush(serialPort, serialSend, closeSerial)

	// Await connections
	log.Println("Waiting for TCP connection...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Failed to accept TCP connection: %v", err)
		}
		connectedClients += 1
		log.Printf("TCP connection established. %d Connected clients right now\n", connectedClients)
		go handleConnection(conn, serialSend)
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

func handleConnection(con net.Conn, serialSend chan CommandToSerial) {
	defer con.Close()
	pongbytes := []byte("pong")

	readTimeout := 30 * time.Second
	// start the ping sender to make sure the read timeout never happens
	go ping_sender(con, readTimeout)
	responseChan := make(chan []byte)

	for {
		con.SetReadDeadline(time.Now().Add(readTimeout))
		buffer := make([]byte, 128)
		_, err := con.Read(buffer)

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Printf("[%s] Client read timeout, disconnecting\n", con.RemoteAddr().String())
			connectedClients -= 1
			return
		} else if err != nil {
			fmt.Printf("[%s] Read error: %s\n", con.RemoteAddr().String(), err)
			connectedClients -= 1
			return
		}
		if bytes.HasPrefix(buffer, pongbytes) {
			fmt.Printf("[%s] Received Pong\n", con.RemoteAddr().String())
		} else {
			fmt.Println("Received Data", string(buffer))
			serialSend <- CommandToSerial{buffer, responseChan}
			buffer = <-responseChan
			fmt.Println("Respond Data:", string(buffer))
			con.Write(buffer)
		}
	}
}

func ping_sender(con net.Conn, timeout time.Duration) {
	for {
		time.Sleep(timeout - 2*time.Second)
		con.Write([]byte("ping"))
	}
}

func serialFlush(serial serial.Port, sendBuffer chan CommandToSerial, done chan bool) {
	for {
		select {
		case command := <-sendBuffer:
			_, err := serial.Write(command.message)
			if err != nil {
				log.Printf("Error forwarding data from serial to TCP: %v\n", err)
			}
			responseBuffer := make([]byte, 128)
			serial.Read(responseBuffer)
			command.responseChan <- responseBuffer
		case <-done:
			return
		}
	}
}
