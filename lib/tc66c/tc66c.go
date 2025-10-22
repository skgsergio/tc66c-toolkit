package tc66c

import (
	"encoding/binary"
	"fmt"
	"time"

	"go.bug.st/serial"
)

const (
	// Packet sizes
	BlockSize  = 64
	NumBlocks  = 3
	PacketSize = BlockSize * NumBlocks // 192 bytes

	// Block prefixes
	Block1Prefix = "pac1"
	Block2Prefix = "pac2"
	Block3Prefix = "pac3"
)

// Commands supported in normal mode
const (
	CmdQuery  = "query"  // Check device mode (4-byte response)
	CmdGetVA  = "getva"  // Poll readings (192-byte response)
	CmdGetRec = "gtrec"  // Retrieve recordings (variable length)
	CmdLastP  = "lastp"  // Previous page (0-length)
	CmdNextP  = "nextp"  // Next page (0-length)
	CmdRotat  = "rotat"  // Rotate screen (0-length)
)

// Commands supported in bootloader mode
const (
	CmdUpdate = "update" // Enter firmware update mode (5-byte response "uprdy")
)

// Firmware update constants
const (
	FirmwareChunkSize = 64      // Size of each firmware chunk
	UpdateModeResponse = "uprdy" // Expected response when entering update mode
	ChunkOKResponse = "OK"       // Expected response after each chunk
)

// AES-ECB encryption key (static 32-byte key from protocol documentation)
var AESKey = []byte{
	0x58, 0x21, 0xfa, 0x56, 0x01, 0xb2, 0xf0, 0x26,
	0x87, 0xff, 0x12, 0x04, 0x62, 0x2a, 0x4f, 0xb0,
	0x86, 0xf4, 0x02, 0x60, 0x81, 0x6f, 0x9a, 0x0b,
	0xa7, 0xf1, 0x06, 0x61, 0x9a, 0xb8, 0x72, 0x88,
}

// DeviceMode represents the operational mode of the TC66C device
type DeviceMode int

const (
	ModeFirmware DeviceMode = iota // Normal firmware mode
	ModeBootloader                 // Bootloader mode
	ModeUnknown                    // Unknown mode
)

// String returns a string representation of the device mode
func (m DeviceMode) String() string {
	switch m {
	case ModeFirmware:
		return "firmware"
	case ModeBootloader:
		return "bootloader"
	default:
		return "unknown"
	}
}

// TC66C represents a connection to a TC66C device
type TC66C struct {
	port serial.Port
	Mode DeviceMode // Current device mode (firmware/bootloader)
}

// NewTC66C creates a new TC66C device connection
func NewTC66C(portName string) (*TC66C, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port %s: %w", portName, err)
	}

	// Set read timeout
	err = port.SetReadTimeout(2 * time.Second)
	if err != nil {
		port.Close()
		return nil, fmt.Errorf("failed to set read timeout: %w", err)
	}

	tc := &TC66C{
		port: port,
		Mode: ModeUnknown,
	}

	// Query device mode
	deviceMode, err := tc.queryDeviceMode()
	if err != nil {
		port.Close()
		return nil, fmt.Errorf("failed to query device mode: %w", err)
	}
	tc.Mode = deviceMode

	return tc, nil
}

// queryDeviceMode queries the device to determine if it's in firmware or bootloader mode
func (tc *TC66C) queryDeviceMode() (DeviceMode, error) {
	response, err := tc.Query()
	if err != nil {
		return ModeUnknown, err
	}

	if len(response) != 4 {
		return ModeUnknown, fmt.Errorf("invalid query response length: expected 4, got %d", len(response))
	}

	// Parse the response to determine mode
	// Based on protocol documentation:
	// - "firm" (0x6669726D) indicates firmware mode
	// - "boot" (0x626F6F74) indicates bootloader mode
	responseStr := string(response)

	switch responseStr {
	case "firm":
		return ModeFirmware, nil
	case "boot":
		return ModeBootloader, nil
	default:
		return ModeUnknown, fmt.Errorf("unknown device mode response: %q", responseStr)
	}
}

// Close closes the serial port connection
func (tc *TC66C) Close() error {
	if tc.port != nil {
		return tc.port.Close()
	}
	return nil
}

// flushBuffer drains any pending data from the serial port
func (tc *TC66C) flushBuffer() {
	// Set a very short timeout to quickly drain the buffer
	tc.port.SetReadTimeout(10 * time.Millisecond)
	buf := make([]byte, 1024)
	for {
		n, _ := tc.port.Read(buf)
		if n == 0 {
			break
		}
	}
	// Restore normal timeout
	tc.port.SetReadTimeout(2 * time.Second)
}

// sendCommand sends a command to the device
func (tc *TC66C) sendCommand(cmd string) error {
	// Flush any pending data before sending command
	tc.flushBuffer()

	// Commands are sent as plain text followed by \r\n
	_, err := tc.port.Write([]byte(cmd + "\r\n"))
	if err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	// Small delay to let the device process the command
	time.Sleep(50 * time.Millisecond)

	return nil
}

// readResponse reads a response of the specified size from the device
func (tc *TC66C) readResponse(size int) ([]byte, error) {
	buffer := make([]byte, size)
	n := 0

	// Read until we have all the expected bytes
	for n < size {
		bytesRead, err := tc.port.Read(buffer[n:])
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		if bytesRead == 0 {
			return nil, fmt.Errorf("timeout reading response (got %d of %d bytes)", n, size)
		}
		n += bytesRead
	}

	return buffer, nil
}

// Query sends the 'query' command to check device mode
func (tc *TC66C) Query() ([]byte, error) {
	err := tc.sendCommand(CmdQuery)
	if err != nil {
		return nil, err
	}

	return tc.readResponse(4)
}

// GetReading sends the 'getva' command and returns a parsed Reading
func (tc *TC66C) GetReading() (*Reading, error) {
	if tc.Mode != ModeFirmware {
		return nil, fmt.Errorf("device must be in firmware mode (current mode: %s)", tc.Mode)
	}

	err := tc.sendCommand(CmdGetVA)
	if err != nil {
		return nil, err
	}

	// Read the 192-byte encrypted response
	encrypted, err := tc.readResponse(PacketSize)
	if err != nil {
		return nil, err
	}

	// Decrypt the packet
	decrypted, err := DecryptPacket(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt packet: %w", err)
	}

	// Parse the decrypted data
	reading, err := ParseReading(decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reading: %w", err)
	}

	return reading, nil
}

// GetRecordings sends the 'gtrec' command to retrieve recordings
// Returns a slice of RecordingEntry structs containing voltage and current pairs
func (tc *TC66C) GetRecordings() ([]*RecordingEntry, error) {
	if tc.Mode != ModeFirmware {
		return nil, fmt.Errorf("device must be in firmware mode (current mode: %s)", tc.Mode)
	}

	err := tc.sendCommand(CmdGetRec)
	if err != nil {
		return nil, err
	}

	results := make([]*RecordingEntry, 0)
	buffer := make([]byte, 0)
	chunk := make([]byte, 8)

	// Read 8-byte chunks until we get 0 bytes (timeout/end of data)
	for {
		n, err := tc.port.Read(chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to read recording chunk: %w", err)
		}
		if n == 0 {
			break
		}

		// Append the chunk to our buffer
		buffer = append(buffer, chunk[:n]...)

		// Process complete 8-byte records from the buffer
		for len(buffer) >= 8 {
			// Extract two uint32 values (little-endian)
			voltageRaw := binary.LittleEndian.Uint32(buffer[0:4])
			currentRaw := binary.LittleEndian.Uint32(buffer[4:8])

			// Convert to float with proper scaling
			// Voltage: divide by 1000, then by 10 = /10000
			voltage := float64(voltageRaw) / 1000.0 / 10.0
			// Current: divide by 1000, then by 100 = /100000
			current := float64(currentRaw) / 1000.0 / 100.0

			results = append(results, &RecordingEntry{
				Voltage: voltage,
				Current: current,
			})

			// Remove processed bytes from buffer
			buffer = buffer[8:]
		}
	}

	return results, nil
}

// PreviousPage sends the 'lastp' command to go to the previous page
func (tc *TC66C) PreviousPage() error {
	if tc.Mode != ModeFirmware {
		return fmt.Errorf("device must be in firmware mode (current mode: %s)", tc.Mode)
	}
	return tc.sendCommand(CmdLastP)
}

// NextPage sends the 'nextp' command to go to the next page
func (tc *TC66C) NextPage() error {
	if tc.Mode != ModeFirmware {
		return fmt.Errorf("device must be in firmware mode (current mode: %s)", tc.Mode)
	}
	return tc.sendCommand(CmdNextP)
}

// RotateScreen sends the 'rotat' command to rotate the screen
func (tc *TC66C) RotateScreen() error {
	if tc.Mode != ModeFirmware {
		return fmt.Errorf("device must be in firmware mode (current mode: %s)", tc.Mode)
	}
	return tc.sendCommand(CmdRotat)
}

// FirmwareUpdateProgress represents the progress of a firmware update
type FirmwareUpdateProgress struct {
	BytesSent  int
	TotalBytes int
	ChunksSent int
	TotalChunks int
}

// UpdateFirmware updates the device firmware from the provided file
// The device must be in bootloader mode before calling this function
// progressCallback is called after each chunk is sent (can be nil)
func (tc *TC66C) UpdateFirmware(firmwareData []byte, progressCallback func(FirmwareUpdateProgress)) error {
	// Safety check: device must be in bootloader mode
	if tc.Mode != ModeBootloader {
		return fmt.Errorf("device must be in bootloader mode to update firmware (current mode: %s)", tc.Mode)
	}

	// Calculate file size and chunk count
	fileSize := len(firmwareData)
	if fileSize == 0 {
		return fmt.Errorf("firmware data is empty")
	}

	chunkCount := (fileSize + FirmwareChunkSize - 1) / FirmwareChunkSize

	// Enter firmware update mode
	err := tc.sendCommand(CmdUpdate)
	if err != nil {
		return fmt.Errorf("failed to send update command: %w", err)
	}

	// Read the "uprdy" response (5 bytes)
	response, err := tc.readResponse(5)
	if err != nil {
		return fmt.Errorf("failed to read update mode response: %w", err)
	}

	if string(response) != UpdateModeResponse {
		return fmt.Errorf("device replied with '%s', expected '%s'", string(response), UpdateModeResponse)
	}

	// Send firmware in chunks
	bytesSent := 0
	chunksSent := 0

	for bytesSent < fileSize {
		// Calculate chunk size (last chunk may be smaller)
		chunkEnd := bytesSent + FirmwareChunkSize
		if chunkEnd > fileSize {
			chunkEnd = fileSize
		}
		chunk := firmwareData[bytesSent:chunkEnd]

		// Send chunk
		_, err := tc.port.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", chunksSent+1, err)
		}

		// Wait for "OK" response (2 bytes)
		chunkResponse, err := tc.readResponse(2)
		if err != nil {
			return fmt.Errorf("failed to read response for chunk %d: %w", chunksSent+1, err)
		}

		if string(chunkResponse) != ChunkOKResponse {
			return fmt.Errorf("device replied with '%s' for chunk %d, expected '%s'. Device may not boot normally, try again",
				string(chunkResponse), chunksSent+1, ChunkOKResponse)
		}

		// Update progress
		bytesSent += len(chunk)
		chunksSent++

		// Call progress callback if provided
		if progressCallback != nil {
			progressCallback(FirmwareUpdateProgress{
				BytesSent:   bytesSent,
				TotalBytes:  fileSize,
				ChunksSent:  chunksSent,
				TotalChunks: chunkCount,
			})
		}
	}

	return nil
}
