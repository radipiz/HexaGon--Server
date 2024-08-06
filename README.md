# HexaGon
Just a TCP to serial pipe

## Configuration and run

```
Usage of ./hexagon:
  -baud int
        Selected Baud Rate for the serial interface (default 9600)
  -port int
        Port for the server (default 3092)
  -serial string
        Serial port to use to communicate with the Hexagon controller (default "nil")
```

If _serial_ is unset ("nil"), hexaGon will dump a list of available ports.

## Build

```
# Current platform
go build
# Linux
env GOOS=linux GOARCH=amd64 go build -o hexagon
# Windows
env GOOS=windows GOARCH=amd64 go build -o hexagon-windows-amd64.exe
# See all available platforms
go tool dist list
``` 

## Infodump

Create a pipe for testing using _socat_:
```
socat -d -d pty,raw,echo=0 pty,raw,echo=0
```

