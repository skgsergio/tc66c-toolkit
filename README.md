# TC66C Toolkit

A command-line toolkit for interacting with TC66/TC66C USB power meters. Read measurements, poll data continuously, retrieve recordings, and update firmware.

## Features

- **Get readings**: Single snapshot of voltage, current, power, and more
- **Continuous polling**: Monitor readings in real-time at configurable intervals
- **Web UI**: Browser-based interface with real-time graphing and monitoring
- **Recording retrieval**: Download stored measurement data from the device
- **Firmware updates**: Flash new firmware to your device (bootloader mode)
- **JSON output**: Export data in JSON format for scripting and analysis
- **Cross-platform**: Works on Linux, macOS, and Windows

## Installation

### From Source

Requires Go 1.25 or later:

```bash
git clone https://github.com/yourusername/tc66c-toolkit
cd tc66c-toolkit
go build -o tc66c-toolkit ./cmd/toolkit
```

## Usage

### Basic Commands

#### Get a Single Reading

```bash
# Basic reading
tc66c-toolkit get

# JSON output
tc66c-toolkit get --json

# Custom serial port
tc66c-toolkit get -p /dev/ttyUSB0
```

#### Continuous Polling

```bash
# Poll every 500ms (default)
tc66c-toolkit poll

# Poll every second
tc66c-toolkit poll -i 1s

# Poll with JSON output
tc66c-toolkit poll --json
```

#### Web UI

Start a web server with a browser-based interface for real-time monitoring:

```bash
# Start web server (default: localhost:8080)
tc66c-toolkit web

# Custom address and port
tc66c-toolkit web -a 0.0.0.0 -w 8080
```

Then open your browser to `http://localhost:8080`. The Web UI provides:

- **Serial port detection**: Automatically lists available serial ports
- **Real-time graphing**: Dual Y-axis charts with configurable metrics
- **Live readings**: Display of voltage, current, power, temperature, and more
- **Configurable polling**: Adjustable intervals from 100ms to 2s
- **WebSocket updates**: Efficient real-time data streaming

#### Retrieve Recordings

```bash
tc66c-toolkit recording
```

#### Update Firmware

**Warning**: Only use firmware files from trusted sources.

```bash
# Enter bootloader mode first:
# 1. Unplug the device
# 2. Press and hold K1 button
# 3. Plug in the device while holding K1
# 4. Release K1

tc66c-toolkit update -f firmware.bin
```

### Global Flags

- `-p, --port`: Serial port device path (default: `/dev/ttyACM0`)
- `-h, --help`: Show help

### Command-Specific Flags

**poll**:
- `-i, --interval`: Polling interval (default: `500ms`)
- `-j, --json`: Output in JSON format

**get**:
- `-j, --json`: Output in JSON format

**web**:
- `-a, --address`: Address to bind the web server (default: `localhost`)
- `-w, --web-port`: Port for the web server (default: `8080`)

**update**:
- `-f, --file`: Firmware file (required)

## Output Formats

### Text Format (Default)

```
Product: TC66
Version: 1.14
Serial: 12345678
Runs: 42
Voltage: 5.1234 V
Current: 0.51234 A
Power: 2.6234 W
Resistance: 10.00 Ω
Group 0: 1234 mAh / 5678 mWh
Group 1: 2345 mAh / 6789 mWh
Temperature: 25.0 °C
D+ Voltage: 2.75 V
D- Voltage: 2.75 V
```

### JSON Format

```json
{"product":"TC66","version":"1.14","serial_number":12345678,"num_runs":42,"voltage":5.1234,"current":0.51234,"power":2.6234,"resistance":10.00,"group0_mah":1234,"group0_mwh":5678,"group1_mah":2345,"group1_mwh":6789,"temperature_sign":0,"temperature":25.0,"dplus_voltage":2.75,"dminus_voltage":2.75}
```

## Library Usage

The toolkit can also be used as a Go library:

```go
package main

import (
    "fmt"
    "github.com/yourusername/tc66c-toolkit/lib/tc66c"
)

func main() {
    device, err := tc66c.NewTC66C("/dev/ttyACM0")
    if err != nil {
        panic(err)
    }
    defer device.Close()

    reading, err := device.GetReading()
    if err != nil {
        panic(err)
    }

    fmt.Printf("Voltage: %.4f V\n", reading.Voltage)
    fmt.Printf("Current: %.5f A\n", reading.Current)
    fmt.Printf("Power: %.4f W\n", reading.Power)
}
```

## Troubleshooting

### Permission Denied on Linux

Add your user to the `dialout` group:

```bash
sudo usermod -a -G dialout $USER
```

Then log out and back in.

Alternatively, run with `sudo` (not recommended for regular use).

### Device Not Found

Check which port your device is on:

```bash
# Linux
ls /dev/ttyACM* /dev/ttyUSB*

# macOS
ls /dev/cu.usbmodem*

# Windows
# Check Device Manager for COM ports
```

Then specify the port:

```bash
tc66c-toolkit get -p /dev/ttyUSB0  # or COM3 on Windows
```

## Protocol

The communication protocol is based on the [sigrok project's TC66C protocol documentation](https://sigrok.org/wiki/RDTech_TC66C). The device uses:

- **Baud rate**: 115200
- **Data bits**: 8
- **Parity**: None
- **Stop bits**: 1

All measurement data is AES-ECB encrypted with a static key.

## License

See [LICENSE.md](LICENSE.md) for details.
