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

## Infodump

Create a pipe for testing using _socat_:
```
socat -d -d pty,raw,echo=0 pty,raw,echo=0
```

