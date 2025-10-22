package tc66c

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Reading represents a single reading from the TC66C device
type Reading struct {
	// pac1 block
	Product         string  // Product name (e.g., "TC66")
	Version         string  // Firmware version (e.g., "1.14")
	SerialNumber    uint32  // Module serial number
	NumRuns         uint32  // Number of runs
	Voltage         float64 // Voltage in V
	Current         float64 // Current in A
	Power           float64 // Power in W

	// pac2 block
	Resistance      float64 // Resistance in Ω
	Group0MAh       uint32  // Group 0 mAh
	Group0MWh       uint32  // Group 0 mWh
	Group1MAh       uint32  // Group 1 mAh
	Group1MWh       uint32  // Group 1 mWh
	TemperatureSign uint32  // Temperature sign (0 = positive, 1 = negative)
	Temperature     float64 // Temperature in °C
	DPlusVoltage    float64 // D+ line voltage in V
	DMinusVoltage   float64 // D- line voltage in V
}

// ParseReading parses a decrypted 192-byte packet into a Reading struct
func ParseReading(data []byte) (*Reading, error) {
	if len(data) != PacketSize {
		return nil, fmt.Errorf("invalid data size: expected %d, got %d", PacketSize, len(data))
	}

	reading := &Reading{}

	// Parse pac1 block (bytes 0-63)
	pac1 := data[0:64]

	// Verify pac1 prefix
	if string(pac1[0:4]) != Block1Prefix {
		return nil, fmt.Errorf("invalid pac1 prefix: expected %s, got %s", Block1Prefix, string(pac1[0:4]))
	}

	// Extract product name (bytes 4-7)
	reading.Product = strings.TrimRight(string(pac1[4:8]), "\x00")

	// Extract version (bytes 8-11)
	reading.Version = strings.TrimRight(string(pac1[8:12]), "\x00")

	// Extract serial number (bytes 12-15)
	reading.SerialNumber = binary.LittleEndian.Uint32(pac1[12:16])

	// Extract number of runs (bytes 44-47)
	reading.NumRuns = binary.LittleEndian.Uint32(pac1[44:48])

	// Extract voltage (bytes 48-51), scale: 1e-4 V
	reading.Voltage = float64(binary.LittleEndian.Uint32(pac1[48:52])) * 1e-4

	// Extract current (bytes 52-55), scale: 1e-5 A
	reading.Current = float64(binary.LittleEndian.Uint32(pac1[52:56])) * 1e-5

	// Extract power (bytes 56-59), scale: 1e-4 W
	reading.Power = float64(binary.LittleEndian.Uint32(pac1[56:60])) * 1e-4

	// Verify pac1 checksum (bytes 60-63)
	pac1Checksum := binary.LittleEndian.Uint16(pac1[60:62])
	if !VerifyChecksum(pac1[0:60], pac1Checksum) {
		return nil, fmt.Errorf("pac1 checksum verification failed")
	}

	// Parse pac2 block (bytes 64-127)
	pac2 := data[64:128]

	// Verify pac2 prefix
	if string(pac2[0:4]) != Block2Prefix {
		return nil, fmt.Errorf("invalid pac2 prefix: expected %s, got %s", Block2Prefix, string(pac2[0:4]))
	}

	// Extract resistance (bytes 4-7), scale: 1e-2 Ω
	reading.Resistance = float64(binary.LittleEndian.Uint32(pac2[4:8])) * 1e-2

	// Extract Group 0 mAh (bytes 8-11)
	reading.Group0MAh = binary.LittleEndian.Uint32(pac2[8:12])

	// Extract Group 0 mWh (bytes 12-15)
	reading.Group0MWh = binary.LittleEndian.Uint32(pac2[12:16])

	// Extract Group 1 mAh (bytes 16-19)
	reading.Group1MAh = binary.LittleEndian.Uint32(pac2[16:20])

	// Extract Group 1 mWh (bytes 20-23)
	reading.Group1MWh = binary.LittleEndian.Uint32(pac2[20:24])

	// Extract temperature sign (bytes 24-27)
	reading.TemperatureSign = binary.LittleEndian.Uint32(pac2[24:28])

	// Extract temperature (bytes 28-31), no scaling mentioned in docs
	tempRaw := binary.LittleEndian.Uint32(pac2[28:32])
	reading.Temperature = float64(tempRaw)
	if reading.TemperatureSign != 0 {
		reading.Temperature = -reading.Temperature
	}

	// Extract D+ voltage (bytes 32-35), scale: 1e-2 V
	reading.DPlusVoltage = float64(binary.LittleEndian.Uint32(pac2[32:36])) * 1e-2

	// Extract D- voltage (bytes 36-39), scale: 1e-2 V
	reading.DMinusVoltage = float64(binary.LittleEndian.Uint32(pac2[36:40])) * 1e-2

	// Verify pac2 checksum (bytes 60-63)
	pac2Checksum := binary.LittleEndian.Uint16(pac2[60:62])
	if !VerifyChecksum(pac2[0:60], pac2Checksum) {
		return nil, fmt.Errorf("pac2 checksum verification failed")
	}

	// Parse pac3 block (bytes 128-191)
	pac3 := data[128:192]

	// Verify pac3 prefix
	if string(pac3[0:4]) != Block3Prefix {
		return nil, fmt.Errorf("invalid pac3 prefix: expected %s, got %s", Block3Prefix, string(pac3[0:4]))
	}

	// Verify pac3 checksum (bytes 60-63)
	pac3Checksum := binary.LittleEndian.Uint16(pac3[60:62])
	if !VerifyChecksum(pac3[0:60], pac3Checksum) {
		return nil, fmt.Errorf("pac3 checksum verification failed")
	}

	return reading, nil
}

// String returns a formatted string representation of the reading
func (r *Reading) String() string {
	return fmt.Sprintf(`Product: %s
Version: %s
Serial: %d
Runs: %d
Voltage: %.4f V
Current: %.5f A
Power: %.4f W
Resistance: %.2f Ω
Group 0: %d mAh / %d mWh
Group 1: %d mAh / %d mWh
Temperature: %.1f °C
D+ Voltage: %.2f V
D- Voltage: %.2f V`,
		r.Product, r.Version, r.SerialNumber, r.NumRuns,
		r.Voltage, r.Current, r.Power, r.Resistance,
		r.Group0MAh, r.Group0MWh,
		r.Group1MAh, r.Group1MWh,
		r.Temperature,
		r.DPlusVoltage, r.DMinusVoltage)
}

// ShortString returns a compact one-line representation of the reading
func (r *Reading) ShortString() string {
	return fmt.Sprintf("V: %.4fV | I: %.5fA | P: %.4fW | R: %.2fΩ | T: %.1f°C | D+: %.2fV | D-: %.2fV",
		r.Voltage,
		r.Current,
		r.Power,
		r.Resistance,
		r.Temperature,
		r.DPlusVoltage,
		r.DMinusVoltage,
	)
}

// RecordingEntry represents a single recording entry from the device
type RecordingEntry struct {
	Voltage float64 // Voltage in V
	Current float64 // Current in A
}

// String returns a formatted string representation of a recording entry
func (re *RecordingEntry) String() string {
	return fmt.Sprintf("V: %.4f V, I: %.5f A", re.Voltage, re.Current)
}
